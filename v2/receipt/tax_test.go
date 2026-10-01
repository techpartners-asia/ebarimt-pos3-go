package receipt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/receipt"
)

// Literal expected values, worked by hand from total = base + VAT + city tax:
//
//	VAT + city:  base = 2800/1.12 = 2500        -> 250.00 / 50.00
//	VAT only:    base = 2800/1.10 = 2545.4545.. -> 254.55 / 0
//	city only:   base = 2800/1.02 = 2745.0980.. -> 0 / 54.90
//	neither:                                        0 / 0
func TestSplitTaxes_2800_AllFourBranches(t *testing.T) {
	total := money.FromTugrik(2800)
	tests := []struct {
		name        string
		m           receipt.Merchant
		taxType     receipt.TaxType
		hasCity     bool
		wantVAT     string
		wantCityTax string
	}{
		{"vat+city", receipt.Merchant{VATPayer: true, CityTaxPayer: true}, receipt.TaxVATable, true, "250", "50"},
		{"vat only", receipt.Merchant{VATPayer: true, CityTaxPayer: true}, receipt.TaxVATable, false, "254.55", "0"},
		{"city only", receipt.Merchant{VATPayer: true, CityTaxPayer: true}, receipt.TaxVATExempt, true, "0", "54.9"},
		{"neither", receipt.Merchant{VATPayer: true, CityTaxPayer: true}, receipt.TaxVATExempt, false, "0", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vat, city := receipt.SplitTaxes(total, tt.taxType, tt.hasCity, tt.m)
			assert.Equal(t, tt.wantVAT, vat.String(), "vat")
			assert.Equal(t, tt.wantCityTax, city.String(), "city tax")
		})
	}
}

// Each gate must matter on its own: the merchant flags, the item flag and the
// tax type. A dropped gate flips exactly one of these.
func TestSplitTaxes_EveryGateMatters(t *testing.T) {
	total := money.FromTugrik(2800)
	both := receipt.Merchant{VATPayer: true, CityTaxPayer: true}

	// addVAT = VATPayer && VAT_ABLE
	vat, _ := receipt.SplitTaxes(total, receipt.TaxVATable, false, receipt.Merchant{VATPayer: false, CityTaxPayer: true})
	assert.True(t, vat.IsZero(), "non-VAT-payer must not add VAT")
	for _, tt := range []receipt.TaxType{receipt.TaxVATExempt, receipt.TaxVATZeroRated, receipt.TaxOutsideVAT} {
		vat, _ := receipt.SplitTaxes(total, tt, false, both)
		assert.True(t, vat.IsZero(), "%s must not add VAT", tt)
	}

	// addCity = CityTaxPayer && HasCityTax && type != NOT_VAT
	_, city := receipt.SplitTaxes(total, receipt.TaxVATable, true, receipt.Merchant{VATPayer: true, CityTaxPayer: false})
	assert.True(t, city.IsZero(), "non-city-payer must not add city tax")
	_, city = receipt.SplitTaxes(total, receipt.TaxVATable, false, both)
	assert.True(t, city.IsZero(), "item without the city-tax flag must not add city tax")
	_, city = receipt.SplitTaxes(total, receipt.TaxOutsideVAT, true, both)
	assert.True(t, city.IsZero(), "NOT_VAT must not add city tax")
	_, city = receipt.SplitTaxes(total, receipt.TaxVATZeroRated, true, both)
	assert.Equal(t, "54.9", city.String(), "VAT_ZERO still carries city tax")
}

func TestSplitTaxes_RoundsHalfUpOnce(t *testing.T) {
	m := receipt.Merchant{VATPayer: true}
	// 1100/11 = 100 exactly; 1.10/11 = 0.1; 0.55/11 = 0.05; 0.60/11 = 0.0545.. -> 0.05; 0.61/11 = 0.0554.. -> 0.06
	for in, want := range map[string]string{"1100": "100", "1.10": "0.1", "0.55": "0.05", "0.60": "0.05", "0.61": "0.06"} {
		vat, _ := receipt.SplitTaxes(money.MustParse(in), receipt.TaxVATable, false, m)
		assert.Equal(t, want, vat.String(), in)
	}
	// 457.40/11 = 41.5818.. ; 1114.92/11 = 101.3563..
	vat, _ := receipt.SplitTaxes(money.MustParse("457.40"), receipt.TaxVATable, false, m)
	assert.Equal(t, "41.58", vat.String())
	vat, _ = receipt.SplitTaxes(money.MustParse("1114.92"), receipt.TaxVATable, false, m)
	assert.Equal(t, "101.36", vat.String())
}
