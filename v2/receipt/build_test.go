package receipt_test

import (
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/posapi"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/receipt"
)

const merchantTIN = "37900846788"

func tugrik(n int64) money.Amount  { return money.FromTugrik(n) }
func qty(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func item(name string, total money.Amount, tt receipt.TaxType) receipt.Item {
	it := receipt.Item{
		Name: name, ClassificationCode: "2349010", MeasureUnit: "ширхэг",
		TaxType: tt, Quantity: qty("1"), Total: total,
	}
	if tt != receipt.TaxVATable {
		it.TaxReliefCode = "3"
	}
	return it
}

func invalid(t *testing.T, err error) *ebarimt.Error {
	t.Helper()
	var e *ebarimt.Error
	require.True(t, errors.As(err, &e), "want *ebarimt.Error, got %T: %v", err, err)
	require.Equal(t, ebarimt.Invalid, e.Kind)
	return e
}

func fields(e *ebarimt.Error) []string {
	var out []string
	for _, v := range e.Violations {
		out = append(out, v.Field)
	}
	return out
}

// The goldens are the four production-proven fixtures of the reference client,
// rebuilt through receipt.Build. Regenerate only deliberately:
//
//	UPDATE_GOLDEN=1 go test ./receipt -run Golden
func TestBuild_Golden(t *testing.T) {
	both := receipt.Merchant{TIN: merchantTIN, VATPayer: true, CityTaxPayer: true}
	neither := receipt.Merchant{TIN: merchantTIN}
	base := receipt.Sale{PosNo: "10000001", BranchNo: "7", DistrictCode: "3420"}
	cola := receipt.Item{Name: "Cola 0.5l", ClassificationCode: "2349010", Quantity: qty("1"), HasCityTax: true,
		MeasureUnit: "ширхэг", Total: tugrik(2800), TaxType: receipt.TaxVATable}

	tests := []struct {
		file string
		sale func() receipt.Sale
		m    receipt.Merchant
	}{
		{"receipt_a_b2c_city_tax.golden.json", func() receipt.Sale {
			s := base
			s.Type = receipt.B2CReceipt
			s.Items = []receipt.Item{cola}
			s.Payments = []receipt.Payment{{Code: receipt.PaymentCard, Amount: tugrik(2800)}}
			return s
		}, both},
		{"receipt_d_b2c_qpay.golden.json", func() receipt.Sale {
			s := base
			s.Type = receipt.B2CReceipt
			s.Items = []receipt.Item{cola}
			s.Payments = []receipt.Payment{{Code: receipt.PaymentQPay, Amount: tugrik(2800)}}
			return s
		}, both},
		{"receipt_b_b2c_mixed.golden.json", func() receipt.Sale {
			s := base
			s.Type = receipt.B2CReceipt
			s.Items = []receipt.Item{
				{Name: "Chips", ClassificationCode: "2349010", Quantity: qty("1"), MeasureUnit: "ширхэг", Total: tugrik(3500), TaxType: receipt.TaxVATable},
				{Name: "Water", ClassificationCode: "2349010", TaxReliefCode: "3", Quantity: qty("1"), MeasureUnit: "ширхэг", Total: tugrik(3300), TaxType: receipt.TaxVATZeroRated},
			}
			s.Payments = []receipt.Payment{{Code: receipt.PaymentCard, Amount: tugrik(6800)}}
			return s
		}, neither},
		{"receipt_c_b2b_coupon_split.golden.json", func() receipt.Sale {
			s := base
			s.Type = receipt.B2BReceipt
			s.Buyer = receipt.Buyer{TIN: "61200064714"}
			s.ReportMonth = &receipt.Month{Year: 2026, Month: 9}
			s.Items = []receipt.Item{
				{Name: "a", ClassificationCode: "2349010", Quantity: qty("1"), MeasureUnit: "ширхэг", Total: money.MustParse("457.40"), TaxType: receipt.TaxVATable},
				{Name: "b", ClassificationCode: "2349010", Quantity: qty("1"), MeasureUnit: "ширхэг", Total: money.MustParse("1114.92"), TaxType: receipt.TaxVATable},
			}
			s.Payments = []receipt.Payment{{Code: receipt.PaymentCard, Amount: money.MustParse("1572.32")}}
			return s
		}, receipt.Merchant{TIN: merchantTIN, VATPayer: true}},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			req, err := receipt.Build(tt.sale(), tt.m)
			require.NoError(t, err)
			got, err := json.Marshal(req)
			require.NoError(t, err)

			path := filepath.Join("testdata", tt.file)
			if os.Getenv("UPDATE_GOLDEN") != "" {
				require.NoError(t, os.WriteFile(path, got, 0o644))
			}
			want, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, string(want), string(got))
		})
	}
}

