package posapi

import (
	"encoding/json"
	"time"

	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
)

// Tax types keep their documented wire values
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787209763945).
type (
	// TaxType is a sub-receipt's tax class.
	TaxType string
	// ReceiptType is the receipt's buyer/document kind.
	ReceiptType string
	// PaymentCode is how the buyer paid.
	PaymentCode string
	// PaymentStatus is the state of one payment.
	PaymentStatus string
	// BarcodeType is the kind of an item's barcode.
	BarcodeType string
	// LocationType is the kind of a receipt location.
	LocationType string
)

const (
	TaxVATable      TaxType = "VAT_ABLE" // standard VAT
	TaxVATExempt    TaxType = "VAT_FREE" // exempt (VAT law, Art. 13)
	TaxVATZeroRated TaxType = "VAT_ZERO" // 0% (Art. 12)
	TaxOutsideVAT   TaxType = "NOT_VAT"  // outside VAT scope
)

const (
	B2CReceipt ReceiptType = "B2C_RECEIPT"
	B2BReceipt ReceiptType = "B2B_RECEIPT"
	B2CInvoice ReceiptType = "B2C_INVOICE"
	B2BInvoice ReceiptType = "B2B_INVOICE"
)

const (
	PaymentCash         PaymentCode = "CASH"
	PaymentCard         PaymentCode = "PAYMENT_CARD"
	PaymentBankTransfer PaymentCode = "BANK_TRANSFER"
	PaymentQPay         PaymentCode = "BANK_TRANSFER_QPAY"
)

const (
	PaymentPaid     PaymentStatus = "PAID"
	PaymentPay      PaymentStatus = "PAY"
	PaymentReversed PaymentStatus = "REVERSED"
	PaymentError    PaymentStatus = "ERROR"
)

const (
	BarcodeUndefined BarcodeType = "UNDEFINED"
	BarcodeGS1       BarcodeType = "GS1"
	BarcodeISBN      BarcodeType = "ISBN"
)

const (
	LocationGPS     LocationType = "GPS"
	LocationLicense LocationType = "LICENSE"
)

// StatusSuccess is the ReceiptResponse.Status of an accepted receipt.
const StatusSuccess = "SUCCESS"

// DateLayout is the timestamp layout PosAPI uses ("yyyy-MM-dd HH:mm:ss").
const DateLayout = "2006-01-02 15:04:05"

// Ulaanbaatar is the zone PosAPI timestamps are in (UTC+8, no daylight saving
// since 2017). A fixed zone, so no tzdata is needed on the host.
func Ulaanbaatar() *time.Location { return ulaanbaatar }

var ulaanbaatar = time.FixedZone("Asia/Ulaanbaatar", 8*60*60)

// ReceiptRequest is the body of POST /rest/receipt. Build one with
// receipt.Build rather than by hand.
//
// Field order and JSON tags are part of the contract: the marshalled bytes are
// pinned by golden files (receipt/testdata). Where the published documentation
// spells a field differently (totalVAT for totalVat, inactiveId for inActiveId)
// the spelling that production has proven is kept; switching is gated on a
// staging check (design issue #37, open question 1). PosAPI decodes field names
// case-insensitively on our side, so responses read either way.
type ReceiptRequest struct {
	TotalAmount  money.Amount `json:"totalAmount"`
	TotalVat     money.Amount `json:"totalVat"`
	TotalCityTax money.Amount `json:"totalCityTax"`
	BranchNo     string       `json:"branchNo"`
	DistrictCode string       `json:"districtCode"`
	MerchantTin  string       `json:"merchantTin"`
	PosNo        string       `json:"posNo"`
	CustomerTin  string       `json:"customerTin"`
	ConsumerNo   string       `json:"consumerNo"`
	Type         ReceiptType  `json:"type"`
	InActiveID   string       `json:"inActiveId"`
	InvoiceID    string       `json:"invoiceId"`
	ReportMonth  *string      `json:"reportMonth"`
	// BillIDSuffix is the optional per-day unique suffix of the receipt id. It
	// is sent as given and omitted when empty; this SDK never derives it.
	BillIDSuffix string    `json:"billIdSuffix,omitempty"`
	Data         any       `json:"data"`
	Receipts     []Receipt `json:"receipts"`
	Payments     []Payment `json:"payments"`
}

