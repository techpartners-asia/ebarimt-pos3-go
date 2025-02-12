package main

import (
	"fmt"

	ebarimt3 "github.com/techpartners-asia/ebarimt-pos3-go/client"
	pos3Models "github.com/techpartners-asia/ebarimt-pos3-go/client/models"
	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
)

// EBARIMT_ENDPOINT: "http://103.50.205.106:7080"
// EBARIMT_POS_NO: "10005608"
// EBARIMT_MERCHANT_TIN: "87001066048"

const (
	OrgCode      = ""
	BranchNo     = "2"
	DistrictCode = "3420"
)

func main() {

	sdk := ebarimt3.New(ebarimt3.Input{
		Endpoint:    "http://103.50.205.106:7080",
		MerchantTin: "87001066048",
		PosNo:       "10005608",
	})

	// * NOTE * - VAT_ABLE & VAT_ZERO & VAT_FREE & NO_VAT
	items := []pos3Models.CreateItemInputModel{{
		Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
		TaxType:            constants.TAX_VAT_ZERO,
		ClassificationCode: "2441030",
		Qty:                10,
		IsCityTax:          true,
		MeasureUnit:        "unit",
		TotalAmount:        50,
		TaxProductCode:     "447",
	}, {
		Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
		TaxType:            constants.TAX_VAT_FREE,
		ClassificationCode: "2441030",
		Qty:                2,
		IsCityTax:          true,
		MeasureUnit:        "unit",
		TotalAmount:        100,
		TaxProductCode:     "447",
	}, {
		Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
		TaxType:            constants.TAX_VAT_ABLE,
		ClassificationCode: "2441030",
		Qty:                1,
		IsCityTax:          false,
		MeasureUnit:        "unit",
		TotalAmount:        20,
		TaxProductCode:     "447",
	}, {
		Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
		TaxType:            constants.TAX_NO_VAT,
		ClassificationCode: "2441030",
		Qty:                22,
		IsCityTax:          true,
		MeasureUnit:        "unit",
		TotalAmount:        400,
		TaxProductCode:     "447",
	}, {
		Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
		TaxType:            constants.TAX_VAT_ABLE,
		ClassificationCode: "2441030",
		Qty:                1,
		IsCityTax:          true,
		MeasureUnit:        "unit",
		TotalAmount:        20,
		TaxProductCode:     "447",
	}, {
		Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
		TaxType:            constants.TAX_VAT_ABLE,
		ClassificationCode: "2441030",
		Qty:                3,
		IsCityTax:          true,
		MeasureUnit:        "unit",
		TotalAmount:        44,
		TaxProductCode:     "447",
	},
		{
			Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
			TaxType:            constants.TAX_VAT_FREE,
			ClassificationCode: "2441030",
			Qty:                2,
			IsCityTax:          true,
			MeasureUnit:        "unit",
			TotalAmount:        109,
			TaxProductCode:     "447",
		},
		{
			Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
			TaxType:            constants.TAX_VAT_ZERO,
			ClassificationCode: "2441030",
			Qty:                10,
			IsCityTax:          true,
			MeasureUnit:        "unit",
			TotalAmount:        55,
			TaxProductCode:     "447",
		},
	}

	if _, err := sdk.Create(pos3Models.CreateInputModel{
		OrgCode:      OrgCode,
		BranchNo:     BranchNo,
		DistrictCode: DistrictCode,
		Items:        items,
		DB:           nil,
	}); err != nil {
		fmt.Println("VAT ABLE & VAT ZERO & VAT FREE & NO VAT ERROR: ", err)
		return
	}

	fmt.Println("VAT ABLE & VAT ZERO & VAT FREE & NO VAT SUCCESS")

}
