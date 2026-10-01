// Package postest is an in-memory fake of the PosAPI 3.0 daemon, so code that
// issues receipts can be tested without a daemon, a network or a tax identity.
//
//	pos := postest.New(t)
//	client := pos.Client() // a posapi.Client pointed at the fake
//	issued, err := client.IssueReceipt(ctx, req)
//	pos.Requests()         // everything it received, in order
//
// The fake enforces the rules PosAPI itself enforces, and answers a violation
// the way the daemon does: HTTP 400 with {"status":"ERROR","message":...}:
//
//   - totals are exact: each sub-receipt equals the sum of its items, the header
//     equals the sum of the sub-receipts, and payments (if any) equal the header;
//   - classificationCode is present and 7 digits on every item;
//   - taxProductCode is present on VAT_FREE, VAT_ZERO and NOT_VAT items;
//   - B2B receipts carry customerTin;
//   - an item's city tax and VAT agree with its total (within 0.01), and a
//     wrong one is refused with PosAPI's message
//     "НХАТ-г 2%-р тооцоолоогүй байна. Шалгуур дүн: <expected>";
//   - DELETE /rest/receipt must carry a body naming a receipt the fake issued.
//
// A good receipt is answered SUCCESS with a 33-digit id, a lottery (B2C_RECEIPT
// only, as in production) and qrData. The values are fake. The fake does not
// know a merchant's registration, so it cannot tell that a missing city tax or
// VAT should have been there; it only checks what is present.
//
// Its wording of messages other than the city-tax one is modelled on the live
// daemon's but is not guaranteed identical: assert on Kind and HTTPStatus, not
// on message text, outside the city-tax case.
package postest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/posapi"
)

// Request is one request the fake received.
type Request struct {
	Method string
	Path   string
	Query  string
	Body   []byte
}

// Server is a running fake daemon. It is closed when the test ends.
type Server struct {
	// URL is the daemon's base URL; use it as posapi.Config.Endpoint.
	URL string

	srv *httptest.Server

	mu       sync.Mutex
	reqs     []Request
	issued   map[string]time.Time // receipt id -> when issued
	returned map[string]bool
	seq      int
	failNext []failure
}

type failure struct {
	status  int
	message string
}

// New starts a fake daemon that lives until the end of t.
func New(t testing.TB) *Server {
	t.Helper()
	s := &Server{issued: map[string]time.Time{}, returned: map[string]bool{}}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	s.URL = s.srv.URL
	t.Cleanup(s.srv.Close)
	return s
}

// Client returns a posapi.Client pointed at the fake.
func (s *Server) Client() *posapi.Client {
	return posapi.New(posapi.Config{Endpoint: s.URL, HTTPClient: s.srv.Client(), Timeout: 10 * time.Second})
}

// Requests returns everything received so far, oldest first.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.reqs...)
}