// Receipt is one sub-receipt of a ReceiptRequest (one per tax type).
type Receipt struct {
	ID            string       `json:"id"`
	BankAccountID int          `json:"bankAccountId"`
	TotalAmount   money.Amount `json:"totalAmount"`
	TotalVat      money.Amount `json:"totalVat"`
	TotalCityTax  money.Amount `json:"totalCityTax"`
	TaxType       TaxType      `json:"taxType"`
	MerchantTin   string       `json:"merchantTin"`
	CustomerTin   string       `json:"customerTin"`
	BankAccountNo string       `json:"bankAccountNo"`
	IBan          string       `json:"iBan"`
	Data          *ReceiptData `json:"data"`
	Items         []Item       `json:"items"`
}

// ReceiptData is a sub-receipt's additional data.
type ReceiptData struct {
	Location []Location `json:"location"`
}

// Location is where the receipt was issued, either by GPS or by licensed
// location (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787214708166).
type Location struct {
	Type             LocationType `json:"locationType"`
	Latitude         string       `json:"latitude,omitempty"`
	Longitude        string       `json:"longitude,omitempty"`
	Accuracy         string       `json:"accuracy,omitempty"`
	IPAddress        string       `json:"ipAddress,omitempty"`
	RegistrationCode int64        `json:"registrationCode,omitempty"`
	LicenseNo        []int64      `json:"licenseNo,omitempty"`
}

// Item is one sold product or service line.
type Item struct {
	Name               string         `json:"name"`
	BarCode            string         `json:"barCode"`
	BarCodeType        BarcodeType    `json:"barCodeType"`
	ClassificationCode string         `json:"classificationCode"`
	TaxProductCode     string         `json:"taxProductCode"`
	MeasureUnit        string         `json:"measureUnit"`
	Qty                money.Quantity `json:"qty"`
	UnitPrice          money.Amount   `json:"unitPrice"`
	TotalBonus         money.Amount   `json:"totalBonus"`
	TotalVat           money.Amount   `json:"totalVat"`
	TotalCityTax       money.Amount   `json:"totalCityTax"`
	TotalAmount        money.Amount   `json:"totalAmount"`
	Data               *ItemData      `json:"data"`
}

// ItemData is an item's additional data.
type ItemData struct {
	LotNo   string   `json:"lotNo,omitempty"`
	StockQR []string `json:"stockQR,omitempty"`
}

// Payment is how a receipt was paid.
type Payment struct {
	Code         PaymentCode   `json:"code"`
	ExchangeCode string        `json:"exchangeCode"`
	Status       PaymentStatus `json:"status"`
	PaidAmount   money.Amount  `json:"paidAmount"`
	Data         any           `json:"data"`
}

// ReceiptResponse is the body PosAPI answers POST /rest/receipt with, as it
// appears on the wire. Callers get the friendlier IssuedReceipt; this type is
// exported for fakes (package postest). QrData and Lottery are secrets of the
// issued receipt: the spec forbids storing them anywhere but the printed
// receipt.
type ReceiptResponse struct {
	ID           string       `json:"id"`
	Version      string       `json:"version"`
	TotalAmount  money.Amount `json:"totalAmount"`
	TotalVat     money.Amount `json:"totalVat"`
	TotalCityTax money.Amount `json:"totalCityTax"`
	BranchNo     string       `json:"branchNo"`
	DistrictCode string       `json:"districtCode"`
	MerchantTin  string       `json:"merchantTin"`
	PosNo        string       `json:"posNo"`
	CustomerTin  string       `json:"customerTin"`
	ConsumerNo   string       `json:"consumerNo"`
	Type         ReceiptType  `json:"type"`
	InvoiceID    string       `json:"invoiceId"`
	Receipts     []Receipt    `json:"receipts"`
	Payments     []Payment    `json:"payments"`
	PosID        float64      `json:"posId"`
	Status       string       `json:"status"`
	Message      string       `json:"message"`
	QrData       string       `json:"qrData"`
	Lottery      string       `json:"lottery"`
	Date         string       `json:"date"`
	Easy         bool         `json:"easy"`
}

