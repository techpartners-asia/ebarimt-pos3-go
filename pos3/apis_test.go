package pos3

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"resty.dev/v3"
)

// A stalled POS endpoint must not hang the caller: with a bounded client
// timeout the request returns an error instead of blocking indefinitely.
func TestPosRequestTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := New(ConnectionInput{
		PosEndpoint: srv.URL,
		Client:      resty.New().SetTimeout(20 * time.Millisecond),
	}).(*pos3)

	start := time.Now()
	_, err := p.httpPosRequest(nil, PosInfoAPI, "", nil)
	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("request did not honour the timeout: took %s", elapsed)
	}
}

// New's default client must not follow redirects. A 3xx surfaces as its own
// response, never a silent hop to the location it points at — following one is
// what stalled eBarimt callers before.
func TestDefaultClientDoesNotFollowRedirects(t *testing.T) {
	var followed bool
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/followed")
		w.WriteHeader(http.StatusFound)
		_, _ = w.Write([]byte("REDIRECT_BODY"))
	})
	mux.HandleFunc("/followed", func(w http.ResponseWriter, r *http.Request) {
		followed = true
		_, _ = w.Write([]byte("FOLLOWED_BODY"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := New(ConnectionInput{PosEndpoint: srv.URL}).(*pos3)

	body, err := p.httpPosRequest(nil, PosInfoAPI, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if followed {
		t.Fatal("client followed the redirect; it must not")
	}
	if strings.Contains(string(body), "FOLLOWED_BODY") {
		t.Fatalf("got the redirect target's body, expected the 3xx body: %s", body)
	}
}

// A successful POS call returns the raw body for the caller to decode.
func TestPosRequestReturnsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"SUCCESS"}`))
	}))
	defer srv.Close()

	p := New(ConnectionInput{PosEndpoint: srv.URL}).(*pos3)

	body, err := p.httpPosRequest(nil, PosSendAPI, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), "SUCCESS") {
		t.Fatalf("unexpected body: %s", body)
	}
}