// FailNext makes the next request answer with HTTP status and
// {"status":"ERROR","message":message} instead of being processed. Several calls
// queue up.
func (s *Server) FailNext(status int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = append(s.failNext, failure{status, message})
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	s.reqs = append(s.reqs, Request{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: body})
	var fail *failure
	if len(s.failNext) > 0 {
		f := s.failNext[0]
		s.failNext = s.failNext[1:]
		fail = &f
	}
	s.mu.Unlock()
	if fail != nil {
		writeError(w, fail.status, fail.message)
		return
	}

	switch {
	case r.URL.Path == "/rest/receipt" && r.Method == http.MethodPost:
		s.issue(w, body)
	case r.URL.Path == "/rest/receipt" && r.Method == http.MethodDelete:
		s.returnReceipt(w, body)
	case r.URL.Path == "/rest/sendData" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{})
	case r.URL.Path == "/rest/info" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"operatorName": "TEST OPERATOR1", "operatorTIN": "37900846788", "posId": 101317077, "posNo": "101317077",
			"lastSentDate": time.Now().In(posapi.Ulaanbaatar()).Format(posapi.DateLayout), "leftLotteries": 1000,
			"merchants": []any{},
		})
	case r.URL.Path == "/rest/bankAccounts" && r.Method == http.MethodGet:
		if r.URL.Query().Get("tin") == "" {
			writeError(w, http.StatusBadRequest, "tin талбарын утга хоосон")
			return
		}
		writeJSON(w, http.StatusOK, []any{})
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) issue(w http.ResponseWriter, body []byte) {
	var req posapi.ReceiptRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "request body is not a valid receipt")
		return
	}
	if msg := validate(req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	s.mu.Lock()
	s.seq++
	n := s.seq
	id := fmt.Sprintf("%012s%021d", req.MerchantTin, n)
	id = strings.ReplaceAll(id, " ", "0")
	if len(id) > 33 {
		id = id[len(id)-33:]
	}
	s.issued[id] = time.Now()
	s.mu.Unlock()

	resp := posapi.ReceiptResponse{
		ID: id, Version: "3.2.44",
		TotalAmount: req.TotalAmount, TotalVat: req.TotalVat, TotalCityTax: req.TotalCityTax,
		BranchNo: req.BranchNo, DistrictCode: req.DistrictCode, MerchantTin: req.MerchantTin, PosNo: req.PosNo,
		CustomerTin: req.CustomerTin, ConsumerNo: req.ConsumerNo, Type: req.Type, InvoiceID: req.InvoiceID,
		Payments: req.Payments, PosID: 101317077, Status: posapi.StatusSuccess,
		QrData: fmt.Sprintf("3089232652190338989305256596260686282968065577729989820643412572570296032561569375192317676711258451339246060371%080d", n),
		Date:   time.Now().In(posapi.Ulaanbaatar()).Format(posapi.DateLayout),
	}
	if req.Type == posapi.B2CReceipt {
		resp.Lottery = fmt.Sprintf("AA %08d", n)
	}
	for i, r := range req.Receipts {
		r.ID = fmt.Sprintf("%s9%02d", id[:30], i+1) // distinct from the header id
		resp.Receipts = append(resp.Receipts, r)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) returnReceipt(w http.ResponseWriter, body []byte) {
	if len(strings.TrimSpace(string(body))) == 0 {
		writeError(w, http.StatusBadRequest, "id талбарын утга хоосон")
		return
	}
	var d struct {
		ID   string `json:"id"`
		Date string `json:"date"`
	}
	if err := json.Unmarshal(body, &d); err != nil || d.ID == "" {
		writeError(w, http.StatusBadRequest, "id талбарын утга хоосон")
		return
	}
	if _, err := time.Parse(posapi.DateLayout, d.Date); err != nil {
		writeError(w, http.StatusBadRequest, `date талбар "yyyy-MM-dd HH:mm:ss" форматтай байх ёстой`)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.issued[d.ID]; !ok {
		writeError(w, http.StatusBadRequest, "баримт олдсонгүй")
		return
	}
	if s.returned[d.ID] {
		writeError(w, http.StatusBadRequest, "баримт аль хэдийн буцаагдсан байна")
		return
	}
	s.returned[d.ID] = true
	writeJSON(w, http.StatusOK, map[string]any{})
}

var (
	tolerance = money.MustParse("0.01")
	rateVAT   = decimal.New(10, -2)
	rateCity  = decimal.New(2, -2)
)

// validate returns PosAPI's refusal message, or "" for an acceptable request.
func validate(req posapi.ReceiptRequest) string {
	if len(req.Receipts) == 0 {
		return "receipts талбарын утга хоосон"
	}
	if req.MerchantTin == "" {
		return "merchantTin талбарын утга хоосон"
	}
	if (req.Type == posapi.B2BReceipt || req.Type == posapi.B2BInvoice) && req.CustomerTin == "" {
		return "customerTin талбарын утга хоосон"
	}
	var sumAmount, sumVat, sumCity money.Amount
	for _, r := range req.Receipts {
		if len(r.Items) == 0 {
			return "items талбарын утга хоосон"
		}
		var a, v, c money.Amount
		for _, it := range r.Items {
			if msg := validateItem(r.TaxType, it); msg != "" {
				return msg
			}
			a, v, c = a.Add(it.TotalAmount), v.Add(it.TotalVat), c.Add(it.TotalCityTax)
		}
		if !a.Equal(r.TotalAmount) || !v.Equal(r.TotalVat) || !c.Equal(r.TotalCityTax) {
			return "дэд баримтын дүн барааны дүнгийн нийлбэртэй тэнцэхгүй байна"
		}
		sumAmount, sumVat, sumCity = sumAmount.Add(r.TotalAmount), sumVat.Add(r.TotalVat), sumCity.Add(r.TotalCityTax)
	}
	if !sumAmount.Equal(req.TotalAmount) || !sumVat.Equal(req.TotalVat) || !sumCity.Equal(req.TotalCityTax) {
		return "totalAmount дүн дэд баримтуудын нийлбэртэй тэнцэхгүй байна"
	}
	if len(req.Payments) > 0 {
		var paid money.Amount
		for _, p := range req.Payments {
			paid = paid.Add(p.PaidAmount)
		}
		if !paid.Equal(req.TotalAmount) {
			return "төлбөрийн дүн totalAmount-тай тэнцэхгүй байна"
		}
	}
	return ""
}

func validateItem(taxType posapi.TaxType, it posapi.Item) string {
	if it.ClassificationCode == "" {
		return "classificationCode талбарын утга хоосон"
	}
	if len(it.ClassificationCode) != 7 || strings.Trim(it.ClassificationCode, "0123456789") != "" {
		return "classificationCode талбарын урт 7 оронтой тоо байх ёстой"
	}
	if taxType != posapi.TaxVATable && taxType != "" && it.TaxProductCode == "" {
		return "taxProductCode талбарын утга хоосон"
	}
	total := it.TotalAmount.Decimal()
	// total = base + VAT + city tax: the divisor depends on what is present.
	divisor := decimal.NewFromInt(1)
	if !it.TotalVat.IsZero() {
		divisor = divisor.Add(rateVAT)
	}
	if !it.TotalCityTax.IsZero() {
		divisor = divisor.Add(rateCity)
	}
	if !it.TotalCityTax.IsZero() {
		want := money.FromDecimal(total.Mul(rateCity).DivRound(divisor, 2))
		if diff(want, it.TotalCityTax) {
			return fmt.Sprintf("НХАТ-г 2%%-р тооцоолоогүй байна. Шалгуур дүн: %v", want)
		}
	}
	if !it.TotalVat.IsZero() {
		want := money.FromDecimal(total.Mul(rateVAT).DivRound(divisor, 2))
		if diff(want, it.TotalVat) {
			return fmt.Sprintf("НӨАТ-г 10%%-р тооцоолоогүй байна. Шалгуур дүн: %v", want)
		}
	}
	return ""
}

func diff(want, got money.Amount) bool {
	d := want.Sub(got)
	if d.Sign() < 0 {
		d = money.Amount{}.Sub(d)
	}
	return d.Cmp(tolerance) > 0
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"status": "ERROR", "message": message})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
