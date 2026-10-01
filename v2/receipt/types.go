package receipt

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/posapi"
)

// The wire vocabulary is defined once, in posapi, and re-exported here so a
// caller building a Sale only imports receipt.
type (
	TaxType      = posapi.TaxType
	ReceiptType  = posapi.ReceiptType
	PaymentCode  = posapi.PaymentCode
	BarcodeType  = posapi.BarcodeType
	LocationType = posapi.LocationType
	// Location is where the receipt was issued, GPS or LICENSE
	// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787214708166).
	Location = posapi.Location
)

const (
	// TaxVATable is standard VAT (wire VAT_ABLE).
	TaxVATable = posapi.TaxVATable
	// TaxVATExempt is exempt from VAT, VAT law Art. 13 (wire VAT_FREE).
	TaxVATExempt = posapi.TaxVATExempt
	// TaxVATZeroRated is 0% VAT, VAT law Art. 12 (wire VAT_ZERO).
	TaxVATZeroRated = posapi.TaxVATZeroRated
	// TaxOutsideVAT is outside the VAT scope (wire NOT_VAT).
	TaxOutsideVAT = posapi.TaxOutsideVAT
)

const (
	B2CReceipt = posapi.B2CReceipt
	B2BReceipt = posapi.B2BReceipt
	B2CInvoice = posapi.B2CInvoice
	B2BInvoice = posapi.B2BInvoice
)

const (
	PaymentCash         = posapi.PaymentCash
	PaymentCard         = posapi.PaymentCard
	PaymentBankTransfer = posapi.PaymentBankTransfer
	PaymentQPay         = posapi.PaymentQPay
)

const (
	BarcodeGS1       = posapi.BarcodeGS1
	BarcodeISBN      = posapi.BarcodeISBN
	BarcodeUndefined = posapi.BarcodeUndefined
)

const (
	LocationGPS     = posapi.LocationGPS
	LocationLicense = posapi.LocationLicense
)

// Sale is everything about one sale that Build needs. Amounts are whole receipt
// totals with every tax included.
type Sale struct {
	Type  ReceiptType
	Buyer Buyer
	Items []Item
	// Payments are required and must sum to the sale's total.
	Payments []Payment
	// PosNo is the merchant's internal cash register number (wire posNo).
	PosNo        string
	BranchNo     string
	DistrictCode string // from registry.Districts
	// BillIDSuffix is the optional per-day unique suffix of the receipt id
	// (wire billIdSuffix). It is sent as given and omitted when empty; this
	// SDK does not derive it.
	BillIDSuffix string
	// ReportMonth back-dates a receipt into the previous month. Only B2B
	// receipts and invoices; PosAPI accepts it only on days 1-7 of a month
	// (release notes v3.1.82), which a pure Build cannot check.
	ReportMonth *Month
	// ReplacesReceiptID is the id of the receipt this one corrects or
	// partially returns (wire inActiveId). In a chain of corrections it is the
	// previous receipt's id.
	ReplacesReceiptID string
	// Location is GPS or LICENSE location data (release notes v3.2.48),
	// attached to every sub-receipt.
	Location *Location
}

// Buyer identifies the other side of the sale.
type Buyer struct {
	TIN        string // B2B types: the buyer's taxpayer number
	ConsumerNo string // B2CReceipt only: the citizen's 8-digit ebarimt number
}

// Item is one sold line.
type Item struct {
	Name        string
	Barcode     string
	BarcodeType BarcodeType // empty means BarcodeUndefined
	// ClassificationCode is the 7-digit БҮНА code, required for every item.
	ClassificationCode string
	TaxType            TaxType
	// TaxReliefCode is the legal basis for relief (wire taxProductCode):
	// required for VAT_FREE, VAT_ZERO and NOT_VAT, empty for VAT_ABLE. List the
	// valid codes with registry.TaxReliefCodes.
	TaxReliefCode string
	// MeasureUnit is the unit of Quantity (wire measureUnit).
	MeasureUnit string
	HasCityTax  bool
	Quantity    decimal.Decimal
	// Total is the line total with all taxes included.
	Total          money.Amount
	ExciseStampQRs []string // wire stockQR, for excise-stamped goods
	MedicineLotNo  string   // wire lotNo
}

// Payment is one way the sale was paid.
type Payment struct {
	Code         PaymentCode
	Amount       money.Amount
	ExchangeCode string // third-party payment system code, optional
}

// Merchant is the seller's tax registration, as registry.Taxpayer reports it.
type Merchant struct {
	TIN          string
	VATPayer     bool // vatPayer
	CityTaxPayer bool // cityPayer: only Ulaanbaatar branches
	// VATFreeProject marks a project exempt from VAT (freeProject): every item
	// is then sent as VAT_FREE with relief code "304".
	VATFreeProject bool
}

// Month is a calendar month.
type Month struct {
	Year  int
	Month time.Month
}

// String is the wire form: the first day of the month, "2006-01-02".
func (m Month) String() string { return fmt.Sprintf("%04d-%02d-01", m.Year, int(m.Month)) }
