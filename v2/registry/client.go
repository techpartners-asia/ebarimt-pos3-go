// Package registry reads the public tax registry and reference lists of the
// eBarimt system (host api.ebarimt.mn, staging st-api.ebarimt.mn). Everything
// except MerchantLocations needs no credentials.
//
// A Client has no mutable state after construction and is safe for concurrent
// use. The package logs nothing. All calls are idempotent GETs, so they may
// retry: set Config.Retries (default 0, no retry). Redirects are never followed.
//
// Every failure is an *ebarimt.Error. A lookup that was answered but has no
// such subject is Kind NotFound (errors.Is(err, ebarimt.ErrNotFound)).
package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/internal/httpx"
)

// Config configures a Client.
type Config struct {
	// Environment selects the documented host. The zero value is Production.
	Environment ebarimt.Environment
	// BaseURL overrides the environment's registry host.
	BaseURL string
	// APIKey is the X-API-KEY of the supplier, needed only by MerchantLocations
	// (request it from posapi@itc.gov.mn).
	APIKey string
	// Timeout bounds each attempt. Zero means 90 seconds.
	Timeout time.Duration
	// Retries is how many extra attempts an idempotent GET gets after a
	// transport failure or a 5xx answer. Zero (the default) means none.
	Retries int
	// RetryBackoff is the wait before the first retry, doubling after each.
	// Zero means 200ms.
	RetryBackoff time.Duration
	// HTTPClient is used instead of the default transport. It is copied;
	// redirects are never followed regardless.
	HTTPClient *http.Client
}

// Client reads the registry.
type Client struct {
	base    string
	apiKey  string
	retries int
	backoff time.Duration
	doer    *httpx.Doer
}

// New builds a Client from cfg.
func New(cfg Config) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = cfg.Environment.RegistryURL()
	}
	backoff := cfg.RetryBackoff
	if backoff <= 0 {
		backoff = 200 * time.Millisecond
	}
	retries := cfg.Retries
	if retries < 0 {
		retries = 0
	}
	return &Client{
		base:    strings.TrimRight(base, "/"),
		apiKey:  cfg.APIKey,
		retries: retries,
		backoff: backoff,
		doer:    httpx.New(cfg.HTTPClient, cfg.Timeout),
	}
}

// get performs an idempotent GET with the configured retries.
func (c *Client) get(ctx context.Context, op, path string, query url.Values, withKey bool) (httpx.Result, error) {
	target := c.base + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	var header http.Header
	if withKey {
		header = http.Header{"X-Api-Key": []string{c.apiKey}}
	}
	wait := c.backoff
	for attempt := 0; ; attempt++ {
		res, err := c.doer.Do(ctx, op, http.MethodGet, target, header, nil)
		if err == nil || attempt >= c.retries || !retryable(err) {
			return res, err
		}
		select {
		case <-ctx.Done():
			return res, err
		case <-time.After(wait):
		}
		wait *= 2
	}
}

// retryable: a transport failure, or a 5xx answer.
func retryable(err error) bool {
	e, ok := err.(*ebarimt.Error)
	if !ok {
		return false
	}
	if e.Kind == ebarimt.Transport && e.Err != nil && e.HTTPStatus == 0 {
		return true
	}
	return e.HTTPStatus >= 500
}

// envelope is the {msg,status,data} wrapper of the lookup services.
type envelope struct {
	Msg    string          `json:"msg"`
	Status json.RawMessage `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// unwrap decodes the envelope of a 2xx answer. A status other than 200 is a
// Rejected error carrying the envelope's own status and message, unless the
// caller handles it first via the returned envelope (ok=false).
func unwrap(op string, res httpx.Result) (env envelope, status int, err error) {
	if err := json.Unmarshal(res.Body, &env); err != nil {
		return env, 0, httpx.Undecodable(op, res.Status, err)
	}
	st := httpx.RawString(env.Status)
	n, convErr := strconv.Atoi(st)
	if convErr != nil {
		return env, 0, &ebarimt.Error{Kind: ebarimt.Rejected, Op: op, HTTPStatus: res.Status, Status: st, Message: env.Msg}
	}
	return env, n, nil
}

func isNull(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s == "" || s == "null"
}

func rejected(op string, res httpx.Result, env envelope) *ebarimt.Error {
	return &ebarimt.Error{Kind: ebarimt.Rejected, Op: op, HTTPStatus: res.Status, Status: httpx.RawString(env.Status), Message: env.Msg}
}

// lookup runs a GET whose answer is an envelope with status 200 and returns its data.
func (c *Client) lookup(ctx context.Context, op, path string, query url.Values) (json.RawMessage, error) {
	res, err := c.get(ctx, op, path, query, false)
	if err != nil {
		return nil, err
	}
	env, status, err := unwrap(op, res)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, rejected(op, res, env)
	}
	return env.Data, nil
}

func invalid(op, field, rule string) *ebarimt.Error {
	return &ebarimt.Error{Kind: ebarimt.Invalid, Op: op, Violations: []ebarimt.Violation{{Field: field, Rule: rule}}}
}
