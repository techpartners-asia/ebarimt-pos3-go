package money_test

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
)

func TestSumIsExact(t *testing.T) {
	// 457.40 + 1114.92 is 1572.3200000000002 in float64.
	got := money.MustParse("457.40").Add(money.MustParse("1114.92"))
	assert.Equal(t, "1572.32", got.String())
	assert.True(t, got.Equal(money.MustParse("1572.32")))
}

func TestRoundingIsHalfUpToTwoPlaces(t *testing.T) {
	for in, want := range map[string]string{
		"0.005":    "0.01",
		"0.004":    "0",
		"254.545":  "254.55",
		"254.5454": "254.55",
		"2.675":    "2.68", // float64 gets this wrong (2.67)
		"10":       "10",
		"10.10":    "10.1",
	} {
		assert.Equal(t, want, money.MustParse(in).String(), in)
	}
	assert.Equal(t, "0.34", money.FromDecimal(decimal.RequireFromString("0.3350")).String())
}

func TestMulDivRoundOnce(t *testing.T) {
	a := money.FromTugrik(2800)
	assert.Equal(t, "254.55", a.Div(decimal.RequireFromString("11")).String())
	assert.Equal(t, "280", a.Mul(decimal.RequireFromString("0.1")).String())
	assert.Equal(t, "1.67", money.FromTugrik(5).Div(decimal.RequireFromString("3")).String())
}

func TestJSONShortestForm(t *testing.T) {
	type doc struct {
		A money.Amount `json:"a"`
	}
	for amount, want := range map[string]string{
		"2800": `{"a":2800}`, "2800.00": `{"a":2800}`, "457.4": `{"a":457.4}`,
		"1572.32": `{"a":1572.32}`, "0": `{"a":0}`, "0.50": `{"a":0.5}`,
	} {
		b, err := json.Marshal(doc{money.MustParse(amount)})
		require.NoError(t, err)
		assert.Equal(t, want, string(b), amount)
	}
	// Zero value marshals as 0.
	b, err := json.Marshal(doc{})
	require.NoError(t, err)
	assert.Equal(t, `{"a":0}`, string(b))
}

func TestUnmarshalAcceptsNumberAndString(t *testing.T) {
	var d struct {
		A money.Amount   `json:"a"`
		B money.Amount   `json:"b"`
		C money.Quantity `json:"c"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"a":1572.32,"b":"1100","c":0.355}`), &d))
	assert.Equal(t, "1572.32", d.A.String())
	assert.Equal(t, "1100", d.B.String())
	assert.Equal(t, "0.355", d.C.String())
	require.Error(t, json.Unmarshal([]byte(`{"a":"abc"}`), &d))
	require.Error(t, json.Unmarshal([]byte(`{"a":true}`), &d))
}

func TestQuantityIsNotRounded(t *testing.T) {
	q := money.QuantityFromDecimal(decimal.RequireFromString("0.3555"))
	b, err := json.Marshal(q)
	require.NoError(t, err)
	assert.Equal(t, "0.3555", string(b))
}

func TestParseError(t *testing.T) {
	_, err := money.Parse("12,5")
	require.Error(t, err)
}