func TestBuild_TotalsAreExactSums(t *testing.T) {
	s := receipt.Sale{Type: receipt.B2CReceipt, Items: []receipt.Item{
		item("a", money.MustParse("457.40"), receipt.TaxVATable),
		item("b", money.MustParse("1114.92"), receipt.TaxVATable),
	}, Payments: []receipt.Payment{{Code: receipt.PaymentCard, Amount: money.MustParse("1572.32")}}}
	req, err := receipt.Build(s, receipt.Merchant{TIN: merchantTIN, VATPayer: true})
	require.NoError(t, err)
	assert.Equal(t, "1572.32", req.TotalAmount.String())
	// 41.58 + 101.36, each item rounded to 2 places before summing.
	assert.Equal(t, "142.94", req.TotalVat.String())
	assert.Equal(t, "142.94", req.Receipts[0].TotalVat.String())
	assert.Equal(t, "41.58", req.Receipts[0].Items[0].TotalVat.String())
	assert.Equal(t, "101.36", req.Receipts[0].Items[1].TotalVat.String())
}

func TestBuild_OneSubReceiptPerTaxTypeSorted(t *testing.T) {
	s := receipt.Sale{Type: receipt.B2CReceipt, Items: []receipt.Item{
		item("zero", tugrik(1000), receipt.TaxVATZeroRated),
		item("vat", tugrik(2000), receipt.TaxVATable),
		item("outside", tugrik(500), receipt.TaxOutsideVAT),
		item("vat2", tugrik(300), receipt.TaxVATable),
		item("free", tugrik(700), receipt.TaxVATExempt),
	}, Payments: []receipt.Payment{{Code: receipt.PaymentCash, Amount: tugrik(4500)}}}
	for i := 0; i < 20; i++ { // map iteration is random; the order must not be
		req, err := receipt.Build(s, receipt.Merchant{TIN: merchantTIN, VATPayer: true})
		require.NoError(t, err)
		var order []string
		for _, r := range req.Receipts {
			order = append(order, string(r.TaxType))
		}
		require.Equal(t, []string{"NOT_VAT", "VAT_ABLE", "VAT_FREE", "VAT_ZERO"}, order)
		assert.Len(t, req.Receipts[1].Items, 2)
	}
}

func TestBuild_NotVATOnlySaleIsOneRequestWithItsItems(t *testing.T) {
	s := receipt.Sale{Type: receipt.B2CReceipt,
		Items:    []receipt.Item{item("x", tugrik(500), receipt.TaxOutsideVAT)},
		Payments: []receipt.Payment{{Code: receipt.PaymentCash, Amount: tugrik(500)}}}
	req, err := receipt.Build(s, receipt.Merchant{TIN: merchantTIN, VATPayer: true, CityTaxPayer: true})
	require.NoError(t, err)
	require.Len(t, req.Receipts, 1)
	assert.Equal(t, "500", req.TotalAmount.String())
	assert.Equal(t, posapi.TaxOutsideVAT, req.Receipts[0].TaxType)
}

