// Package posapi is the client for the local PosAPI 3.0 daemon: issue a
// receipt, return one, flush the daemon's backlog to the tax authority, read its
// status and bank accounts.
//
// Guide: https://developer.itc.gov.mn/detail/proj-1787042993564?item=doc-1787208308636
//
// A Client has no mutable state after construction and is safe for concurrent
// use. The package logs nothing and retries nothing: IssueReceipt files a tax
// document and is not idempotent, so a retry decision belongs to the caller,
// who has the sale to reconcile against. Redirects are never followed.
//
// Every failure is an *ebarimt.Error (see its Kind). A response body never
// appears in one.
package posapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/internal/httpx"
)

// DefaultTimeout bounds each request end to end.
const DefaultTimeout = httpx.DefaultTimeout

// Config configures a Client.
type Config struct {
	// Endpoint is the base URL of the PosAPI daemon, e.g. http://10.0.0.5:7080.
	// It is always supplied by the caller: the daemon runs on the caller's
	// network. Required.
	Endpoint string
	// Timeout bounds each request end to end. Zero means DefaultTimeout. The
	// daemon can take tens of seconds to answer /rest/receipt while it flushes
	// its backlog, so keep it generous.
	Timeout time.Duration
	// HTTPClient is used instead of the default transport (tests, custom
	// certificates). It is copied; redirects are never followed regardless.
	HTTPClient *http.Client
}

// Client talks to one PosAPI daemon.
type Client struct {
	endpoint string
	doer     *httpx.Doer
}

// New builds a Client from cfg. An empty Endpoint is reported by the first
// call, as an Invalid error.
func New(cfg Config) *Client {
	return &Client{
		endpoint: strings.TrimRight(cfg.Endpoint, "/"),
		doer:     httpx.New(cfg.HTTPClient, cfg.Timeout),
	}
}

func (c *Client) do(ctx context.Context, op, method, path string, body []byte) (httpx.Result, error) {
	if c.endpoint == "" {
		return httpx.Result{}, &ebarimt.Error{Kind: ebarimt.Invalid, Op: op,
			Violations: []ebarimt.Violation{{Field: "Config.Endpoint", Rule: "the PosAPI daemon URL is required"}}}
	}
	return c.doer.Do(ctx, op, method, c.endpoint+path, nil, body)
}

// IssueReceipt files a receipt: POST {Endpoint}/rest/receipt
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787209763945).
//
// Failures, by Kind:
//
//	Rejected       PosAPI answered and refused (a non-2xx answer carrying its
//	               message, or a 2xx whose status is not SUCCESS). Nothing was
//	               filed; the message says what to fix.
//	Transport      the request never fully left (dial failure, cancelled
//	               context, TLS), or the path answered with a non-2xx and no
//	               PosAPI document (a proxy's HTML page). Nothing was filed as
//	               far as the daemon is concerned.
//	Indeterminate  the request was fully written and then the answer was lost
//	               or unreadable. The receipt may exist. Do not retry blindly:
//	               reconcile against the daemon first, or the sale is filed
//	               twice.
//
// IssuedReceipt carries lottery and qrData; print them and discard them.
func (c *Client) IssueReceipt(ctx context.Context, req ReceiptRequest) (IssuedReceipt, error) {
	const op = "posapi.IssueReceipt"
	body, err := json.Marshal(req)
	if err != nil {
		return IssuedReceipt{}, &ebarimt.Error{Kind: ebarimt.Invalid, Op: op, Err: err}
	}
	res, err := c.do(ctx, op, http.MethodPost, "/rest/receipt", body)
	if err != nil {
		var e *ebarimt.Error
		// Err is set only when no answer was obtained (or its body was lost);
		// a non-2xx answer without a PosAPI document has no Err and stays Transport.
		if errors.As(err, &e) && e.Kind == ebarimt.Transport && e.Err != nil && res.Written {
			e.Kind = ebarimt.Indeterminate
		}
		return IssuedReceipt{}, err
	}
	var wire ReceiptResponse
	if err := json.Unmarshal(res.Body, &wire); err != nil {
		// The daemon answered 2xx: the receipt very likely exists.
		e := httpx.Undecodable(op, res.Status, err)
		e.Kind = ebarimt.Indeterminate
		return IssuedReceipt{}, e
	}
	if wire.Status != StatusSuccess {
		return IssuedReceipt{}, &ebarimt.Error{Kind: ebarimt.Rejected, Op: op,
			HTTPStatus: res.Status, Status: wire.Status, Message: wire.Message}
	}
	return IssuedReceipt{
		ID:           wire.ID,
		Version:      wire.Version,
		Type:         wire.Type,
		TotalAmount:  wire.TotalAmount,
		TotalVAT:     wire.TotalVat,
		TotalCityTax: wire.TotalCityTax,
		BranchNo:     wire.BranchNo,
		DistrictCode: wire.DistrictCode,
		MerchantTIN:  wire.MerchantTin,
		PosNo:        wire.PosNo,
		CustomerTIN:  wire.CustomerTin,
		ConsumerNo:   wire.ConsumerNo,
		Receipts:     wire.Receipts,
		Payments:     wire.Payments,
		PosID:        int64(wire.PosID),
		Date:         ParseTime(wire.Date),
		DateText:     wire.Date,
		Easy:         wire.Easy,
		QRData:       wire.QrData,
		Lottery:      wire.Lottery,
	}, nil
}

