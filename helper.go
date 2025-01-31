package ebarimt3

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"github.com/techpartners-asia/ebarimt-pos3-go/utils"
	"gorm.io/gorm"
)

type EbarimtV3Item struct {
	structs.Receipt
}

type CreateInput struct {
	// DB      *gorm.DB
	Order   *structs.Order
}

type ItemData struct {
	ItemID uint `json:"item-id"`
}

func (co *EbarimtClient) Create(input CreateInput) (*structs.Ebarimt ,error) {

	if utils.IsNil(input.Order) {
		return nil ,nil
	}


	// * NOTE * : Step - 1
	request := buildRequest(input.Order)

	request.Type = func() constants.ReceiptType {
		if !input.Order.IsOrg {
			return constants.RECEIPT_B2C_RECEIPT
		}
		return constants.RECEIPT_B2B_RECEIPT
	}()

	request.CustomerTin = func() string {
		if input.Order.IsOrg {
			tin, err := co.GetTinInfo(input.Order.OrgRegNo)
			if err != nil {
				return ""
			}
			return tin.Data
		}
		return ""
	}()

	request.Payments = []structs.Payment{
		{
			Code:   constants.PAYMENT_CARD,
			Status: constants.STATUS_PAID,
		},
	}

	// * NOTE * : Step - 2
	receiptItems := buildReceiptItemMap(input.Order)

	// * NOTE * : Step - 3 : Seperate Receipts by Tax Type
	if len(receiptItems) > 0 && len(receiptItems[constants.TAX_NO_VAT].Items) > 0 {
		receipts, totalVat, totalAmount, totalCityTax := buildReceipt(input.Order.MerchantTin,map[constants.TaxType]EbarimtV3Item{
			constants.TAX_NO_VAT: receiptItems[constants.TAX_NO_VAT],
		}, "")
		delete(receiptItems, constants.TAX_NO_VAT)

		requestNoVat := request
		requestNoVat.TotalAmount = totalAmount
		requestNoVat.TotalVat = totalVat
		requestNoVat.TotalCityTax = totalCityTax
		requestNoVat.Payments[0].PaidAmount = totalAmount
		requestNoVat.Receipts = receipts

		// * NOTE * : Step - 4
		_, err := co.makeRequest(requestNoVat, input.Order)
		if err != nil {
			return nil,err
		}

	}

	if len(receiptItems) == 0 {
		return nil, errors.New("Empty item")
	}

	// * NOTE * : Other Tax Types
	receipts, totalVat, totalAmount, totalCityTax := buildReceipt(input.Order.MerchantTin,receiptItems, "")
	request.TotalAmount = totalAmount
	request.TotalVat = totalVat
	request.TotalCityTax = totalCityTax
	request.Payments[0].PaidAmount = totalAmount
	request.Receipts = receipts

	// * NOTE * : Step - 4
	ebarimt, err := co.makeRequest(request, input.Order)
	if err != nil {
		return nil,err
	}

	return ebarimt, nil
}

type CreateInvoiceInput struct {
	DB            *gorm.DB
	Order         *structs.Order
	BankAccountNo string
	OrgCode       string
	Storage       *minio.Client
}

func (co *EbarimtClient) CreateForInvoice(input CreateInvoiceInput) (*structs.Ebarimt,error) {
	if utils.IsNil(input.Order) {
		return nil, errors.New("Empty request")
	}

	// * NOTE * : Step - 1

	request := buildRequest(input.Order)

	request.Type = func() constants.ReceiptType {
		if input.Order.Type ==constants.RECEIPT_B2C_INVOICE {
			return constants.RECEIPT_B2C_INVOICE
		}
		return constants.RECEIPT_B2B_INVOICE
	}()

	request.CustomerTin = func() string {
		if input.Order.Type == constants.RECEIPT_B2B_INVOICE {
			tin, err := co.GetTinInfo(input.OrgCode)
			if err != nil {
				return ""
			}
			return tin.Data
		}
		return ""
	}()

	// * NOTE * : Step - 2
	receiptItems := buildReceiptItemMap(input.Order)

	// * NOTE * : Step - 3 : Seperate Receipts by Tax Type
	if len(receiptItems) > 0 && len(receiptItems[constants.TAX_NO_VAT].Items) > 0 {
		receipts, totalVat, totalAmount, totalCityTax := buildReceipt(input.Order.MerchantTin,map[constants.TaxType]EbarimtV3Item{
			constants.TAX_VAT_ABLE: receiptItems[constants.TAX_NO_VAT],
		}, "")
		delete(receiptItems, constants.TAX_NO_VAT)
		request.TotalAmount = totalAmount
		request.TotalVat = totalVat
		request.TotalCityTax = totalCityTax
		// request.Payments[0].PaidAmount = totalAmount
		request.Receipts = receipts

		// * NOTE * : Step - 4
		_, err := co.makeRequest(request, input.Order)
		if err != nil {
			return nil,err
		}
	}

	if len(receiptItems) == 0 {
		return nil, errors.New("Empty items")
	}

	// * NOTE * : Other Tax Types
	receipts, totalVat, totalAmount, totalCityTax := buildReceipt(input.Order.MerchantTin, receiptItems, input.Order.BankAccountNo)
	request.TotalAmount = totalAmount
	request.TotalVat = totalVat
	request.TotalCityTax = totalCityTax
	// request.Payments[0].PaidAmount = totalAmount
	request.Receipts = receipts

	// * NOTE * : Step - 4
	ebarimt, err := co.makeRequest(request, input.Order)
	if err != nil {
		return nil,err
	}

	return ebarimt, nil
}