// IssuedReceipt is an accepted receipt. QRData and Lottery exist to be printed
// on the customer's receipt and nowhere else: do not store or log them.
type IssuedReceipt struct {
	ID           string // 33-digit receipt id (ДДТД)
	Version      string
	Type         ReceiptType
	TotalAmount  money.Amount
	TotalVAT     money.Amount
	TotalCityTax money.Amount
	BranchNo     string
	DistrictCode string
	MerchantTIN  string
	PosNo        string
	CustomerTIN  string
	ConsumerNo   string
	// Receipts are the sub-receipts; their IDs are the per-tax-type receipt ids.
	Receipts []Receipt
	Payments []Payment
	PosID    int64
	// Date is when PosAPI filed the receipt, in Ulaanbaatar time; zero if the
	// daemon's value was in an unrecognised layout (see DateText).
	Date     time.Time
	DateText string
	Easy     bool // filed through easy registration; paper receipt optional
	QRData   string
	Lottery  string
}

// deleteRequest is the body of DELETE /rest/receipt.
type deleteRequest struct {
	ID   string `json:"id"`
	Date string `json:"date"`
}

// DaemonStatus is the answer of GET /rest/info.
type DaemonStatus struct {
	OperatorName string
	OperatorTIN  string
	PosID        int64
	PosNo        string
	// LastSentAt is when the daemon last sent its backlog to the tax authority;
	// zero if it never did or the value was in an unrecognised layout.
	LastSentAt    time.Time
	LotteriesLeft int
	Merchants     []DaemonMerchant
}

// DaemonMerchant is a business registered in the daemon.
type DaemonMerchant struct {
	Name      string `json:"name"`
	TIN       string `json:"tin"`
	Customers []DaemonCustomer
}

// DaemonCustomer is a buyer business known to a DaemonMerchant.
type DaemonCustomer struct {
	Name     string `json:"name"`
	TIN      string `json:"tin"`
	VATPayer bool   `json:"vatPayer"`
}

// UnmarshalJSON decodes the daemon's /rest/info document.
func (s *DaemonStatus) UnmarshalJSON(b []byte) error {
	var w struct {
		OperatorName  string  `json:"operatorName"`
		OperatorTIN   string  `json:"operatorTIN"`
		PosID         float64 `json:"posId"`
		PosNo         string  `json:"posNo"`
		LastSentDate  string  `json:"lastSentDate"`
		LeftLotteries int     `json:"leftLotteries"`
		Merchants     []struct {
			Name      string           `json:"name"`
			TIN       string           `json:"tin"`
			Customers []DaemonCustomer `json:"customers"`
		} `json:"merchants"`
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	*s = DaemonStatus{
		OperatorName:  w.OperatorName,
		OperatorTIN:   w.OperatorTIN,
		PosID:         int64(w.PosID),
		PosNo:         w.PosNo,
		LastSentAt:    ParseTime(w.LastSentDate),
		LotteriesLeft: w.LeftLotteries,
	}
	for _, m := range w.Merchants {
		s.Merchants = append(s.Merchants, DaemonMerchant{Name: m.Name, TIN: m.TIN, Customers: m.Customers})
	}
	return nil
}

// BankAccount is a bank account registered for a merchant in the daemon.
type BankAccount struct {
	ID          int64  `json:"id"`
	TIN         string `json:"tin"`
	AccountNo   string `json:"bankAccountNo"`
	AccountName string `json:"bankAccountName"`
	BankID      int64  `json:"bankId"`
	IBan        string `json:"iBan"`
	BankName    string `json:"bankName"`
}

// timeLayouts are the layouts PosAPI timestamps have been seen in.
var timeLayouts = []string{DateLayout, "2006-01-02T15:04:05", time.RFC3339, "2006-01-02"}

// ParseTime reads a PosAPI timestamp, assuming Ulaanbaatar time when the value
// carries no zone. It returns the zero time for "" or an unrecognised layout.
func ParseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, l := range timeLayouts {
		if t, err := time.ParseInLocation(l, s, ulaanbaatar); err == nil {
			return t
		}
	}
	return time.Time{}
}
