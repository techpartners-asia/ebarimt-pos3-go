package ebarimtv3

import (
	"fmt"

	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"github.com/techpartners-asia/ebarimt-pos3-go/utils"
)

func (e *EbarimtClient) buildRequest(input structs.CreateInputModel) structs.ReceiptRequest {
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
			if len(input.CustomerTin) > 0 {
				res, err := e.GetInfo(input.CustomerTin)
				if err != nil {
					return ""
				}
				if res.Status != 200 {
					return ""
				}
				return input.CustomerTin
			}

			if len(input.OrgCode) > 0 {
				tin, err := e.GetTinInfo(input.OrgCode)
				if err != nil {
					return ""
				}
				return fmt.Sprintf("%d", tin.Data)
			}
			return ""
		}(),
		// TotalAmount:  input.TotalAmount,
		// TotalVat:     input.TotalVat,
		// TotalCityTax: input.TotalCityTax,
		ConsumerNo: "",
		ReportMonth: func() *string {
			if input.ReportMonth != nil {
				return input.ReportMonth
			}
			return nil
		}(),
	}

	return ebarimtRequest
}

// * NOTE * : Step - 2 : Categorying by Tax Type
func (e *EbarimtClient) buildReceiptItemMap(items []structs.CreateItemInputModel, receiptRequest *structs.ReceiptRequest) (map[constants.TaxType]structs.Receipt, error) {

	info, err := e.GetInfo(e.GetMerchantTin())
	if err != nil {
		return nil, err
	}

	receiptItems := make(map[constants.TaxType]structs.Receipt, len(items))

	for _, item := range items {
		if len(item.TaxType) == 0 {
			continue
		}

		if len(item.ClassificationCode) == 0 {
			continue
		}

		productTaxType := item.TaxType

		receiptItem := structs.Item{
			BarCode:            "",
			MeasureUnit:        item.MeasureUnit,
			BarCodeType:        constants.BARCODE_UNDEFINED,
			ClassificationCode: item.ClassificationCode,
			TotalVat: func() float64 {
				if !info.Data.VatPayer {
					return 0
				}

				if item.TaxType == constants.TAX_VAT_ABLE {
					if item.IsCityTax {
						return utils.GetVatWithCityTax(item.TotalAmount)
					}
					return utils.GetVat(item.TotalAmount)
				}

				return 0
			}(),
			UnitPrice:   utils.NumberPrecision(item.TotalAmount / float64(item.Qty)),
			TotalAmount: item.TotalAmount,
			TotalCityTax: func() float64 {
				if item.TaxType != constants.TAX_VAT_ABLE {
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
		receipt.TotalAmount += receiptItem.TotalAmount
		receipt.TotalVat += receiptItem.TotalVat
		receipt.TotalCityTax += receiptItem.TotalCityTax
		receipt.CustomerTin = receiptRequest.CustomerTin
		receipt.MerchantTin = e.GetMerchantTin()
		receipt.Items = append(receipt.Items, receiptItem)
		receiptItems[productTaxType] = receipt
	}

	return receiptItems, nil
}

// * NOTE * : Step - 3 Format Receipts like grouping [{tax_type , items : []}]
func (e *EbarimtClient) buildReceipt(request *structs.ReceiptRequest, items map[constants.TaxType]structs.Receipt) {
	receipts := make([]structs.Receipt, 0, len(items))
	totalAmount := 0.0
	totalVat := 0.0
	totalCityTax := 0.0

	for _, item := range items {

		receipts = append(receipts, item)
		totalAmount += item.TotalAmount
		totalVat += item.TotalVat
		totalCityTax += item.TotalCityTax
	}

	request.TotalAmount = totalAmount
	request.TotalVat = totalVat
	request.TotalCityTax = totalCityTax
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

}
