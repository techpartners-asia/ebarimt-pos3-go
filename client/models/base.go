package models

import (
	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
)

type (
	CreateInputModel struct {
		OrgCode      string                 `json:"org_code"`
		BranchNo     string                 `json:"branch_no"`
		DistrictCode string                 `json:"district_code"`
		Payments     []structs.Payment      `json:"payments"` // Хоосон явуулбал , Payments нь автоматаар Card төлбөр болгон
		Items        []CreateItemInputModel `json:"items"`
		// DB           *gorm.DB               // Хоосон байж болно. Хэрвээ байвал, database дээр хадгална автоматаар
	}

	CreateItemInputModel struct {
		Name               string            `json:"name"`
		TaxType            constants.TaxType `json:"tax_type"`
		ClassificationCode string            `json:"classification_code"`
		Qty                float64           `json:"qty"`
		IsCityTax          bool              `json:"is_city_tax"`
		IsIncludedTax      bool              `json:"is_included_tax"`
		MeasureUnit        string            `json:"measure_unit"`
		TotalAmount        float64           `json:"total_amount"`
		TaxProductCode     string            `json:"tax_product_code"`
	}
)
