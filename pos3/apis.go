package pos3

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"github.com/techpartners-asia/ebarimt-pos3-go/utils"
	"resty.dev/v3"
)

var (
	TokenAPI = utils.API{
		Url:    "https://auth.itc.gov.mn/auth/realms/ITC/protocol/openid-connect/token",
		Method: http.MethodPost,
	}

	// Нээлттэй API холболт
	GetBranchInfoAPI = utils.API{
		Url:    "https://ebarimt.techpartners.asia/api/info/check/getBranchInfo",
		Method: http.MethodGet,
		IsAuth: false,
		DevUrl: "https://st-api.ebarimt.mn/api/info/check/getBranchInfo",
	}
	GetTinInfoAPI = utils.API{
		Url:    "https://ebarimt.techpartners.asia/api/info/check/getTinInfo?regNo=",
		Method: http.MethodGet,
		IsAuth: false,
		DevUrl: "https://st-api.ebarimt.mn/api/info/check/getTinInfo?regNo=",
	}

	GetInfoAPI = utils.API{
		Url:    "https://ebarimt.techpartners.asia/api/info/check/getInfo?tin=",
		Method: http.MethodGet,
		IsAuth: false,
		DevUrl: "https://st-api.ebarimt.mn/api/info/check/getInfo?tin=",
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
		Url:    "https://ebarimt.techpartners.asia/api/tpi/receipt/getSalesTotalData",
		Method: http.MethodPost,
		IsAuth: true,
	}
	GetSalesListERPAPI = utils.API{
		Url:    "https://ebarimt.techpartners.asia/api/tpi/receipt/getSaleListERP",
		Method: http.MethodPost,
		IsAuth: true,
	}
	SaveOprMerchantsAPI = utils.API{
		Url:    "https://ebarimt.techpartners.asia/api/tpi/receipt/%20saveOprMerchants",
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
	url := api.Url + ext
	if p.isDev && len(api.DevUrl) > 0 {
		url = api.DevUrl + ext
	}

	req := p.client.R().SetHeader("Accept", utils.HttpAcceptPublic)
	for _, header := range headers {
		req.SetHeader(header.Name, header.Value)
	}
	if body != nil {
		req.SetBody(body)
	}
	if api.IsAuth {
		token, err := p.auth()
		if err != nil {
			return nil, err
		}
		p.token = &token
		req.SetAuthToken(p.token.AccessToken)
	}

	res, err := req.Execute(api.Method, url)
	if err != nil {
		return nil, err
	}
	defer closeBody(res)

	if !res.IsSuccess() {
		return nil, errors.New(res.String())
	}
	return res.Bytes(), nil
}

func (q *pos3) auth() (authRes structs.TokenResponse, err error) {
	if q.token != nil {
		expireInA, _ := time.Parse(time.RFC3339, q.token.ExpiresIn)
		expireInB := expireInA.Add(time.Duration(-12) * time.Hour)
		if time.Now().Before(expireInB) {
			return *q.token, nil
		}
	}
	body := structs.TokenRequest{
		GrantType: "",
		Username:  "",
		Password:  "",
		ClientID:  "",
	}

	res, err := q.client.R().
		SetHeader("Accept", utils.HttpAcceptPrivate).
		SetHeader("Content-Type", utils.HttpContentType).
		SetBody(body).
		Execute(TokenAPI.Method, TokenAPI.Url)
	if err != nil {
		return authRes, err
	}
	defer closeBody(res)

	if !res.IsSuccess() {
		return authRes, fmt.Errorf("%s- Ebarimt POS 3.0 openid connect error response: %s", time.Now().Format(utils.TimeFormatYYYYMMDDHHMMSS), res.Status())
	}
	if err := json.Unmarshal(res.Bytes(), &authRes); err != nil {
		return authRes, err
	}
	return authRes, nil
}

func (p *pos3) httpPosRequest(body interface{}, api utils.API, ext string, headers []CustomHeader) ([]byte, error) {
	req := p.client.R().SetHeader("Content-type", utils.HttpAcceptPrivate)
	for _, header := range headers {
		req.SetHeader(header.Name, header.Value)
	}
	if body != nil {
		req.SetBody(body)
	}
	if api.IsAuth {
		token, err := p.auth()
		if err != nil {
			return nil, err
		}
		p.token = &token
		req.SetAuthToken(p.token.AccessToken)
	}

	res, err := req.Execute(api.Method, p.posEndpoint+api.Url+ext)
	if err != nil {
		return nil, err
	}
	defer closeBody(res)

	// The POS 3.0 endpoint returns meaningful bodies on non-2xx statuses that
	// callers parse, so the status itself is intentionally not treated as an error.
	return res.Bytes(), nil
}

// closeBody releases a response body. Resty drains and closes it while decoding
// 2xx and >=400 responses, but not for 204 or 3xx, and the body's Close is what
// cancels the per-request timeout context.
func closeBody(res *resty.Response) {
	if res != nil && res.Body != nil {
		_ = res.Body.Close()
	}
}
