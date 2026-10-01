package oidc_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/oidc"
)

// tokenServer answers the password grant like Keycloak does: expires_in is a
// JSON number.
func tokenServer(t *testing.T, tokens *atomic.Int32, form *url.Values) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		f, _ := url.ParseQuery(string(b))
		mu.Lock()
		if form != nil {
			*form = f
		}
		mu.Unlock()
		tokens.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok-1","expires_in":300,"refresh_expires_in":1800,"refresh_token":"r","token_type":"Bearer","not-before-policy":0,"session_state":"s","scope":"profile"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTokenIsFetchedOnceAndReused(t *testing.T) {
	var tokens atomic.Int32
	var form url.Values
	ts := tokenServer(t, &tokens, &form)
	cfg := oidc.Config{TokenURL: ts.URL, ClientID: oidc.ClientVATPS, Username: "AA10010110", Password: "pw"}

	src := cfg.TokenSource(context.Background())
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tok, err := src.Token()
			assert.NoError(t, err)
			assert.Equal(t, "tok-1", tok.AccessToken)
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, tokens.Load(), "50 callers must cost one token request")

	assert.Equal(t, "password", form.Get("grant_type"))
	assert.Equal(t, "vatps", form.Get("client_id"))
	assert.Equal(t, "AA10010110", form.Get("username"))
	assert.Equal(t, "pw", form.Get("password"))
	assert.Empty(t, form.Get("client_secret"))
}

func TestHTTPClientAddsBearerAndOptionalAPIKey(t *testing.T) {
	var tokens atomic.Int32
	ts := tokenServer(t, &tokens, nil)

	var mu sync.Mutex
	var seen []http.Header
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Clone())
		mu.Unlock()
	}))
	defer api.Close()

	cfg := oidc.Config{TokenURL: ts.URL, ClientID: oidc.ClientVATPS, Username: "u", Password: "p"}
	for _, key := range []string{"", "KEY-1"} {
		hc := cfg.HTTPClient(context.Background(), key)
		for i := 0; i < 3; i++ {
			resp, err := hc.Get(api.URL)
			require.NoError(t, err)
			resp.Body.Close()
		}
	}
	require.Len(t, seen, 6)
	for i, h := range seen {
		assert.Equal(t, "Bearer tok-1", h.Get("Authorization"))
		if i < 3 {
			assert.Empty(t, h.Get("X-API-KEY"))
		} else {
			assert.Equal(t, "KEY-1", h.Get("X-API-KEY"))
		}
	}
	assert.EqualValues(t, 2, tokens.Load(), "one token request per client, not per call")
}

func TestTokenErrorCarriesCodeNotBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Invalid user credentials","secret":"LEAK-ME"}`))
	}))
	defer srv.Close()
	_, err := oidc.Config{TokenURL: srv.URL, ClientID: "vatps", Username: "u", Password: "bad"}.TokenSource(context.Background()).Token()
	var e *ebarimt.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, ebarimt.Rejected, e.Kind)
	assert.Equal(t, 401, e.HTTPStatus)
	assert.Equal(t, "invalid_grant", e.Status)
	assert.Equal(t, "Invalid user credentials", e.Message)
	assert.NotContains(t, err.Error(), "LEAK-ME")
	assert.NotContains(t, err.Error(), "bad")
}

func TestTokenEndpointDownIsTransport(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	_, err := oidc.Config{TokenURL: url, ClientID: "vatps"}.TokenSource(context.Background()).Token()
	var e *ebarimt.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, ebarimt.Transport, e.Kind)
}

func TestMissingConfigIsInvalid(t *testing.T) {
	_, err := oidc.Config{}.TokenSource(context.Background()).Token()
	var e *ebarimt.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, ebarimt.Invalid, e.Kind)
}
