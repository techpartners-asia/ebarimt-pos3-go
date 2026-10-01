package postest_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/posapi"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/postest"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/receipt"
)

var ctx = context.Background()

func asError(t *testing.T, err error) *ebarimt.Error {
	t.Helper()
	var e *ebarimt.Error
	require.True(t, errors.As(err, &e), "want *ebarimt.Error, got %T: %v", err, err)
	return e
}

func sale(t *testing.T, mutate func(*receipt.Sale), m receipt.Merchant) posapi.ReceiptRequest {
	t.Helper()
	s := receipt.Sale{
		Type: receipt.B2CReceipt, PosNo: "10000001", BranchNo: "001", DistrictCode: "2501",
		Items: []receipt.Item{{Name: "Cola", ClassificationCode: "2349010", MeasureUnit: "ш", TaxType: receipt.TaxVATable,
			HasCityTax: true, Quantity: decimal.NewFromInt(1), Total: money.FromTugrik(2800)}},
		Payments: []receipt.Payment{{Code: receipt.PaymentCash, Amount: money.FromTugrik(2800)}},
	}
	if mutate != nil {
		mutate(&s)
	}
	if m.TIN == "" {
		m = receipt.Merchant{TIN: "37900846788", VATPayer: true, CityTaxPayer: true}
	}
	req, err := receipt.Build(s, m)
	require.NoError(t, err)
	return req
}

