// Package httpx is the one HTTP round trip shared by posapi and registry.
// It owns the rules that must not differ between them: redirects are never
// followed, bodies are size-capped, a non-2xx answer is always an error, and no
// response body ever ends up inside an error.
package httpx

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync/atomic"
	"time"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
)

const (
	// DefaultTimeout bounds each request end to end.
	DefaultTimeout = 90 * time.Second
	maxBody        = 8 << 20
)

// Doer performs requests for one client. It has no mutable state after
// construction and is safe for concurrent use.
type Doer struct {
	HTTP    *http.Client
	Timeout time.Duration
}

// New builds a Doer. hc, when set, replaces the default transport (tests,
// custom certificates, an oidc client); it is copied, never modified.
// Redirects are never followed regardless.
func New(hc *http.Client, timeout time.Duration) *Doer {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	var c http.Client
	if hc != nil {
		c = *hc
	} else {
		c = http.Client{Transport: NewTransport(timeout)}
	}
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Doer{HTTP: &c, Timeout: timeout}
}

// NewTransport keeps connection establishment short (an unreachable host fails
// fast) while the wait for the first response header gets the full timeout: the
// PosAPI daemon can legitimately take tens of seconds to answer /rest/receipt
// while it flushes its backlog to the tax authority.
func NewTransport(timeout time.Duration) *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       20,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
		ForceAttemptHTTP2:     true,
	}
}

// Result is what came back from a 2xx answer.
type Result struct {
	Status int
	Body   []byte
	// Written reports that the request was completely written to the
	// connection. It is meaningful even when Do returned an error: a failure
	// with Written true means the remote system may have acted on the request.
	Written bool
}

// Do performs one request and returns the body of a 2xx answer.
//
// A transport failure is an *ebarimt.Error of Kind Transport. A non-2xx answer
// is Kind Rejected when its body is a {status,message} document, and Kind
// Transport with only the HTTP status otherwise (an HTML error page, an empty
// body): without the remote system's own words it is a failure of the path, not
// a refusal.
func (d *Doer) Do(ctx context.Context, op, method, target string, header http.Header, body []byte) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()

	var res Result
	var written atomic.Bool
	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		WroteRequest: func(i httptrace.WroteRequestInfo) {
			if i.Err == nil {
				written.Store(true)
			}
		},
	})

	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, rd)
	if err != nil {
		return res, &ebarimt.Error{Kind: ebarimt.Invalid, Op: op, Err: err}
	}
	for k, v := range header {
		req.Header[k] = v
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := d.HTTP.Do(req)
	res.Written = written.Load()
	if err != nil {
		return res, &ebarimt.Error{Kind: ebarimt.Transport, Op: op, Err: err}
	}
	defer resp.Body.Close()
	res.Status = resp.StatusCode

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	res.Written = written.Load()
	if err != nil {
		return res, &ebarimt.Error{Kind: ebarimt.Transport, Op: op, HTTPStatus: resp.StatusCode, Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		e := &ebarimt.Error{Kind: ebarimt.Transport, Op: op, HTTPStatus: resp.StatusCode}
		e.Status, e.Message = ErrorDoc(data)
		if e.Status != "" || e.Message != "" {
			e.Kind = ebarimt.Rejected
		}
		return res, e
	}
	res.Body = data
	return res, nil
}

// ErrorDoc extracts status and message from a {status,message} / {msg}
// document. Anything else (an HTML error page, an empty body) yields "", "".
func ErrorDoc(data []byte) (status, message string) {
	var d struct {
		Status  json.RawMessage `json:"status"`
		Message string          `json:"message"`
		Msg     string          `json:"msg"`
	}
	if json.Unmarshal(data, &d) != nil {
		return "", ""
	}
	message = d.Message
	if message == "" {
		message = d.Msg
	}
	return RawString(d.Status), message
}

// RawString renders a JSON string or number as plain text.
func RawString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return strings.TrimSpace(string(raw))
}

// Undecodable is the error for a 2xx answer that cannot be read as the
// document the call expects. The cause is a decoder error, never the body.
func Undecodable(op string, httpStatus int, err error) *ebarimt.Error {
	return &ebarimt.Error{Kind: ebarimt.Transport, Op: op, HTTPStatus: httpStatus, Message: "undecodable response", Err: err}
}