func TestBuild_UnitPriceTruncates(t *testing.T) {
	for _, tt := range []struct{ total, qty, want string }{
		{"0.87", "3", "0.29"},       // float64 int(v*100)/100 gives 0.28
		{"2", "3", "0.66"},          // rounding would say 0.67
		{"1000", "0.355", "2816.9"}, // 2816.9014..
		{"2800", "1", "2800"},
	} {
		s := receipt.Sale{Type: receipt.B2CReceipt, Payments: []receipt.Payment{{Code: receipt.PaymentCash, Amount: money.MustParse(tt.total)}}}
		it := item("x", money.MustParse(tt.total), receipt.TaxVATable)
		it.Quantity = qty(tt.qty)
		s.Items = []receipt.Item{it}
		req, err := receipt.Build(s, receipt.Merchant{TIN: merchantTIN})
		require.NoError(t, err)
		assert.Equal(t, tt.want, req.Receipts[0].Items[0].UnitPrice.String(), "%s / %s", tt.total, tt.qty)
	}
}

func TestBuild_ReportsEveryViolationAtOnce(t *testing.T) {
	s := receipt.Sale{
		Type:  receipt.B2CReceipt,
		Buyer: receipt.Buyer{TIN: "123"},
		Items: []receipt.Item{
			{Name: "ok", ClassificationCode: "2349010", TaxType: receipt.TaxVATable, Quantity: qty("1"), Total: tugrik(100)},
			{Name: "bad", ClassificationCode: "234", TaxType: receipt.TaxVATable, TaxReliefCode: "3", Quantity: qty("0"), Total: tugrik(0)},
			{Name: "nocode", TaxType: receipt.TaxVATExempt, Quantity: qty("1"), Total: tugrik(50)},
		},
		Payments: []receipt.Payment{{Code: receipt.PaymentCash, Amount: tugrik(1)}},
	}
	_, err := receipt.Build(s, receipt.Merchant{TIN: merchantTIN})
	e := invalid(t, err)
	assert.ElementsMatch(t, []string{
		"buyer.tin",
		"items[1].quantity", "items[1].total", "items[1].classificationCode", "items[1].taxReliefCode",
		"items[2].classificationCode", "items[2].taxReliefCode",
		"payments",
	}, fields(e))
	assert.Contains(t, e.Error(), "items[2].classificationCode")
}