func (co *EbarimtClient) SendEbarimt() error {

	res, err := co.SendData()
	if err != nil {
		return err
	}

	fmt.Println("Ebarimt Send Data response : ", res)

	return nil
}

// * NOTE * : To use these below steps for creating Ebarimt and Invoice

// * NOTE * : Step - 1 : Building Request For invoice and product
func buildRequest(order *structs.Order) structs.ReceiptRequest {
	ebarimtRequest := structs.ReceiptRequest{
		TotalCityTax: 0,
		BranchNo:     fmt.Sprintf("%v", order.BranchID),
		DistrictCode: func() string {
			return "3420"
		}(),
		MerchantTin: viper.GetString("EBARIMT_MERCHANT_TIN"),
		PosNo:       viper.GetString("EBARIMT_POS_NO"),
		// Type: func() pos3.ReceiptType {
		// 	if !input.Payment.IsOrg {
		// 		return pos3.RECEIPT_B2C_RECEIPT
		// 	}
		// 	return pos3.RECEIPT_B2B_RECEIPT
		// }(),
		// CustomerTin: func() string {
		// 	if input.Payment.IsOrg {
		// 		tin, err := co.GetTinInfo(input.Payment.OrgRegNo)
		// 		if err != nil {
		// 			return ""
		// 		}
		// 		return tin.Data
		// 		// return input.Payment.OrgRegNo
		// 	}
		// 	return ""
		// }(),
		ConsumerNo:  "",
		ReportMonth: nil,
	}

	return ebarimtRequest
}

// * NOTE * : Step - 2 : Categorying by Tax Type
func buildReceiptItemMap(order *structs.Order) map[constants.TaxType]EbarimtV3Item {

	receiptItems := make(map[constants.TaxType]EbarimtV3Item, 0)

	for _, item := range order.Items {
		if len(item.TaxType) == 0 {
			continue
		}

		productTaxType := constants.TaxType(*&item.TaxType)

		receiptItem := structs.Item{
			BarCode:     "",
			MeasureUnit: "ш",
			BarCodeType: constants.BARCODE_UNDEFINED,
			Data: ItemData{
				ItemID: item.ID,
			},
			ClassificationCode: item.ClassificationCode,
			TotalVat: func() float64 {
				if item.TaxType == constants.TAX_VAT_ABLE {
					if item.IsCityTax {
						return utils.GetVatWithCityTax(item.PriceTotal)
					}
					return utils.GetVat(item.PriceTotal)
				}
				return 0
			}(),
			// UnitPrice:   item.PriceTotal / float64(item.Qty),
			TotalAmount: utils.NumberPrecision(item.PriceTotal),
			TotalCityTax: func() float64 {
				if item.TaxType == (constants.TAX_NO_VAT) {
					return 0
				}

				if item.IsCityTax {
					if item.TaxType == (constants.TAX_VAT_ABLE) {
						return utils.GetCityTax(item.PriceTotal)
					}
					return utils.GetCityTaxWithoutVat(item.PriceTotal)
				}
				return 0
			}(),
			Qty: (item.Qty),
			TaxProductCode: func() string {
				if item.TaxType == (constants.TAX_VAT_ZERO) || item.TaxType == (constants.TAX_VAT_FREE) || item.TaxType == (constants.TAX_NO_VAT) {
					return item.TaxProductCode
				}
				return ""
			}(),
		}

		
		receiptItem.Name = item.ProductName
		receipt := receiptItems[productTaxType]
		receipt.TaxType = productTaxType
		receipt.TotalAmount += utils.NumberPrecision(receiptItem.TotalAmount)
		receipt.TotalVat += utils.NumberPrecision(receiptItem.TotalVat)
		receipt.TotalCityTax += utils.NumberPrecision(receiptItem.TotalCityTax)
		receipt.Items = append(receipt.Items, receiptItem)
		receiptItems[productTaxType] = receipt
	}

	return receiptItems
}

// * NOTE * : Step - 3 Format Receipts like grouping [{tax_type , items : []}]
func buildReceipt(merchantTin string,items map[constants.TaxType]EbarimtV3Item, bankAccountNo string) (receipts []structs.Receipt, totalVat, totalAmount, totalCityTax float64) {

	totalVat = 0.0
	totalAmount = 0.0
	totalCityTax = 0.0

	receipts = make([]structs.Receipt, 0, len(items))

	for _, item := range items {
		totalCityTax += utils.NumberPrecision(item.TotalCityTax)
		totalAmount += utils.NumberPrecision(item.TotalAmount)
		totalVat += utils.NumberPrecision(item.TotalVat)
		item.MerchantTin = merchantTin
		item.BankAccountNo = bankAccountNo
		receipts = append(receipts, item.Receipt)
	}
	return
}

