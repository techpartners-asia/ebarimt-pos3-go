package pos3

import "github.com/techpartners-asia/ebarimt-pos3-go/structs"

type pos3 struct {
	posEndpoint string
	apiKey      string
	posNo       string
	merchantTin string
	token       *structs.TokenResponse
}

type ConnectionInput struct {
	PosEndpoint string
	ApiKey      string
	PosNo       string
	MerchantTin string
}

func New(input ConnectionInput) Pos3 {
	return &pos3{
		apiKey:      input.ApiKey,
		posEndpoint: input.PosEndpoint,
		posNo:       input.PosNo,
		merchantTin: input.MerchantTin,
	}
}

type Pos3 interface {
	// Цахим төлбөрийн баримт
	GetInfo(regNo string) (structs.GetInfoResponse, error)
	GetTinInfo(regNo string) (structs.GetTinInfoResponse, error)
	GetBranchInfo() (structs.GetBranchInfoResponse, error)
	GetSalesTotalData(body structs.GetSalesTotalDataRequest) (structs.GetSalesTotalDataResponse, error)
	GetSalesListERP(body structs.GetSalesListERPRequest) (structs.GetSalesTotalDataResponse, error)
	SaveOprMerchants(body structs.SaveOprMerchantsRequest) (structs.SaveOprMerchantsResponse, error)
	// хялбар бүртгэл
	ConsumerInfo(regNo string) (structs.ConsumerInfoResponse, error)
	GetProfile(body structs.GetProfileRequest) (structs.GetProfileResponse, error)
	ApproveQr(body structs.ApproveQrRequest) (structs.ApproveQrResponse, error)
	ForiegnerPassportInfo(fNumber, passportNo string) (structs.ForiegnerInfoResponse, error)
	ForiegnerCustomerNoInfo(loginName string) (structs.ForiegnerInfoResponse, error)
	ForiegnerInfoRegister(passportNo string, body structs.ForiegnerInfoRequest) (structs.ForiegnerInfoResponse, error)

	// POS
	ReceiptSend(body structs.ReceiptRequest) (structs.ReceiptResponse, error)
	ReceiptDelete(body structs.ReceiptDeleteRequest) (structs.Response, error)
	SendData() (structs.Response, error)
	Info() (structs.InfoResponse, error)
	BankAccounts(tin string) ([]structs.BankAccountData, error)
}
