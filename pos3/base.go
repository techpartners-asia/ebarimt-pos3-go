package pos3

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"resty.dev/v3"
)

// defaultTimeout bounds every eBarimt HTTP call end to end. The upstream POS 3.0
// and gov services can stall indefinitely; without a client timeout a single
// stalled request blocks the caller (and any worker running it) until the socket
// is torn down. This ceiling, plus the finer per-phase deadlines in newTransport,
// fails fast so the caller can retry.
const defaultTimeout = 60 * time.Second

type pos3 struct {
	posEndpoint string
	apiKey      string
	token       *structs.TokenResponse
	merchanTin  string
	posNo       string
	isDev       bool
	client      *resty.Client
}

type ConnectionInput struct {
	PosEndpoint string
	ApiKey      string
	PosNo       string
	MerchantTin string
	IsDev       bool
	// TimeoutSeconds bounds each HTTP request end to end. Zero uses defaultTimeout (60s).
	TimeoutSeconds int
	// Client injects a preconfigured resty client (tests, custom transport or
	// certificates). Zero builds one from newTransport and the timeout above.
	Client *resty.Client
}

func New(input ConnectionInput) Pos3 {
	client := input.Client
	if client == nil {
		timeout := defaultTimeout
		if input.TimeoutSeconds > 0 {
			timeout = time.Duration(input.TimeoutSeconds) * time.Second
		}
		client = resty.New().
			SetTransport(newTransport()).
			SetTimeout(timeout).
			SetRedirectPolicy(resty.NoRedirectPolicy())
	}
	return &pos3{
		apiKey:      input.ApiKey,
		posEndpoint: input.PosEndpoint,
		merchanTin:  input.MerchantTin,
		posNo:       input.PosNo,
		isDev:       input.IsDev,
		client:      client,
	}
}

// newTransport returns an http.Transport with per-phase deadlines so a stalled
// connect, TLS handshake, or slow gov response cannot occupy the caller for the
// full client timeout. NoRedirectPolicy on the client surfaces a 3xx as a
// non-2xx error instead of silently following it (a redirect on POST /rest/receipt
// is never something to follow, and following one is what stalled callers before).
func newTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       20,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ForceAttemptHTTP2:     true,
	}
}

type Pos3 interface {
	// Get Inputs
	GetMerchantTin() string
	GetPosNo() string
	// Цахим төлбөрийн баримт
	GetInfo(customerTin string) (structs.GetInfoResponse, error)
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
