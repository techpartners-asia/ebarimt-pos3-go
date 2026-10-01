// Package oidc provides the access token for the authenticated tax-authority
// services (token pages:
// https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787244940244,
// ...item=api-1787245192309, ...item=api-1787245375939).
//
// The services use the OAuth2 password grant against one Keycloak realm, with a
// client_id per service family ("vatps" for api.ebarimt.mn, "e-inventory" for
// service.itc.gov.mn). It is built on golang.org/x/oauth2: the token is cached
// and refreshed once, however many goroutines ask, so a burst of calls costs one
// token request.
//
// Errors from the token endpoint are *ebarimt.Error and carry the endpoint's
// error code and description, never its response body.
package oidc

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/oauth2"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
)

// Client IDs of the documented service families.
const (
	ClientVATPS     = "vatps"       // api.ebarimt.mn: sales, breakdowns, easy registration
	ClientInventory = "e-inventory" // service.itc.gov.mn: excise-stamp inventory
)

// Config names the token endpoint and the credentials of one service account.
type Config struct {
	// TokenURL is the token endpoint, e.g. ebarimt.Production.TokenURL().
	TokenURL string
	ClientID string
	Username string
	Password string
}

// TokenSource returns a source of valid access tokens. Tokens are fetched with
// the password grant on first use and again only when expired; concurrent
// callers share one fetch. ctx governs every token request the source ever
// makes (and carries a custom client via oauth2.HTTPClient), so pass a context
// that lives as long as the source is used, not a request-scoped one.
func (c Config) TokenSource(ctx context.Context) oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, &passwordSource{ctx: ctx, cfg: c})
}

// HTTPClient returns an *http.Client that adds "Authorization: Bearer <token>"
// to every request and, when apiKey is not empty, "X-API-KEY: <apiKey>"
// (needed by the sales-breakdown services). Hand it to a client's
// Config.HTTPClient.
func (c Config) HTTPClient(ctx context.Context, apiKey string) *http.Client {
	base := http.DefaultTransport
	if hc, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok && hc != nil && hc.Transport != nil {
		base = hc.Transport
	}
	return &http.Client{Transport: &oauth2.Transport{
		Source: c.TokenSource(ctx),
		Base:   apiKeyTransport{key: apiKey, base: base},
	}}
}

type apiKeyTransport struct {
	key  string
	base http.RoundTripper
}

func (t apiKeyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.key == "" {
		return t.base.RoundTrip(r)
	}
	r = r.Clone(r.Context())
	r.Header.Set("X-API-KEY", t.key)
	return t.base.RoundTrip(r)
}

type passwordSource struct {
	ctx context.Context
	cfg Config
}

func (s *passwordSource) Token() (*oauth2.Token, error) {
	const op = "oidc.Token"
	if s.cfg.TokenURL == "" || s.cfg.ClientID == "" {
		return nil, &ebarimt.Error{Kind: ebarimt.Invalid, Op: op, Violations: []ebarimt.Violation{
			{Field: "Config.TokenURL / Config.ClientID", Rule: "required"}}}
	}
	conf := oauth2.Config{
		ClientID: s.cfg.ClientID,
		Endpoint: oauth2.Endpoint{TokenURL: s.cfg.TokenURL, AuthStyle: oauth2.AuthStyleInParams},
	}
	tok, err := conf.PasswordCredentialsToken(s.ctx, s.cfg.Username, s.cfg.Password)
	if err != nil {
		var re *oauth2.RetrieveError
		if errors.As(err, &re) {
			// RetrieveError.Error() embeds the response body; keep only the
			// parsed code and description.
			msg := re.ErrorDescription
			if msg == "" {
				msg = re.ErrorCode
			}
			kind := ebarimt.Transport
			if re.Response != nil && re.Response.StatusCode >= 400 && re.Response.StatusCode < 500 {
				kind = ebarimt.Rejected
			}
			e := &ebarimt.Error{Kind: kind, Op: op, Status: re.ErrorCode, Message: msg}
			if re.Response != nil {
				e.HTTPStatus = re.Response.StatusCode
			}
			return nil, e
		}
		return nil, &ebarimt.Error{Kind: ebarimt.Transport, Op: op, Err: err}
	}
	return tok, nil
}