func TestBuild_Rules(t *testing.T) {
	m := receipt.Merchant{TIN: merchantTIN}
	pay := func(n int64) []receipt.Payment {
		return []receipt.Payment{{Code: receipt.PaymentCash, Amount: tugrik(n)}}
	}
	ok := func() receipt.Sale {
		return receipt.Sale{Type: receipt.B2CReceipt, Items: []receipt.Item{item("a", tugrik(100), receipt.TaxVATable)}, Payments: pay(100)}
	}
	month := &receipt.Month{Year: 2026, Month: 9}

	tests := []struct {
		name   string
		mutate func(*receipt.Sale)
		want   []string // violating fields; nil = valid
	}{
		{"valid baseline", func(*receipt.Sale) {}, nil},
		{"no items", func(s *receipt.Sale) { s.Items = nil }, []string{"items"}},
		{"qty zero", func(s *receipt.Sale) { s.Items[0].Quantity = qty("0") }, []string{"items[0].quantity"}},
		{"qty negative", func(s *receipt.Sale) { s.Items[0].Quantity = qty("-1") }, []string{"items[0].quantity"}},
		{"total zero", func(s *receipt.Sale) { s.Items[0].Total = tugrik(0) }, []string{"items[0].total", "payments"}},
		{"classification 6 digits", func(s *receipt.Sale) { s.Items[0].ClassificationCode = "234901" }, []string{"items[0].classificationCode"}},
		{"classification letters", func(s *receipt.Sale) { s.Items[0].ClassificationCode = "23490x1" }, []string{"items[0].classificationCode"}},
		{"classification empty", func(s *receipt.Sale) { s.Items[0].ClassificationCode = "" }, []string{"items[0].classificationCode"}},
		{"VAT_FREE needs relief", func(s *receipt.Sale) { s.Items[0].TaxType = receipt.TaxVATExempt }, []string{"items[0].taxReliefCode"}},
		{"VAT_ZERO needs relief", func(s *receipt.Sale) { s.Items[0].TaxType = receipt.TaxVATZeroRated }, []string{"items[0].taxReliefCode"}},
		{"NOT_VAT needs relief", func(s *receipt.Sale) { s.Items[0].TaxType = receipt.TaxOutsideVAT }, []string{"items[0].taxReliefCode"}},
		{"VAT_FREE with relief", func(s *receipt.Sale) { s.Items[0].TaxType = receipt.TaxVATExempt; s.Items[0].TaxReliefCode = "320" }, nil},
		{"VAT_ABLE with relief", func(s *receipt.Sale) { s.Items[0].TaxReliefCode = "320" }, []string{"items[0].taxReliefCode"}},
		{"unknown tax type", func(s *receipt.Sale) { s.Items[0].TaxType = "VAT" }, []string{"items[0].taxType"}},
		{"no payments", func(s *receipt.Sale) { s.Payments = nil }, []string{"payments"}},
		{"payments short", func(s *receipt.Sale) { s.Payments = pay(99) }, []string{"payments"}},
		{"payments over", func(s *receipt.Sale) { s.Payments = pay(101) }, []string{"payments"}},
		{"payments split ok", func(s *receipt.Sale) {
			s.Payments = []receipt.Payment{{Code: receipt.PaymentCash, Amount: tugrik(60)}, {Code: receipt.PaymentCard, Amount: tugrik(40)}}
		}, nil},
		{"B2B needs buyer tin", func(s *receipt.Sale) { s.Type = receipt.B2BReceipt }, []string{"buyer.tin"}},
		{"B2B invoice needs buyer tin", func(s *receipt.Sale) { s.Type = receipt.B2BInvoice }, []string{"buyer.tin"}},
		{"B2B with tin", func(s *receipt.Sale) { s.Type = receipt.B2BReceipt; s.Buyer.TIN = "61200064714" }, nil},
		{"B2C with tin", func(s *receipt.Sale) { s.Buyer.TIN = "61200064714" }, []string{"buyer.tin"}},
		{"B2C invoice with tin", func(s *receipt.Sale) { s.Type = receipt.B2CInvoice; s.Buyer.TIN = "61200064714" }, []string{"buyer.tin"}},
		{"consumer no on B2C receipt", func(s *receipt.Sale) { s.Buyer.ConsumerNo = "10038071" }, nil},
		{"consumer no on B2B", func(s *receipt.Sale) {
			s.Type = receipt.B2BReceipt
			s.Buyer = receipt.Buyer{TIN: "61200064714", ConsumerNo: "10038071"}
		}, []string{"buyer.consumerNo"}},
		{"consumer no on B2C invoice", func(s *receipt.Sale) { s.Type = receipt.B2CInvoice; s.Buyer.ConsumerNo = "10038071" }, []string{"buyer.consumerNo"}},
		{"report month on B2C receipt", func(s *receipt.Sale) { s.ReportMonth = month }, []string{"reportMonth"}},
		{"report month on B2B receipt", func(s *receipt.Sale) { s.Type = receipt.B2BReceipt; s.Buyer.TIN = "6"; s.ReportMonth = month }, nil},
		{"report month on B2C invoice", func(s *receipt.Sale) { s.Type = receipt.B2CInvoice; s.ReportMonth = month }, nil},
		{"report month on B2B invoice", func(s *receipt.Sale) { s.Type = receipt.B2BInvoice; s.Buyer.TIN = "6"; s.ReportMonth = month }, nil},
		{"report month 13", func(s *receipt.Sale) {
			s.Type = receipt.B2CInvoice
			s.ReportMonth = &receipt.Month{Year: 2026, Month: 13}
		}, []string{"reportMonth"}},
		{"unknown type", func(s *receipt.Sale) { s.Type = "B2X" }, []string{"type"}},
		{"empty type", func(s *receipt.Sale) { s.Type = "" }, []string{"type"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ok()
			tt.mutate(&s)
			_, err := receipt.Build(s, m)
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			assert.ElementsMatch(t, tt.want, fields(invalid(t, err)))
		})
	}

	t.Run("merchant tin required", func(t *testing.T) {
		_, err := receipt.Build(ok(), receipt.Merchant{})
		assert.Equal(t, []string{"merchant.tin"}, fields(invalid(t, err)))
	})
}

