package ebarimt3Sdk

import (
	"fmt"

	"github.com/techpartners-asia/ebarimt-pos3-go/client/models"
	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"github.com/techpartners-asia/ebarimt-pos3-go/utils"
)

func (e *EbarimtClient) buildRequest(input models.CreateInputModel) structs.ReceiptRequest {
	ebarimtRequest := structs.ReceiptRequest{
		BranchNo: fmt.Sprintf("%v", input.BranchNo),
		DistrictCode: func() string {
			return input.DistrictCode
		}(),
		MerchantTin: e.GetMerchantTin(),
		PosNo:       e.GetPosNo(),
		Type: func() constants.ReceiptType {
			if len(input.OrgCode) == 0 {
				return constants.RECEIPT_B2C_RECEIPT
			}
			return constants.RECEIPT_B2B_RECEIPT
		}(),
		CustomerTin: func() string {
			if len(input.OrgCode) > 0 {
				tin, err := e.GetTinInfo(input.OrgCode)
				if err != nil {
					return ""
				}
				return tin.Data
			}
			return ""
		}(),
		ConsumerNo:  "",
		ReportMonth: nil,
	}

	return ebarimtRequest
}

// * NOTE * : Step - 2 : Categorying by Tax Type
func (e *EbarimtClient) buildReceiptItemMap(items []models.CreateItemInputModel) map[constants.TaxType]structs.Receipt {

	receiptItems := make(map[constants.TaxType]structs.Receipt, len(items))

	for _, item := range items {
		if len(item.TaxType) == 0 {
			continue
		}

		productTaxType := item.TaxType

		receiptItem := structs.Item{
			BarCode:            "",
			MeasureUnit:        item.MeasureUnit,
			BarCodeType:        constants.BARCODE_UNDEFINED,
			ClassificationCode: item.ClassificationCode,
			TotalVat: func() float64 {
				if item.TaxType == constants.TAX_VAT_ABLE {
					if item.IsCityTax {
						fmt.Println(utils.GetVatWithCityTax(item.TotalAmount))
						return utils.GetVatWithCityTax(item.TotalAmount)
					}
					fmt.Println(utils.GetVat(item.TotalAmount))
					return utils.GetVat(item.TotalAmount)
				}
				return 0
			}(),
			UnitPrice:   item.TotalAmount / float64(item.Qty),
			TotalAmount: item.TotalAmount,
			TotalCityTax: func() float64 {
				if item.TaxType == constants.TAX_NO_VAT {
					return 0
				}

				if item.IsCityTax {
					if item.TaxType == constants.TAX_VAT_ABLE {
						return utils.GetCityTax(item.TotalAmount)
					}
					return utils.GetCityTaxWithoutVat(item.TotalAmount)
				}
				return 0
			}(),
			Qty: item.Qty,
			TaxProductCode: func() string {
				if item.TaxType == constants.TAX_VAT_ABLE {
					return ""
				}

				return item.TaxProductCode
			}(),
		}

		receiptItem.Name = item.Name
		receipt := receiptItems[productTaxType]
		receipt.TaxType = productTaxType
		receipt.TotalAmount = receipt.TotalAmount + receiptItem.TotalAmount
		receipt.TotalVat = receipt.TotalVat + receiptItem.TotalVat
		receipt.TotalCityTax = receipt.TotalCityTax + receiptItem.TotalCityTax

		// receipt.TotalAmount = receipt.TotalAmount
		// receipt.TotalVat = receipt.TotalVat
		// receipt.TotalCityTax = receipt.TotalCityTax

		receipt.MerchantTin = e.GetMerchantTin()
		receipt.Items = append(receipt.Items, receiptItem)
		receiptItems[productTaxType] = receipt
	}

	return receiptItems
}

// * NOTE * : Step - 3 Format Receipts like grouping [{tax_type , items : []}]
func (e *EbarimtClient) buildReceipt(request *structs.ReceiptRequest, items map[constants.TaxType]structs.Receipt) {

	request.TotalAmount = 0
	request.TotalVat = 0
	request.TotalCityTax = 0

	receipts := make([]structs.Receipt, 0, len(items))

	for _, item := range items {
		request.TotalAmount += item.TotalAmount
		request.TotalVat += item.TotalVat
		request.TotalCityTax += item.TotalCityTax

		item.MerchantTin = e.GetMerchantTin()
		receipts = append(receipts, item)
	}

	request.Receipts = receipts

	request.Payments = []structs.Payment{
		{
			Code:   constants.PAYMENT_CARD,
			Status: constants.STATUS_PAID,
			PaidAmount: func() float64 {
				return request.TotalAmount
			}(),
		},
	}

	return
}
