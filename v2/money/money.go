// Package money holds exact decimal numbers for receipts.
//
// Amount is a money value in tugrik, always an exact multiple of 0.01. Quantity
// is an exact, unrounded decimal for item quantities. Neither ever passes
// through float64, so totals cannot drift: 457.40 + 1114.92 is 1572.32, not
// 1572.3200000000002.
//
// Both marshal to a JSON number in its shortest form (2800, 457.4, 1572.32),
// which is what PosAPI receives. The receipt spec requires totalAmount to be
// the exact sum of its parts:
// https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787209763945
package money

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// Amount is an exact money value with two decimal places. The zero value is 0.
// Every constructor and operation rounds half away from zero (half-up for the
// non-negative amounts a receipt carries) to two places.
type Amount struct{ d decimal.Decimal }

// FromTugrik is a whole-tugrik amount.
func FromTugrik(n int64) Amount { return Amount{decimal.NewFromInt(n)} }

// FromDecimal rounds d to two places.
func FromDecimal(d decimal.Decimal) Amount { return Amount{d.Round(2)} }

// Parse reads a decimal string such as "457.40", rounding to two places.
func Parse(s string) (Amount, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return Amount{}, fmt.Errorf("money: parse %q: %w", s, err)
	}
	return FromDecimal(d), nil
}

// MustParse is Parse for constants and tests; it panics on a bad string.
func MustParse(s string) Amount {
	a, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return a
}

// Decimal returns the exact value.
func (a Amount) Decimal() decimal.Decimal { return a.d }

// Add returns a+b.
func (a Amount) Add(b Amount) Amount { return Amount{a.d.Add(b.d)} }

// Sub returns a-b.
func (a Amount) Sub(b Amount) Amount { return Amount{a.d.Sub(b.d)} }

// Mul returns a*x rounded to two places.
func (a Amount) Mul(x decimal.Decimal) Amount { return FromDecimal(a.d.Mul(x)) }

// Div returns a/x rounded once, to two places. It panics if x is zero, like
// decimal.Decimal.
func (a Amount) Div(x decimal.Decimal) Amount { return Amount{a.d.DivRound(x, 2)} }

// Cmp compares a and b: -1, 0 or +1.
func (a Amount) Cmp(b Amount) int { return a.d.Cmp(b.d) }

// Equal reports whether a and b are the same amount.
func (a Amount) Equal(b Amount) bool { return a.d.Equal(b.d) }

// Sign returns -1, 0 or +1.
func (a Amount) Sign() int { return a.d.Sign() }

// IsZero reports whether a is 0.
func (a Amount) IsZero() bool { return a.d.IsZero() }

// String is the shortest decimal form: "2800", "457.4", "1572.32".
func (a Amount) String() string { return a.d.String() }

// MarshalJSON writes the shortest-form JSON number.
func (a Amount) MarshalJSON() ([]byte, error) { return []byte(a.d.String()), nil }

// UnmarshalJSON reads a JSON number or a numeric string, rounding to two
// places. null leaves the value unchanged.
func (a *Amount) UnmarshalJSON(b []byte) error {
	d, null, err := readNumber(b)
	if err != nil || null {
		return err
	}
	*a = FromDecimal(d)
	return nil
}

// Quantity is an exact, unrounded decimal (an item quantity such as 0.355 kg).
// The zero value is 0.
type Quantity struct{ d decimal.Decimal }

// QuantityFromDecimal wraps d without rounding.
func QuantityFromDecimal(d decimal.Decimal) Quantity { return Quantity{d} }

// Decimal returns the exact value.
func (q Quantity) Decimal() decimal.Decimal { return q.d }

// String is the shortest decimal form.
func (q Quantity) String() string { return q.d.String() }

// MarshalJSON writes the shortest-form JSON number.
func (q Quantity) MarshalJSON() ([]byte, error) { return []byte(q.d.String()), nil }

// UnmarshalJSON reads a JSON number or a numeric string. null leaves the value
// unchanged.
func (q *Quantity) UnmarshalJSON(b []byte) error {
	d, null, err := readNumber(b)
	if err != nil || null {
		return err
	}
	*q = Quantity{d}
	return nil
}

// readNumber accepts a JSON number or a quoted numeric string.
func readNumber(b []byte) (d decimal.Decimal, null bool, err error) {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		return d, true, nil
	}
	if len(s) > 0 && s[0] == '"' {
		u, uerr := strconv.Unquote(s)
		if uerr != nil {
			return d, false, fmt.Errorf("money: bad number %s", s)
		}
		s = strings.TrimSpace(u)
	}
	if !json.Valid([]byte(s)) {
		return d, false, fmt.Errorf("money: bad number %q", s)
	}
	d, err = decimal.NewFromString(s)
	if err != nil {
		return d, false, fmt.Errorf("money: bad number %q: %w", s, err)
	}
	return d, false, nil
}
