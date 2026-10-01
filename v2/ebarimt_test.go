package ebarimt_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
)

func TestErrorIsAndAs(t *testing.T) {
	var err error = fmt.Errorf("wrapped: %w", &ebarimt.Error{Kind: ebarimt.NotFound, Op: "registry.Taxpayer"})
	assert.ErrorIs(t, err, ebarimt.ErrNotFound)
	var e *ebarimt.Error
	assert.True(t, errors.As(err, &e))
	assert.Equal(t, ebarimt.NotFound, e.Kind)

	assert.NotErrorIs(t, &ebarimt.Error{Kind: ebarimt.Rejected}, ebarimt.ErrNotFound)
}

func TestErrorText(t *testing.T) {
	cause := errors.New("dial tcp 10.0.0.5:7080: connect: connection refused")
	assert.Equal(t, cause.Error(), (&ebarimt.Error{Kind: ebarimt.Transport, Op: "posapi.IssueReceipt", Err: cause}).Error(),
		"a transport failure reads exactly as its cause")
	assert.Equal(t, "ebarimt posapi.IssueReceipt: http 502",
		(&ebarimt.Error{Kind: ebarimt.Transport, Op: "posapi.IssueReceipt", HTTPStatus: 502}).Error())
	assert.Equal(t, "ebarimt posapi.IssueReceipt: http 400: status=ERROR: no good",
		(&ebarimt.Error{Kind: ebarimt.Rejected, Op: "posapi.IssueReceipt", HTTPStatus: 400, Status: "ERROR", Message: "no good"}).Error())
	assert.Contains(t, (&ebarimt.Error{Kind: ebarimt.Indeterminate, Op: "posapi.IssueReceipt", Err: cause}).Error(), "may exist")
	assert.Equal(t, "ebarimt receipt.Build: a: r1; b: r2",
		(&ebarimt.Error{Kind: ebarimt.Invalid, Op: "receipt.Build", Violations: []ebarimt.Violation{{Field: "a", Rule: "r1"}, {Field: "b", Rule: "r2"}}}).Error())
}

func TestEnvironmentZeroValueIsProduction(t *testing.T) {
	var e ebarimt.Environment
	assert.Equal(t, ebarimt.Production, e)
	assert.Equal(t, "production", e.String())
	assert.Equal(t, "staging", ebarimt.Staging.String())
	assert.Empty(t, ebarimt.Staging.ServiceURL(), "no staging service host is published; it must not silently be production")
}