func TestIssueReturnThroughBuild(t *testing.T) {
	pos := postest.New(t)
	c := pos.Client()

	issued, err := c.IssueReceipt(ctx, sale(t, nil, receipt.Merchant{}))
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^\d{33}$`), issued.ID)
	assert.NotEmpty(t, issued.Lottery)
	assert.NotEmpty(t, issued.QRData)
	assert.Equal(t, "2800", issued.TotalAmount.String())
	assert.Equal(t, "250", issued.TotalVAT.String())
	assert.Equal(t, "50", issued.TotalCityTax.String())
	require.Len(t, issued.Receipts, 1)
	assert.NotEqual(t, issued.ID, issued.Receipts[0].ID)
	assert.False(t, issued.Date.IsZero())

	require.NoError(t, c.ReturnReceipt(ctx, issued.ID, issued.Date))
	e := asError(t, c.ReturnReceipt(ctx, issued.ID, issued.Date))
	assert.Equal(t, ebarimt.Rejected, e.Kind, "returning twice is refused")

	reqs := pos.Requests()
	require.Len(t, reqs, 3)
	assert.Equal(t, http.MethodPost, reqs[0].Method)
	assert.Equal(t, http.MethodDelete, reqs[1].Method)
	assert.Contains(t, string(reqs[0].Body), `"totalAmount":2800`)

	other, err := c.IssueReceipt(ctx, sale(t, nil, receipt.Merchant{}))
	require.NoError(t, err)
	assert.NotEqual(t, issued.ID, other.ID)
}

func TestB2BGetsNoLottery(t *testing.T) {
	pos := postest.New(t)
	req := sale(t, func(s *receipt.Sale) { s.Type = receipt.B2BReceipt; s.Buyer.TIN = "61200064714" }, receipt.Merchant{})
	issued, err := pos.Client().IssueReceipt(ctx, req)
	require.NoError(t, err)
	assert.Empty(t, issued.Lottery)
}

func TestEnforcesTotalsExactly(t *testing.T) {
	pos := postest.New(t)
	c := pos.Client()
	for name, mutate := range map[string]func(*posapi.ReceiptRequest){
		"header total off":    func(r *posapi.ReceiptRequest) { r.TotalAmount = r.TotalAmount.Add(money.MustParse("0.01")) },
		"sub-receipt off":     func(r *posapi.ReceiptRequest) { r.Receipts[0].TotalAmount = money.FromTugrik(1) },
		"header vat off":      func(r *posapi.ReceiptRequest) { r.TotalVat = money.FromTugrik(1) },
		"header city off":     func(r *posapi.ReceiptRequest) { r.TotalCityTax = money.FromTugrik(1) },
		"payments don't sum":  func(r *posapi.ReceiptRequest) { r.Payments[0].PaidAmount = money.FromTugrik(2799) },
		"no sub-receipts":     func(r *posapi.ReceiptRequest) { r.Receipts = nil },
		"no merchant":         func(r *posapi.ReceiptRequest) { r.MerchantTin = "" },
		"item total mismatch": func(r *posapi.ReceiptRequest) { r.Receipts[0].Items[0].TotalAmount = money.FromTugrik(2801) },
	} {
		t.Run(name, func(t *testing.T) {
			req := sale(t, nil, receipt.Merchant{})
			mutate(&req)
			_, err := c.IssueReceipt(ctx, req)
			e := asError(t, err)
			assert.Equal(t, ebarimt.Rejected, e.Kind)
			assert.Equal(t, 400, e.HTTPStatus)
		})
	}
}

func TestEnforcesClassificationCodeAndRelief(t *testing.T) {
	c := postest.New(t).Client()

	req := sale(t, nil, receipt.Merchant{})
	req.Receipts[0].Items[0].ClassificationCode = ""
	_, err := c.IssueReceipt(ctx, req)
	assert.Equal(t, "classificationCode талбарын утга хоосон", asError(t, err).Message, "the live daemon's own words")

	req = sale(t, nil, receipt.Merchant{})
	req.Receipts[0].Items[0].ClassificationCode = "2349"
	_, err = c.IssueReceipt(ctx, req)
	assert.Equal(t, ebarimt.Rejected, asError(t, err).Kind)

	req = sale(t, func(s *receipt.Sale) {
		s.Items[0].TaxType, s.Items[0].TaxReliefCode = receipt.TaxVATExempt, "320"
	}, receipt.Merchant{})
	req.Receipts[0].Items[0].TaxProductCode = ""
	_, err = c.IssueReceipt(ctx, req)
	assert.Equal(t, ebarimt.Rejected, asError(t, err).Kind)
}

func TestEnforcesCustomerTinOnB2B(t *testing.T) {
	c := postest.New(t).Client()
	req := sale(t, func(s *receipt.Sale) { s.Type = receipt.B2BReceipt; s.Buyer.TIN = "61200064714" }, receipt.Merchant{})
	req.CustomerTin = ""
	_, err := c.IssueReceipt(ctx, req)
	assert.Equal(t, "customerTin талбарын утга хоосон", asError(t, err).Message)
}

// The famous mistake: city tax taken from the gross total (2800 * 2% = 56)
// instead of from the base. PosAPI answers with the expected figure.
func TestCityTaxCheckUsesPosAPIsOwnMessage(t *testing.T) {
	c := postest.New(t).Client()
	req := sale(t, nil, receipt.Merchant{})
	item := &req.Receipts[0].Items[0]
	item.TotalCityTax = money.FromTugrik(56)
	req.Receipts[0].TotalCityTax, req.TotalCityTax = item.TotalCityTax, item.TotalCityTax

	_, err := c.IssueReceipt(ctx, req)
	e := asError(t, err)
	assert.Equal(t, ebarimt.Rejected, e.Kind)
	assert.Equal(t, "НХАТ-г 2%-р тооцоолоогүй байна. Шалгуур дүн: 50", e.Message)

	// City tax with no VAT: 2800 / 1.02 * 2% = 54.90.
	req = sale(t, func(s *receipt.Sale) {
		s.Items[0].TaxType, s.Items[0].TaxReliefCode = receipt.TaxVATExempt, "320"
	}, receipt.Merchant{})
	item = &req.Receipts[0].Items[0]
	item.TotalCityTax = money.FromTugrik(50)
	req.Receipts[0].TotalCityTax, req.TotalCityTax = item.TotalCityTax, item.TotalCityTax
	_, err = c.IssueReceipt(ctx, req)
	assert.Equal(t, "НХАТ-г 2%-р тооцоолоогүй байна. Шалгуур дүн: 54.9", asError(t, err).Message)
}

func TestVATCheck(t *testing.T) {
	c := postest.New(t).Client()
	req := sale(t, nil, receipt.Merchant{TIN: "37900846788", VATPayer: true})
	req.Receipts[0].Items[0].TotalVat = money.FromTugrik(280)
	req.Receipts[0].TotalVat, req.TotalVat = money.FromTugrik(280), money.FromTugrik(280)
	_, err := c.IssueReceipt(ctx, req)
	assert.Contains(t, asError(t, err).Message, "Шалгуур дүн: 254.55")
}

func TestUnroundedVATFromTheOldClientIsStillAccepted(t *testing.T) {
	c := postest.New(t).Client()
	req := sale(t, nil, receipt.Merchant{TIN: "37900846788", VATPayer: true})
	// 254.54545454545453: what the reference client sends.
	v := money.MustParse("254.55")
	req.Receipts[0].Items[0].TotalVat, req.Receipts[0].TotalVat, req.TotalVat = v, v, v
	_, err := c.IssueReceipt(ctx, req)
	require.NoError(t, err)
}

func TestDeleteMustCarryABody(t *testing.T) {
	pos := postest.New(t)
	raw := func(body string) (int, string) {
		r, _ := http.NewRequest(http.MethodDelete, pos.URL+"/rest/receipt", strings.NewReader(body))
		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		defer resp.Body.Close()
		var d struct{ Message string }
		_ = json.NewDecoder(resp.Body).Decode(&d)
		return resp.StatusCode, d.Message
	}
	code, _ := raw("")
	assert.Equal(t, 400, code, "a DELETE without a body is the v1 bug (#36)")
	code, _ = raw(`{"id":"123456789012345678901234567890123","date":"2026-09-30 10:11:12"}`)
	assert.Equal(t, 400, code, "an id the fake never issued")
	code, _ = raw(`{"id":"x","date":"yesterday"}`)
	assert.Equal(t, 400, code)
}

func TestOtherEndpointsAndFailNext(t *testing.T) {
	pos := postest.New(t)
	c := pos.Client()

	require.NoError(t, c.SendToTaxAuthority(ctx))
	st, err := c.Status(ctx)
	require.NoError(t, err)
	assert.Positive(t, st.LotteriesLeft)
	assert.WithinDuration(t, time.Now(), st.LastSentAt, time.Minute)
	accounts, err := c.BankAccounts(ctx, "37900846788")
	require.NoError(t, err)
	assert.Empty(t, accounts)

	pos.FailNext(503, "maintenance")
	e := asError(t, c.SendToTaxAuthority(ctx))
	assert.Equal(t, 503, e.HTTPStatus)
	assert.Equal(t, "maintenance", e.Message)
	require.NoError(t, c.SendToTaxAuthority(ctx), "only the next request fails")

	r, err := http.Get(pos.URL + "/nope")
	require.NoError(t, err)
	r.Body.Close()
	assert.Equal(t, 404, r.StatusCode)
}