func TestBuild_VATFreeProjectSendsEverythingAsVATFree304(t *testing.T) {
	m := receipt.Merchant{TIN: merchantTIN, VATPayer: true, CityTaxPayer: true, VATFreeProject: true}
	s := receipt.Sale{Type: receipt.B2CReceipt, Items: []receipt.Item{
		item("vat", tugrik(1100), receipt.TaxVATable),
		item("zero", tugrik(500), receipt.TaxVATZeroRated),
		item("outside", tugrik(200), receipt.TaxOutsideVAT),
	}, Payments: []receipt.Payment{{Code: receipt.PaymentCash, Amount: tugrik(1800)}}}
	s.Items[0].HasCityTax = true

	req, err := receipt.Build(s, m)
	require.NoError(t, err)
	require.Len(t, req.Receipts, 1)
	r := req.Receipts[0]
	assert.Equal(t, posapi.TaxVATExempt, r.TaxType)
	require.Len(t, r.Items, 3)
	for _, it := range r.Items {
		assert.Equal(t, "304", it.TaxProductCode, it.Name)
	}
	assert.Equal(t, "0", req.TotalVat.String(), "no VAT is added to exempt items")
	// The city tax still applies to a VAT_FREE line: 1100/1.02*0.02.
	assert.Equal(t, "21.57", r.Items[0].TotalCityTax.String())

	// Without the flag the same sale keeps its three tax types.
	req, err = receipt.Build(s, receipt.Merchant{TIN: merchantTIN, VATPayer: true, CityTaxPayer: true})
	require.NoError(t, err)
	assert.Len(t, req.Receipts, 3)
}

func TestBuild_BillIDSuffixSentAsGivenOmittedWhenEmpty(t *testing.T) {
	s := receipt.Sale{Type: receipt.B2CReceipt, Items: []receipt.Item{item("a", tugrik(100), receipt.TaxVATable)},
		Payments: []receipt.Payment{{Code: receipt.PaymentCash, Amount: tugrik(100)}}}
	req, err := receipt.Build(s, receipt.Merchant{TIN: merchantTIN})
	require.NoError(t, err)
	b, _ := json.Marshal(req)
	assert.NotContains(t, string(b), "billIdSuffix")

	s.BillIDSuffix = "01"
	req, err = receipt.Build(s, receipt.Merchant{TIN: merchantTIN})
	require.NoError(t, err)
	b, _ = json.Marshal(req)
	assert.Contains(t, string(b), `"billIdSuffix":"01"`)
}

func TestBuild_TypedItemAndReceiptData(t *testing.T) {
	it := item("vodka", tugrik(5000), receipt.TaxVATable)
	it.Barcode = "4007675700002"
	it.BarcodeType = receipt.BarcodeGS1
	it.ExciseStampQRs = []string{"QR1", "QR2"}
	it.MedicineLotNo = "LOT-9"
	plain := item("water", tugrik(100), receipt.TaxVATable)
	s := receipt.Sale{
		Type: receipt.B2CReceipt, Items: []receipt.Item{it, plain},
		Payments:          []receipt.Payment{{Code: receipt.PaymentCash, Amount: tugrik(5100)}},
		ReplacesReceiptID: "037900846788001096190000210005299",
		Location:          &receipt.Location{Type: receipt.LocationGPS, Latitude: "47.012312", Longitude: "106.121213", Accuracy: "100m", IPAddress: "192.168.13.1"},
	}
	req, err := receipt.Build(s, receipt.Merchant{TIN: merchantTIN})
	require.NoError(t, err)
	b, err := json.Marshal(req)
	require.NoError(t, err)
	js := string(b)
	assert.Contains(t, js, `"inActiveId":"037900846788001096190000210005299"`)
	assert.Contains(t, js, `"data":{"lotNo":"LOT-9","stockQR":["QR1","QR2"]}`)
	assert.Contains(t, js, `"data":{"location":[{"locationType":"GPS","latitude":"47.012312","longitude":"106.121213","accuracy":"100m","ipAddress":"192.168.13.1"}]}`)
	assert.Contains(t, js, `"barCodeType":"GS1"`)
	assert.Contains(t, js, `"barCodeType":"UNDEFINED"`, "empty barcode type defaults to UNDEFINED")
	assert.Contains(t, js, `"data":null`, "absent data stays null like the goldens")
	assert.Equal(t, `"reportMonth":null`, js[strings.Index(js, `"reportMonth"`):strings.Index(js, `"reportMonth"`)+len(`"reportMonth":null`)])
}

