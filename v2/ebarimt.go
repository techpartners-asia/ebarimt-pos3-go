// Package ebarimt is the root of the v2 SDK for the Mongolian eBarimt PosAPI 3.0
// system (https://developer.itc.gov.mn/detail/proj-1787042993564).
//
// It holds the two things every other package shares: the Error type all calls
// return, and the Environment presets that name every documented host. The
// work itself lives in the sub-packages:
//
//	receipt   build a valid receipt request (pure, no I/O)
//	posapi    talk to the local PosAPI daemon
//	registry  public tax-registry lookups
//	oidc      token source for the authenticated tax-authority services
//	money     exact two-decimal amounts
//	postest   an in-memory fake PosAPI for tests
package ebarimt

import (
	"errors"
	"fmt"
	"strings"
)

// Kind classifies a failed call. Callers branch on it; Message and Status carry
// what the remote system said.
type Kind int

const (
	// Invalid means the request was refused locally, before anything was sent:
	// a field broke a documented rule. Violations lists every broken rule.
	Invalid Kind = iota + 1
	// Rejected means the remote system answered and said no. Status and Message
	// carry its own words. Nothing was filed.
	Rejected
	// NotFound means the lookup was answered and the subject does not exist.
	// errors.Is(err, ErrNotFound) is true.
	NotFound
	// Transport means the call failed without a usable answer, and for a call
	// that files something, before the request was fully written: nothing was
	// filed, a retry is safe.
	Transport
	// Indeterminate means a receipt may exist. The request was fully written,
	// then the answer was lost (connection dropped, timeout, unreadable
	// reply). The caller must not blindly retry: reconcile first, or the sale
	// is filed twice.
	Indeterminate
)

func (k Kind) String() string {
	switch k {
	case Invalid:
		return "invalid"
	case Rejected:
		return "rejected"
	case NotFound:
		return "not found"
	case Transport:
		return "transport"
	case Indeterminate:
		return "indeterminate"
	}
	return fmt.Sprintf("kind(%d)", int(k))
}

// ErrNotFound matches any *Error of Kind NotFound under errors.Is.
var ErrNotFound = errors.New("ebarimt: not found")

// Violation is one broken rule of a locally refused request.
type Violation struct {
	Field string // path of the offending field, e.g. "items[1].classificationCode"
	Rule  string // what is required
}

func (v Violation) String() string { return v.Field + ": " + v.Rule }

// Error is the one error type every call in this SDK returns, so callers can
// errors.As it. It never carries a response body: lottery and qrData of an
// issued receipt cannot leak through it.
//
//   - Transport failure: Err holds the cause, HTTPStatus is 0, and Error() is
//     exactly the cause's text.
//   - Non-2xx answer: HTTPStatus is set; Status and Message are filled when the
//     body is a {status,message} (or {msg}) document, empty otherwise.
//   - 2xx answer that is a refusal (receipt status other than SUCCESS, a lookup
//     envelope status other than 200): HTTPStatus is the 2xx code, Status and
//     Message carry the document's own values.
type Error struct {
	Kind       Kind
	Op         string // which call, e.g. "posapi.IssueReceipt"
	HTTPStatus int
	Status     string // the remote document's own status, if it had one
	Message    string // the remote document's own message, if it had one
	Violations []Violation
	Err        error
}

func (e *Error) Error() string {
	if e.Kind == Transport && e.Err != nil && e.HTTPStatus == 0 {
		return e.Err.Error()
	}
	var b strings.Builder
	b.WriteString("ebarimt ")
	b.WriteString(e.Op)
	if e.Kind == Indeterminate {
		b.WriteString(": outcome unknown, a receipt may exist")
	}
	if e.HTTPStatus != 0 {
		fmt.Fprintf(&b, ": http %d", e.HTTPStatus)
	}
	if e.Status != "" {
		fmt.Fprintf(&b, ": status=%s", e.Status)
	}
	if e.Message != "" {
		b.WriteString(": ")
		b.WriteString(e.Message)
	}
	if len(e.Violations) > 0 {
		parts := make([]string, len(e.Violations))
		for i, v := range e.Violations {
			parts[i] = v.String()
		}
		b.WriteString(": ")
		b.WriteString(strings.Join(parts, "; "))
	}
	if e.Err != nil {
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
	}
	if e.Kind == NotFound && e.Message == "" && len(e.Violations) == 0 && e.Err == nil {
		b.WriteString(": not found")
	}
	return b.String()
}

func (e *Error) Unwrap() error { return e.Err }

// Is makes errors.Is(err, ErrNotFound) true for every NotFound error.
func (e *Error) Is(target error) bool { return target == ErrNotFound && e.Kind == NotFound }

// Environment names a complete set of documented hosts. The zero value is
// Production. The PosAPI daemon's address is not part of it: that is always
// supplied by the caller, because the daemon runs on the caller's network.
//
// Every URL a client uses can still be overridden in that client's Config.
type Environment int

const (
	// Production is the live system.
	Production Environment = iota
	// Staging is the test environment
	// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=doc-1787215000711).
	Staging
)

func (e Environment) String() string {
	if e == Staging {
		return "staging"
	}
	return "production"
}

// RegistryURL is the base URL of the public registry (package registry).
func (e Environment) RegistryURL() string {
	if e == Staging {
		return "https://st-api.ebarimt.mn"
	}
	return "https://api.ebarimt.mn"
}

// TokenURL is the OpenID Connect token endpoint of the authenticated services
// (package oidc).
func (e Environment) TokenURL() string {
	if e == Staging {
		return "https://st.auth.itc.gov.mn/auth/realms/Staging/protocol/openid-connect/token"
	}
	return "https://auth.itc.gov.mn/auth/realms/ITC/protocol/openid-connect/token"
}

// ServiceURL is the base URL of the service.itc.gov.mn services (excise-stamp
// inventory, planned for v2.1). The docs publish no staging host for it, so
// Staging returns "" rather than guess; a client must be given one explicitly.
func (e Environment) ServiceURL() string {
	if e == Staging {
		return ""
	}
	return "https://service.itc.gov.mn"
}
