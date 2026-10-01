package receipt

import (
	"github.com/shopspring/decimal"

	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
)

var (
	vatRate  = decimal.New(10, -2) // 10%
	cityRate = decimal.New(2, -2)  // 2%
	one      = decimal.NewFromInt(1)
)

// SplitTaxes returns the VAT and city tax embedded in a line total.
//
// The spec defines the total as base + VAT + city tax
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787209763945:
// 1000 + 100 + 20 = 1120), so the divisor depends on which taxes are actually
// added, not on the item's flags alone:
//
//	addVAT  = m.VATPayer && t == TaxVATable
//	addCity = m.CityTaxPayer && hasCityTax && t != TaxOutsideVAT
//	base    = total / (1 + 0.10*addVAT + 0.02*addCity)
//	vat     = base * 0.10,  city = base * 0.02   (each rounded half-up to 2 places)
//
// A tax that is not added comes back as 0. Each result is rounded once, from
// the exact quotient.
func SplitTaxes(total money.Amount, t TaxType, hasCityTax bool, m Merchant) (vat, cityTax money.Amount) {
	addVAT := m.VATPayer && t == TaxVATable
	addCity := m.CityTaxPayer && hasCityTax && t != TaxOutsideVAT

	divisor := one
	if addVAT {
		divisor = divisor.Add(vatRate)
	}
	if addCity {
		divisor = divisor.Add(cityRate)
	}
	// total*rate/divisor: one division, one rounding.
	if addVAT {
		vat = money.FromDecimal(total.Decimal().Mul(vatRate).DivRound(divisor, 2))
	}
	if addCity {
		cityTax = money.FromDecimal(total.Decimal().Mul(cityRate).DivRound(divisor, 2))
	}
	return vat, cityTax
}