// Structured property: any well-formed sale, varied along the axes that move
// the arithmetic (merchant flags, tax types, city-tax flags, quantities,
// cents), builds, and header = sum of sub-receipts = sum of items = sum of
// payments, exactly, with every tax a multiple of 0.01.
func TestBuild_SumsAreExactForGeneratedSales(t *testing.T) {
	rng := rand.New(rand.NewSource(37))
	types := []receipt.TaxType{receipt.TaxVATable, receipt.TaxVATExempt, receipt.TaxVATZeroRated, receipt.TaxOutsideVAT}
	for n := 0; n < 500; n++ {
		m := receipt.Merchant{TIN: merchantTIN, VATPayer: rng.Intn(2) == 0, CityTaxPayer: rng.Intn(2) == 0}
		var items []receipt.Item
		var total money.Amount
		for i, k := 0, 1+rng.Intn(6); i < k; i++ {
			tt := types[rng.Intn(len(types))]
			amt := money.FromDecimal(decimal.New(int64(1+rng.Intn(500000)), -2)) // 0.01 .. 5000.00
			it := item("i", amt, tt)
			it.HasCityTax = rng.Intn(2) == 0
			it.Quantity = decimal.New(int64(1+rng.Intn(3000)), -int32(rng.Intn(4)))
			items = append(items, it)
			total = total.Add(amt)
		}
		half := total.Div(decimal.NewFromInt(2))
		s := receipt.Sale{Type: receipt.B2CReceipt, Items: items, Payments: []receipt.Payment{
			{Code: receipt.PaymentCash, Amount: half}, {Code: receipt.PaymentCard, Amount: total.Sub(half)},
		}}
		req, err := receipt.Build(s, m)
		require.NoError(t, err, "case %d", n)

		var rAmt, rVat, rCity money.Amount
		for _, r := range req.Receipts {
			var iAmt, iVat, iCity money.Amount
			for _, it := range r.Items {
				iAmt, iVat, iCity = iAmt.Add(it.TotalAmount), iVat.Add(it.TotalVat), iCity.Add(it.TotalCityTax)
				require.True(t, it.TotalVat.Equal(money.FromDecimal(it.TotalVat.Decimal())), "VAT not 2dp")
			}
			require.True(t, r.TotalAmount.Equal(iAmt) && r.TotalVat.Equal(iVat) && r.TotalCityTax.Equal(iCity), "sub-receipt != items, case %d", n)
			rAmt, rVat, rCity = rAmt.Add(r.TotalAmount), rVat.Add(r.TotalVat), rCity.Add(r.TotalCityTax)
		}
		require.True(t, req.TotalAmount.Equal(rAmt) && req.TotalVat.Equal(rVat) && req.TotalCityTax.Equal(rCity), "header != receipts, case %d", n)
		require.True(t, req.TotalAmount.Equal(total), "header != sum of inputs, case %d", n)
		var paid money.Amount
		for _, p := range req.Payments {
			paid = paid.Add(p.PaidAmount)
		}
		require.True(t, paid.Equal(req.TotalAmount), "payments != header, case %d", n)
	}
}
