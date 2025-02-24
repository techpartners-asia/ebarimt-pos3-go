package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"github.com/techpartners-asia/ebarimt-pos3-go/utils"
)

var items = []structs.CreateItemInputModel{{
	Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
	TaxType:            constants.TAX_VAT_ZERO,
	ClassificationCode: "2441030",
	Qty:                10,
	IsCityTax:          true,
	MeasureUnit:        "unit",
	TotalAmount:        2000,
	TaxProductCode:     "447",
}, {
	Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
	TaxType:            constants.TAX_VAT_FREE,
	ClassificationCode: "2441030",
	Qty:                2,
	IsCityTax:          true,
	MeasureUnit:        "unit",
	TotalAmount:        4550,
	TaxProductCode:     "447",
}, {
	Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
	TaxType:            constants.TAX_VAT_ABLE,
	ClassificationCode: "2441030",
	Qty:                1,
	IsCityTax:          false,
	MeasureUnit:        "unit",
	TotalAmount:        1350,
	TaxProductCode:     "447",
}, {
	Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
	TaxType:            constants.TAX_NO_VAT,
	ClassificationCode: "2441030",
	Qty:                22,
	IsCityTax:          true,
	MeasureUnit:        "unit",
	TotalAmount:        22000,
	TaxProductCode:     "447",
}, {
	Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
	TaxType:            constants.TAX_VAT_ABLE,
	ClassificationCode: "2441030",
	Qty:                1,
	IsCityTax:          true,
	MeasureUnit:        "unit",
	TotalAmount:        2300,
	TaxProductCode:     "447",
}, {
	Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
	TaxType:            constants.TAX_VAT_ABLE,
	ClassificationCode: "2441030",
	Qty:                3,
	IsCityTax:          true,
	MeasureUnit:        "unit",
	TotalAmount:        4500,
	TaxProductCode:     "447",
},
	{
		Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
		TaxType:            constants.TAX_VAT_FREE,
		ClassificationCode: "2441030",
		Qty:                2,
		IsCityTax:          true,
		MeasureUnit:        "unit",
		TotalAmount:        4800,
		TaxProductCode:     "447",
	},
	{
		Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
		TaxType:            constants.TAX_VAT_ZERO,
		ClassificationCode: "2441030",
		Qty:                10,
		IsCityTax:          true,
		MeasureUnit:        "unit",
		TotalAmount:        3700,
		TaxProductCode:     "447",
	},
}

func TestAmounts(t *testing.T) {
	assert := assert.New(t)

	totalVat := 0.0

	for _, item := range items {
		if item.TaxType == constants.TAX_NO_VAT {
			continue
		}

		if item.TaxType == constants.TAX_VAT_ABLE {
			if item.IsCityTax {
				totalVat += utils.GetVatWithCityTax(item.TotalAmount)
			} else {
				totalVat += utils.GetVat(item.TotalAmount)
			}
		}
	}

	assert.Equal(7.53, utils.NumberPrecision(totalVat), "GetVat func is not correct")
}

func TestVats(t *testing.T) {
	assert := assert.New(t)

	sdk := NewSdk()

	res, err := sdk.Create(structs.CreateInputModel{
		OrgCode:      OrgCode,
		BranchNo:     BranchNo,
		DistrictCode: DistrictCode,
		Items:        items,
	})

	assert.Nil(err, fmt.Sprintf("Ebarimt Error : %v ", res.Message))

	if err != nil {
		return
	}

	// for _, receipt := range res.Receipts {
	// 	assert.Equal(utils.NumberPrecision(receipt.TotalAmount), receipt.TotalAmount, "TotalAmount Precision ")
	// 	assert.Equal(utils.NumberPrecision(receipt.TotalVat), receipt.TotalVat, "TotalVat Precision")
	// 	assert.Equal(utils.NumberPrecision(receipt.TotalCityTax), receipt.TotalCityTax, "TotalCityTax Precision")

	// 	for _, item := range receipt.Items {
	// 		assert.Equal(utils.NumberPrecision(item.TotalAmount), item.TotalAmount, "Receipt Item TotalAmount Precision")
	// 		assert.Equal(utils.NumberPrecision(item.TotalVat), item.TotalVat, "Receipt Item TotalVat Precision")
	// 		assert.Equal(utils.NumberPrecision(item.TotalCityTax), item.TotalCityTax, "Receipt Item TotalCityTax Precision")
	// 		assert.Equal(utils.NumberPrecision(item.UnitPrice), item.UnitPrice, "Receipt Item UnitPrice Precision")
	// 	}
	// }

	assert.Equal(constants.POS_STATUS_SUCCESS, res.Status, "Ebarimt Error : %v", res.Message)
}

func TestItems(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(0.35, utils.NumberPrecision(0.35714285714285715), "Number Precision")
}
