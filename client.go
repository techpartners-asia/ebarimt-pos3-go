package ebarimtv3

import (
	"fmt"

	ebarimt3SdkServices "github.com/techpartners-asia/ebarimt-pos3-go/services"
	models "github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"github.com/techpartners-asia/ebarimt-pos3-go/utils"

	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	"github.com/techpartners-asia/ebarimt-pos3-go/pos3"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"gorm.io/gorm"
)

type (
	EbarimtClient struct {
		pos3.Pos3

		// * NOTE * : Optional & Integration To the Third Party
		DB               *gorm.DB
		MailHost         string
		MailPort         string
		MailFrom         string
		MailPassword     string
		StorageEndpoint  string
		StorageAccessKey string
		StorageSecretKey string
	}
	Input struct {
		Endpoint    string
		PosNo       string
		MerchantTin string

		// * NOTE * : Optional & Integration To the Third Party
		DB               *gorm.DB // Хоосон байж болно. Хэрвээ байвал, database дээр хадгална автоматаар
		MailHost         string
		MailPort         string
		MailFrom         string
		MailPassword     string
		StorageEndpoint  string
		StorageAccessKey string
		StorageSecretKey string
	}
)

func New(input Input) *EbarimtClient {
	posv3 := pos3.New(pos3.ConnectionInput{
		PosEndpoint: input.Endpoint,
		PosNo:       input.PosNo,
		MerchantTin: input.MerchantTin,
	})

	if input.DB != nil {
		ebarimt3SdkServices.Register(input.DB)
	}

	return &EbarimtClient{
		posv3,
		input.DB,
		input.MailHost,
		input.MailPort,
		input.MailFrom,
		input.MailPassword,
		input.StorageEndpoint,
		input.StorageAccessKey,
		input.StorageSecretKey,
	}
}

func (e *EbarimtClient) Create(input models.CreateInputModel) (*structs.ReceiptResponse, error) {

	// * NOTE * : Build Request For invoice and product
	request := e.buildRequest(input)

	// * NOTE * : Build RECEIPT ITEMS By Tax Type as Map
	receiptsItems, err := e.buildReceiptItemMap(input.Items)
	if err != nil {
		return nil, err
	}

	// * NOTE * : If has No VAT Items, Send First
	if len(receiptsItems) > 0 && len(receiptsItems[constants.TAX_NO_VAT].Items) > 0 {

		// * NOTE * : Build Receipts for Request Send
		e.buildReceipt(&request, map[constants.TaxType]structs.Receipt{
			constants.TAX_NO_VAT: receiptsItems[constants.TAX_NO_VAT],
		})

		delete(receiptsItems, constants.TAX_NO_VAT)
		// * NOTE * : Step - 4
		res, err := e.ReceiptSend(request)
		if err != nil {
			return nil, err
		}

		if res.Status != constants.POS_STATUS_SUCCESS {
			return nil, fmt.Errorf("Ebarimt Error: %v", res.Message)
		}

		fmt.Println("Ebarimt NO VAT RESPONSE", res)
	}

	// * NOTE * : Other Tax Types
	e.buildReceipt(&request, receiptsItems)

	// * NOTE * : Step - 4
	res, err := e.ReceiptSend(request)
	if err != nil {
		return nil, err
	}

	if res.Status != constants.POS_STATUS_SUCCESS {
		return nil, fmt.Errorf("Ebarimt Error: %v", res.Message)
	}

	fmt.Println("Ebarimt Other Tax Type RESPONSE", res)

	// * NOTE * : Step - 5 : Save Ebarimt to DB
	if e.DB != nil {
		ebarimt3SdkServices.SaveEbarimt(e.DB, &res)
	}

	if e.MailHost != "" && e.MailPort != "" && e.MailFrom != "" && e.MailPassword != "" && input.MailTo != "" {
		// * NOTE * : Step - 6 : Send Ebarimt to Mail
		// TODO : Send Ebarimt to Mail
		ebarimt3SdkServices.SendMail(
			ebarimt3SdkServices.EmailInput{
				Email:            input.MailTo,
				From:             e.MailFrom,
				Password:         e.MailPassword,
				SmtpHost:         e.MailHost,
				SmtpPort:         e.MailPort,
				StorageEndpoint:  e.StorageEndpoint,
				StorageAccessKey: e.StorageAccessKey,
				StorageSecretKey: e.StorageSecretKey,
				Response:         res,
			},
		)
	}

	return &res, nil
}

func (e *EbarimtClient) CalculateTotals(items []models.CreateItemInputModel) (*models.CalculateTotalsOutputModel, error) {

	var output models.CalculateTotalsOutputModel

	for _, item := range items {
		output.TotalVat += func() float64 {
			if item.TaxType == constants.TAX_VAT_ABLE {
				if item.IsCityTax {
					return utils.GetVatWithCityTax(item.TotalAmount)
				}
				return utils.GetVat(item.TotalAmount)
			}
			return 0
		}()

		output.TotalAmount += item.TotalAmount

		output.TotalCityTax += func() float64 {

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
		}()
	}

	return &output, nil
}
