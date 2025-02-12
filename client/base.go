package ebarimt3Sdk

import (
	"fmt"

	models "github.com/techpartners-asia/ebarimt-pos3-go/client/models"
	ebarimt3SdkServices "github.com/techpartners-asia/ebarimt-pos3-go/client/services"

	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	"github.com/techpartners-asia/ebarimt-pos3-go/pos3"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"gorm.io/gorm"
)

type (
	EbarimtClient struct {
		pos3.Pos3

		// * NOTE * : Optional & Integration To the Third Party
		DB       *gorm.DB
		MailHost string
		MailPort int
	}
	Input struct {
		Endpoint    string
		PosNo       string
		MerchantTin string

		// * NOTE * : Optional & Integration To the Third Party
		DB       *gorm.DB
		MailHost string
		MailPort int
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
	}
}

func (e *EbarimtClient) Create(input models.CreateInputModel) (interface{}, error) {

	// * NOTE * : Build Request For invoice and product
	request := e.buildRequest(input)

	// * NOTE * : Build RECEIPT ITEMS By Tax Type as Map
	receiptsItems := e.buildReceiptItemMap(input.Items)

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

	go func(e *EbarimtClient, res structs.ReceiptResponse) {
		// * NOTE * : Step - 5 : Save Ebarimt to DB
		if e.DB != nil {
			ebarimt3SdkServices.SaveEbarimt(e.DB, &res)
		}

		if e.MailHost != "" && e.MailPort != 0 {
			// * NOTE * : Step - 6 : Send Ebarimt to Mail
			// TODO : Send Ebarimt to Mail
		}

	}(e, res)

	return nil, nil
}