// ReturnReceipt returns (deactivates) a receipt: DELETE {Endpoint}/rest/receipt
// with the JSON body {"id":..,"date":..}
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787212335705).
//
// issuedAt is the date PosAPI printed on the receipt (IssuedReceipt.Date); it is
// formatted as given, "yyyy-MM-dd HH:mm:ss", without converting zones. Only B2C
// receipts the citizen has not yet confirmed can be returned.
func (c *Client) ReturnReceipt(ctx context.Context, receiptID string, issuedAt time.Time) error {
	const op = "posapi.ReturnReceipt"
	body, err := json.Marshal(deleteRequest{ID: receiptID, Date: issuedAt.Format(DateLayout)})
	if err != nil {
		return &ebarimt.Error{Kind: ebarimt.Invalid, Op: op, Err: err}
	}
	res, err := c.do(ctx, op, http.MethodDelete, "/rest/receipt", body)
	if err != nil {
		return err
	}
	return checkAck(op, res)
}

// SendToTaxAuthority asks the daemon to flush its backlog to the tax authority:
// GET {Endpoint}/rest/sendData
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787212677960).
// The daemon also does this on its own schedule.
func (c *Client) SendToTaxAuthority(ctx context.Context) error {
	const op = "posapi.SendToTaxAuthority"
	res, err := c.do(ctx, op, http.MethodGet, "/rest/sendData", nil)
	if err != nil {
		return err
	}
	return checkAck(op, res)
}

// checkAck validates the answer of a call whose success carries no value: an
// empty body is fine; anything else must be JSON, and a document whose status is
// the string "ERROR" is a refusal. The daemon's ack document models status as a
// number: only 0 and 200 read as success, any other code is a refusal even on a
// 2xx answer.
func checkAck(op string, res httpx.Result) error {
	if len(strings.TrimSpace(string(res.Body))) == 0 {
		return nil
	}
	var d struct {
		Status  json.RawMessage `json:"status"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(res.Body, &d); err != nil {
		return httpx.Undecodable(op, res.Status, err)
	}
	st := httpx.RawString(d.Status)
	if strings.EqualFold(st, "ERROR") {
		return &ebarimt.Error{Kind: ebarimt.Rejected, Op: op, HTTPStatus: res.Status, Status: st, Message: d.Message}
	}
	if n, err := strconv.Atoi(st); err == nil && n != 0 && n != 200 {
		return &ebarimt.Error{Kind: ebarimt.Rejected, Op: op, HTTPStatus: res.Status, Status: st, Message: d.Message}
	}
	return nil
}

// Status reads the daemon's state: GET {Endpoint}/rest/info
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787212420028).
// Watch LotteriesLeft and LastSentAt: an exhausted lottery pool or a missed
// send deadline (3 days) makes the daemon issue lottery-less receipts.
func (c *Client) Status(ctx context.Context) (DaemonStatus, error) {
	const op = "posapi.Status"
	res, err := c.do(ctx, op, http.MethodGet, "/rest/info", nil)
	if err != nil {
		return DaemonStatus{}, err
	}
	var st DaemonStatus
	if err := json.Unmarshal(res.Body, &st); err != nil {
		return DaemonStatus{}, httpx.Undecodable(op, res.Status, err)
	}
	return st, nil
}

// BankAccounts lists the active bank accounts registered for a merchant or its
// tenant: GET {Endpoint}/rest/bankAccounts?tin=
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787212724910).
func (c *Client) BankAccounts(ctx context.Context, tin string) ([]BankAccount, error) {
	const op = "posapi.BankAccounts"
	if tin == "" {
		return nil, &ebarimt.Error{Kind: ebarimt.Invalid, Op: op,
			Violations: []ebarimt.Violation{{Field: "tin", Rule: "required"}}}
	}
	res, err := c.do(ctx, op, http.MethodGet, "/rest/bankAccounts?tin="+url.QueryEscape(tin), nil)
	if err != nil {
		return nil, err
	}
	var accounts []BankAccount
	if err := json.Unmarshal(res.Body, &accounts); err != nil {
		return nil, httpx.Undecodable(op, res.Status, err)
	}
	return accounts, nil
}