// * NOTE * : Step - 4 Create Ebarimt and Save To the DB
func (co *EbarimtClient) makeRequest(request structs.ReceiptRequest,order *structs.Order) (*structs.Ebarimt, error) {
	resp, err := co.ReceiptSend(request)
	if err != nil {
		return &structs.Ebarimt{}, err
	}

	if resp.Status != constants.POS_STATUS_SUCCESS {
		return &structs.Ebarimt{}, fmt.Errorf("ebarimt error: %s", resp.Message)
	}

	// url, err := storage.GenerateQr(client, resp.QrData)
	// if err != nil {
	// 	return &databases.Ebarimt{}, err
	// }

	instance := structs.Ebarimt{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		BillId:        resp.ID,
		BillType:      string(resp.Type),
		TaxType:       string(resp.Receipts[0].TaxType),
		NonCashAmount: strconv.FormatFloat(resp.TotalAmount, 'f', -1, 64),
		CashAmount:    strconv.FormatFloat(resp.TotalAmount, 'f', -1, 64),
		Amount:        strconv.FormatFloat(resp.TotalAmount, 'f', -1, 64),
		Vat:           strconv.FormatFloat(resp.TotalVat, 'f', -1, 64),
		CityTax:       strconv.FormatFloat(resp.TotalCityTax, 'f', -1, 64),
		BranchNo:      resp.BranchNo,
		DistrictCode:  resp.DistrictCode,
		RegisterNo:    resp.CustomerTIN,
		Date:          resp.Date,
		MerchantId:    resp.MerchantTIN,
		Success:       resp.Status == constants.POS_STATUS_SUCCESS,
		Message:       resp.Message,
		OrderID:       order.ID,
		QrData:        resp.QrData,
		Lottery:       resp.Lottery,
	}

	// if err := db.Create(&instance).Error; err != nil {
	// 	fmt.Println(err)
	// 	return &structs.Ebarimt{}, err
	// }

	// if err := saveStock(db, &instance, resp.Receipts); err != nil {
	// 	fmt.Println("Save ebarimt stock error : ", err.Error())
	// }

	// go func(order structs.Order, resp structs.ReceiptResponse) {
	// 	if err := mail.InitMail().Send(mail.EmailInput{
	// 		Email:    "burabatbold2@gmail.com",
	// 		Subtitle: "Ebarimt",
	// 		Type:     mail.EmailTypeEbarimt,
	// 	}, mail.EbarimtSend{
	// 		Email:      "burabatbold2@gmail.com",
	// 		Date:       resp.Date,
	// 		Amount:     strconv.FormatFloat(resp.TotalAmount, 'f', -1, 64),
	// 		NoatAmount: strconv.FormatFloat(resp.TotalVat, 'f', -1, 64),
	// 		Lottery:    resp.Lottery,
	// 		BillId:     resp.ID,
	// 		BillType:   string(resp.Type),
	// 		Image:      url,
	// 	}); err != nil {
	// 		fmt.Println("Send Ebarimt Mail Error : ", err.Error())
	// 	}
	// }(*order, resp)

	return &instance, nil
}

// * NOTE * : Step - 5 Save Ebarimt Stock and Extracting receipt item Data property for order_item_id
// func saveStock(db *gorm.DB, parent *structs.Ebarimt, receipts []structs.Receipt) error {

// 	for _, receipt := range receipts {

// 		for _, item := range receipt.Items {
// 			var data ItemData
// 			// Parse the data into a struct
// 			dataBytes, err := json.Marshal(item.Data)
// 			if err != nil {
// 				return fmt.Errorf("failed to marshal data: %w", err)
// 			}

// 			err = json.Unmarshal(dataBytes, &data)
// 			if err != nil {
// 				return fmt.Errorf("failed to unmarshal data: %w", err)
// 			}

// 			if err := db.Create(&databases.EbarimtStock{
// 				EbarimtID:          parent.ID,
// 				OrderItemID:        data.ItemID,
// 				Code:               item.Name,
// 				Name:               item.Name,
// 				MeasureUnit:        item.MeasureUnit,
// 				Qty:                utils.NumberToStr(item.Qty),
// 				UnitPrice:          utils.NumberToStr(item.UnitPrice),
// 				TotalAmount:        utils.NumberToStr(item.TotalAmount),
// 				CityTax:            utils.NumberToStr(item.TotalCityTax),
// 				Vat:                utils.NumberToStr(item.TotalVat),
// 				BarCode:            item.BarCode,
// 				ClassificationCode: item.ClassificationCode,
// 				TaxProductCode:     item.TaxProductCode,
// 				TaxType:            string(receipt.TaxType),
// 			}).Error; err != nil {
// 				return err
// 			}
// 		}

// 	}

// 	return nil
// }
