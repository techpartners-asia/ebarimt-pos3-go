package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	ebarimt3SdkServices "github.com/techpartners-asia/ebarimt-pos3-go/services"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"github.com/techpartners-asia/ebarimt-pos3-go/utils"
)

var items = []structs.CreateItemInputModel{{
	Name:               "VAT & VAT ZERO & VAT FREE & NO VAT",
	TaxType:            constants.TAX_VAT_ZERO,
	ClassificationCode: "2441030",
	Qty:                1,
	IsCityTax:          true,
	MeasureUnit:        "unit",
	TotalAmount:        10,
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

func TestSendMail(t *testing.T) {
	ebarimt3SdkServices.SendMail(
		ebarimt3SdkServices.EmailInput{
			Email:            "burabatbold2@gmail.com",
			From:             "no-reply@lifetech.mn",
			Password:         "7fDf#mtz",
			SmtpHost:         "smtp.zoho.com",
			SmtpPort:         "587",
			StorageEndpoint:  "file-powerbank.lifetech.mn",
			StorageAccessKey: "admin",
			StorageSecretKey: "aV969{1]]5^L",

			Response: structs.ReceiptResponse{
				QrData:       "5407284065424164431453299078758426279428312571101109500241404625729681071321549735501597491072545005682667748344848224388939135598794838760522384975371681",
				ID:           "1234567890",
				Date:         time.Now().Format("2006-01-02"),
				TotalAmount:  100,
				TotalVat:     10,
				TotalCityTax: 10,
				Receipts: []structs.Receipt{
					{
						Items: []structs.Item{
							{
								Name:        "Test",
								Qty:         1,
								TotalAmount: 100,
							},
						},
					},
				},
			},
		},
	)
}
