package pos3

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/techpartners-asia/ebarimt-go/utils"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
)

var (
	TokenAPI = utils.API{
		Url:    "https://auth.itc.gov.mn/auth/realms/ITC/protocol/openid-connect/token",
		Method: http.MethodPost,
	}

	// Нээлттэй API холболт
	GetBranchInfoAPI = utils.API{
		Url:    "/api/info/check/getBranchInfo",
		Method: http.MethodGet,
		IsAuth: false,
	}
	GetTinInfoAPI = utils.API{
		Url:    "/api/info/check/getTinInfo?regNo=",
		Method: http.MethodGet,
		IsAuth: false,
	}

	GetInfoAPI = utils.API{
		Url:    "/api/info/check/getInfo?tin=",
		Method: http.MethodGet,
		IsAuth: false,
	}

	// Pos API 3.0 холболт
	PosReceiptSendAPI = utils.API{
		Url:    "/rest/receipt",
		Method: http.MethodPost,
	}
	PosReceiptDeleteAPI = utils.API{
		Url:    "/rest/receipt",
		Method: http.MethodDelete,
	}
	PosSendAPI = utils.API{
		Url:    "/rest/sendData",
		Method: http.MethodGet,
	}
	PosInfoAPI = utils.API{
		Url:    "/rest/info",
		Method: http.MethodGet,
	}
	PosBankAccAPI = utils.API{
		Url:    "/rest/bankAccounts?",
		Method: http.MethodGet,
	}

	// Цахим төлбөрийн баримт API холболт
	GetSalesTotalAPI = utils.API{
		Url:    "https://api.ebarimt.mn/api/tpi/receipt/getSalesTotalData",
		Method: http.MethodPost,
		IsAuth: true,
	}
	GetSalesListERPAPI = utils.API{
		Url:    "https://api.ebarimt.mn/api/tpi/receipt/getSaleListERP",
		Method: http.MethodPost,
		IsAuth: true,
	}
	SaveOprMerchantsAPI = utils.API{
		Url:    "https://api.ebarimt.mn/api/tpi/receipt/%20saveOprMerchants",
		Method: http.MethodPost,
		IsAuth: true,
	}

	// Хялбар бүртгэл API холболт
	ConsumerInfoAPI = utils.API{
		Url:    "https://service.itc.gov.mn/api/easy-register/api/info/consumer/",
		Method: http.MethodGet,
		IsAuth: true,
	}
	GetProfileAPI = utils.API{
		Url:    "https://service.itc.gov.mn/api/easy-register/rest/v1/getProfile",
		Method: http.MethodPost,
		IsAuth: true,
	}
	ApproveQrAPI = utils.API{
		Url:    "https://service.itc.gov.mn/api/easy-register/rest/v1/approveQr",
		Method: http.MethodPost,
		IsAuth: true,
	}

	ForiegnerPassportInfoAPI = utils.API{
		Url:    "https://service.itc.gov.mn/api/easy-register/api/info/foreigner/",
		Method: http.MethodGet,
		IsAuth: true,
	}
	ForiegnerCustomerNoInfoAPI = utils.API{
		Url:    "https://service.itc.gov.mn/api/easy-register/api/info/foreigner/customerNo/",
		Method: http.MethodGet,
		IsAuth: true,
	}
	ForiegnerInfoRegAPI = utils.API{
		Url:    "https://service.itc.gov.mn/api/easy-register/api/info/foreigner/",
		Method: http.MethodPut,
		IsAuth: true,
	}

	// ОАТ API холболт
	GetInventoryListAPI = utils.API{
		Url:    "https://service.itc.gov.mn/rest/tpiMain/mainApi/getInventoryList",
		Method: http.MethodGet,
		IsAuth: false,
	}
	GetActiveStockNoPosAPI = utils.API{
		Url:    "https://service.itc.gov.mn/api/inventory/getActiveStockNoPos",
		Method: http.MethodPost,
		IsAuth: true,
	}
)

type CustomHeader struct {
	Name  string
	Value string
}

func (p *pos3) httpRequest(body interface{}, api utils.API, ext string, headers []CustomHeader) ([]byte, error) {

	var requestByte []byte
	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		requestByte, _ = json.Marshal(body)
		requestBody = bytes.NewReader(requestByte)
	}

	req, _ := http.NewRequest(api.Method, p.posEndpoint+api.Url+ext, requestBody)
	req.Header.Add("Accept", utils.HttpAcceptPublic)
	for _, header := range headers {
		req.Header.Add(header.Name, header.Value)
	}
	if api.IsAuth {
		token, err := p.auth()
		if err != nil {
			return nil, err
		}
		p.token = &token
		req.Header.Add("Authorization", "Bearer "+p.token.AccessToken)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, errors.New(string(response))
	}
	return response, nil
}

func (q *pos3) auth() (authRes structs.TokenResponse, err error) {
	if q.token != nil {
		expireInA, _ := time.Parse(time.RFC3339, q.token.ExpiresIn)
		expireInB := expireInA.Add(time.Duration(-12) * time.Hour)
		now := time.Now()
		if now.Before(expireInB) {
			authRes = *q.token
			err = nil
			return
		}
	}
	body := structs.TokenRequest{
		GrantType: "",
		Username:  "",
		Password:  "",
		ClientID:  "",
	}

	requestByte, _ := json.Marshal(body)
	requestBody := bytes.NewReader(requestByte)

	req, err := http.NewRequest(TokenAPI.Method, TokenAPI.Url, requestBody)
	if err != nil {
		fmt.Println(err.Error())
	}
	req.Header.Add("Accept", utils.HttpAcceptPrivate)
	req.Header.Add("Content-Type", utils.HttpContentType)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return authRes, fmt.Errorf("%s- Ebarimt POS 3.0 openid connect error response: %s", time.Now().Format(utils.TimeFormatYYYYMMDDHHMMSS), res.Status)
	}
	resp, _ := io.ReadAll(res.Body)
	json.Unmarshal(resp, &authRes)
	return authRes, nil
}

func (p *pos3) httpPosRequest(body interface{}, api utils.API, ext string, headers []CustomHeader) ([]byte, error) {

	var requestByte []byte
	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		requestByte, _ = json.Marshal(body)
		requestBody = bytes.NewReader(requestByte)
	}

	req, err := http.NewRequest(api.Method, p.posEndpoint+api.Url+ext, requestBody)
	if err != nil {
		return nil, err
	}
	// req.Header.Add("Accept", utils.HttpAcceptPublic)
	req.Header.Add("Content-type", utils.HttpAcceptPrivate)
	for _, header := range headers {
		req.Header.Add(header.Name, header.Value)
	}
	if api.IsAuth {
		token, err := p.auth()
		if err != nil {
			return nil, err
		}
		p.token = &token
		req.Header.Add("Authorization", "Bearer "+p.token.AccessToken)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	// if res.StatusCode != 200 {
	// 	return nil, errors.New(string(response))
	// }
	return response, nil
}
