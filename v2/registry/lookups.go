package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/internal/httpx"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/posapi"
)

// Taxpayer is a taxpayer's registration as the tax authority reports it.
type Taxpayer struct {
	Name         string
	VATPayer     bool // vatPayer: withholds VAT
	CityTaxPayer bool // cityPayer: withholds city tax (Ulaanbaatar branches only)
	// VATFreeProject is true for a project exempt from VAT (freeProject): its
	// receipts are sent as VAT_FREE with relief code "304".
	VATFreeProject         bool
	IsGovernment           bool
	VATPayerRegisteredDate string
}

// Taxpayer reads a taxpayer's registration by TIN:
// GET /api/info/check/getInfo?tin=
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787213583668).
// An unknown TIN (found=false) is ErrNotFound.
func (c *Client) Taxpayer(ctx context.Context, tin string) (Taxpayer, error) {
	const op = "registry.Taxpayer"
	if tin == "" {
		return Taxpayer{}, invalid(op, "tin", "required")
	}
	raw, err := c.lookup(ctx, op, "/api/info/check/getInfo", url.Values{"tin": {tin}})
	if err != nil {
		return Taxpayer{}, err
	}
	var d struct {
		Name                   string `json:"name"`
		FreeProject            bool   `json:"freeProject"`
		CityPayer              bool   `json:"cityPayer"`
		VatPayer               bool   `json:"vatPayer"`
		Found                  bool   `json:"found"`
		IsGovernment           bool   `json:"isGovernment"`
		VatPayerRegisteredDate string `json:"vatpayerRegisteredDate"`
	}
	if !isNull(raw) {
		if err := json.Unmarshal(raw, &d); err != nil {
			return Taxpayer{}, httpx.Undecodable(op, http.StatusOK, err)
		}
	}
	if !d.Found {
		return Taxpayer{}, &ebarimt.Error{Kind: ebarimt.NotFound, Op: op, HTTPStatus: http.StatusOK, Message: "taxpayer not found"}
	}
	return Taxpayer{
		Name: d.Name, VATPayer: d.VatPayer, CityTaxPayer: d.CityPayer, VATFreeProject: d.FreeProject,
		IsGovernment: d.IsGovernment, VATPayerRegisteredDate: d.VatPayerRegisteredDate,
	}, nil
}

// TINByRegisterNumber resolves a register number to a TIN:
// GET /api/info/check/getTinInfo?regNo=
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787213499347).
//
// An unknown register number is ErrNotFound. The service answers it with
// HTTP 200 and {"msg":"Хэрэглэгчийн мэдээлэл олдсонгүй.","status":500,"data":null}
// (captured live), so an envelope with a non-200 status and no data is
// NotFound, with the envelope's own Status and Message kept on the error. The
// docs type "data" as a string but production returns a number; both are read.
func (c *Client) TINByRegisterNumber(ctx context.Context, regNo string) (string, error) {
	const op = "registry.TINByRegisterNumber"
	if regNo == "" {
		return "", invalid(op, "regNo", "required")
	}
	res, err := c.get(ctx, op, "/api/info/check/getTinInfo", url.Values{"regNo": {regNo}}, false)
	if err != nil {
		return "", err
	}
	env, status, err := unwrap(op, res)
	if err != nil {
		return "", err
	}
	tin := httpx.RawString(env.Data)
	n, convErr := strconv.ParseInt(tin, 10, 64)
	if convErr == nil && n > 0 && status == 200 {
		return tin, nil
	}
	e := rejected(op, res, env)
	if convErr != nil || n <= 0 {
		// No TIN in the answer. The service reports an unknown register number
		// as status 500 with null data (and has been seen using 200 with data 0
		// and 400 with a "not registered" message): that is "no such subject".
		switch status {
		case 200, 400, 404, 500:
			e.Kind = ebarimt.NotFound
		}
	}
	return "", e
}

// District is a tax office and sub-office, the value space of a receipt's
// districtCode.
type District struct {
	BranchCode    string
	BranchName    string
	SubBranchCode string
	SubBranchName string
}

// Code is the 4-digit districtCode: branch code followed by sub-branch code
// ("01"+"02" for Arkhangai/Chuluut is "0102").
func (d District) Code() string { return d.BranchCode + d.SubBranchCode }

// Districts lists the district codes: GET /api/info/check/getBranchInfo
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787213357413).
func (c *Client) Districts(ctx context.Context) ([]District, error) {
	const op = "registry.Districts"
	raw, err := c.lookup(ctx, op, "/api/info/check/getBranchInfo", nil)
	if err != nil {
		return nil, err
	}
	type row struct {
		BranchCode    string `json:"branchCode"`
		BranchName    string `json:"branchName"`
		SubBranchCode string `json:"subBranchCode"`
		SubBranchName string `json:"subBranchName"`
	}
	var rows []row
	switch {
	case isNull(raw):
	case strings.HasPrefix(strings.TrimSpace(string(raw)), "["):
		err = json.Unmarshal(raw, &rows)
	default: // the docs' example shows a single object
		var one row
		if err = json.Unmarshal(raw, &one); err == nil {
			rows = []row{one}
		}
	}
	if err != nil {
		return nil, httpx.Undecodable(op, http.StatusOK, err)
	}
	out := make([]District, len(rows))
	for i, r := range rows {
		out[i] = District(r)
	}
	return out, nil
}

// TaxReliefCode is a legal basis for VAT exemption or 0% VAT, the value space
// of an item's taxProductCode.
type TaxReliefCode struct {
	Code      string
	Name      string
	TaxType   posapi.TaxType // VAT_FREE or VAT_ZERO, from taxTypeName
	StartDate string         // "2006-01-02"
	EndDate   string         // empty while in force
}

// TaxReliefCodes lists the relief codes:
// GET /api/receipt/receipt/getProductTaxCode
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787213804128).
func (c *Client) TaxReliefCodes(ctx context.Context) ([]TaxReliefCode, error) {
	const op = "registry.TaxReliefCodes"
	raw, err := c.lookup(ctx, op, "/api/receipt/receipt/getProductTaxCode", nil)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		StartDate      string `json:"startDate"`
		EndDate        string `json:"endDate"`
		TaxProductCode string `json:"taxProductCode"`
		TaxProductName string `json:"taxProductName"`
		TaxTypeName    string `json:"taxTypeName"`
	}
	if !isNull(raw) {
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, httpx.Undecodable(op, http.StatusOK, err)
		}
	}
	out := make([]TaxReliefCode, len(rows))
	for i, r := range rows {
		out[i] = TaxReliefCode{Code: r.TaxProductCode, Name: r.TaxProductName, TaxType: posapi.TaxType(r.TaxTypeName),
			StartDate: r.StartDate, EndDate: r.EndDate}
	}
	return out, nil
}

// ClassificationNode is one entry of the product classification (БҮНА) tree.
type ClassificationNode struct {
	Code string
	Name string
}

// ProductClassification walks the БҮНА tree one level at a time:
// GET /api/info/check/barcode/v2[/code...]
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787214072304).
//
// With no path it returns the top level (sectors). Each code of path descends
// one level, in order: sector, sub-sector, group, class, sub-class, БҮНА code
// (at most six). The service answers with a bare JSON array of [code, name,
// ...] rows; only the first two columns are read.
//
// The docs print the URL template with a different parameter order
// ({p4}/{p5}/{p1}/{p2}/{p3}/{p6}) than their description of the walk, and the
// walk could not be tried against the live service when this was written:
// this follows the description. Verify before relying on levels below sector.
func (c *Client) ProductClassification(ctx context.Context, path ...string) ([]ClassificationNode, error) {
	const op = "registry.ProductClassification"
	if len(path) > 6 {
		return nil, invalid(op, "path", "at most 6 levels")
	}
	target := "/api/info/check/barcode/v2"
	for i, p := range path {
		if p == "" || strings.ContainsAny(p, "/?#") {
			return nil, invalid(op, "path["+strconv.Itoa(i)+"]", "must be a non-empty code")
		}
		target += "/" + url.PathEscape(p)
	}
	res, err := c.get(ctx, op, target, nil, false)
	if err != nil {
		return nil, err
	}
	var rows [][]json.RawMessage
	if err := json.Unmarshal(res.Body, &rows); err != nil {
		return nil, httpx.Undecodable(op, res.Status, err)
	}
	out := make([]ClassificationNode, 0, len(rows))
	for _, r := range rows {
		var n ClassificationNode
		if len(r) > 0 {
			n.Code = httpx.RawString(r[0])
		}
		if len(r) > 1 {
			n.Name = httpx.RawString(r[1])
		}
		out = append(out, n)
	}
	return out, nil
}

// Page selects a page of RegisteredBarcodes.
type Page struct {
	Number int       // zero-based
	Size   int       // 1..200
	Since  time.Time // only barcodes registered or changed on or after this date; zero means all
}

// Barcode is one registered product barcode.
type Barcode struct {
	Barcode            string
	Name               string
	ChangedOn          string // date registered or last changed, "2006-01-02"
	ClassificationCode string // 7-digit БҮНА code
	ExciseStamp        bool   // carries an excise stamp: sell it with stockQR
}

// BarcodePage is one page of RegisteredBarcodes.
type BarcodePage struct {
	Items         []Barcode
	Number        int
	Size          int
	TotalPages    int
	TotalElements int
	Last          bool
}

// MaxBarcodePageSize is the largest Page.Size the service accepts.
const MaxBarcodePageSize = 200

// RegisteredBarcodes pages through the national product register:
// GET /api/info/check/barcode/all?page=&size=&date=
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787214227405).
func (c *Client) RegisteredBarcodes(ctx context.Context, p Page) (BarcodePage, error) {
	const op = "registry.RegisteredBarcodes"
	if p.Number < 0 {
		return BarcodePage{}, invalid(op, "page.number", "must not be negative")
	}
	if p.Size < 1 || p.Size > MaxBarcodePageSize {
		return BarcodePage{}, invalid(op, "page.size", "must be between 1 and 200")
	}
	since := "1970-01-01"
	if !p.Since.IsZero() {
		since = p.Since.Format("2006-01-02")
	}
	res, err := c.get(ctx, op, "/api/info/check/barcode/all", url.Values{
		"page": {strconv.Itoa(p.Number)}, "size": {strconv.Itoa(p.Size)}, "date": {since},
	}, false)
	if err != nil {
		return BarcodePage{}, err
	}
	var d struct {
		Content       [][]json.RawMessage `json:"content"`
		Number        int                 `json:"number"`
		Size          int                 `json:"size"`
		TotalPages    int                 `json:"totalPages"`
		TotalElements int                 `json:"totalElements"`
		Last          bool                `json:"last"`
	}
	if err := json.Unmarshal(res.Body, &d); err != nil {
		return BarcodePage{}, httpx.Undecodable(op, res.Status, err)
	}
	page := BarcodePage{Number: d.Number, Size: d.Size, TotalPages: d.TotalPages, TotalElements: d.TotalElements, Last: d.Last}
	for _, r := range d.Content {
		var b Barcode
		if len(r) > 0 {
			b.Barcode = httpx.RawString(r[0])
		}
		if len(r) > 1 {
			b.Name = httpx.RawString(r[1])
		}
		if len(r) > 2 {
			b.ChangedOn = httpx.RawString(r[2])
		}
		if len(r) > 3 {
			b.ClassificationCode = httpx.RawString(r[3])
		}
		if len(r) > 4 {
			var flags []string
			if json.Unmarshal(r[4], &flags) == nil {
				for _, f := range flags {
					if f == "stockQR" {
						b.ExciseStamp = true
					}
				}
			}
		}
		page.Items = append(page.Items, b)
	}
	return page, nil
}

// MerchantLocation is a registered place of business, the source of a
// receipt's GPS or LICENSE location.
type MerchantLocation struct {
	MerchantRegisterNumber string
	RegistrationCode       int64
	Longitude              string
	Latitude               string
	StoreName              string
	Address                string
	IsCityTaxPayer         bool
	Licenses               []License
}

// License is a permit held at a MerchantLocation.
type License struct {
	LicenseNo           int64
	OwnerRegisterNumber string
	OwnerName           string
	GrantType           string
	ApprovedAt          string // "2006-01-02 15:04:05"
	TerminatesAt        string // "2006-01-02 15:04:05"
}

// MerchantLocations lists the registered locations of a merchant at a POS:
// GET /api/info/cityTax/location?mrchRegno=&posNo=
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787214708166).
//
// It needs Config.APIKey (X-API-KEY, issued to software suppliers), and the
// merchant and tenant must be registered with that supplier and with the POS.
// The service is keyed by the merchant's register number and the POS number,
// not by TIN.
func (c *Client) MerchantLocations(ctx context.Context, merchantRegisterNumber, posNo string) ([]MerchantLocation, error) {
	const op = "registry.MerchantLocations"
	var v []ebarimt.Violation
	if c.apiKey == "" {
		v = append(v, ebarimt.Violation{Field: "Config.APIKey", Rule: "required for this service"})
	}
	if merchantRegisterNumber == "" {
		v = append(v, ebarimt.Violation{Field: "merchantRegisterNumber", Rule: "required"})
	}
	if posNo == "" {
		v = append(v, ebarimt.Violation{Field: "posNo", Rule: "required"})
	}
	if len(v) > 0 {
		return nil, &ebarimt.Error{Kind: ebarimt.Invalid, Op: op, Violations: v}
	}
	res, err := c.get(ctx, op, "/api/info/cityTax/location", url.Values{"mrchRegno": {merchantRegisterNumber}, "posNo": {posNo}}, true)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		MrchRegno        string `json:"mrchRegno"`
		RegistrationCode int64  `json:"registrationCode"`
		Longitude        string `json:"longitude"`
		Latitude         string `json:"latitude"`
		StoreName        string `json:"storeName"`
		Address          string `json:"address"`
		IsCityTaxPayer   bool   `json:"isCityTaxPayer"`
		Licenses         []struct {
			LicenseNo         int64  `json:"licenseNo"`
			OwnerRegno        string `json:"ownerRegno"`
			OwnerName         string `json:"ownerName"`
			GrantType         string `json:"grantType"`
			GrantApprveDate   string `json:"grantApprveDate"`
			GrantTrminateDate string `json:"grantTrminateDate"`
		} `json:"licenses"`
	}
	if err := json.Unmarshal(res.Body, &rows); err != nil {
		return nil, httpx.Undecodable(op, res.Status, err)
	}
	out := make([]MerchantLocation, len(rows))
	for i, r := range rows {
		l := MerchantLocation{
			MerchantRegisterNumber: r.MrchRegno, RegistrationCode: r.RegistrationCode, Longitude: r.Longitude,
			Latitude: r.Latitude, StoreName: r.StoreName, Address: r.Address, IsCityTaxPayer: r.IsCityTaxPayer,
		}
		for _, x := range r.Licenses {
			l.Licenses = append(l.Licenses, License{LicenseNo: x.LicenseNo, OwnerRegisterNumber: x.OwnerRegno,
				OwnerName: x.OwnerName, GrantType: x.GrantType, ApprovedAt: x.GrantApprveDate, TerminatesAt: x.GrantTrminateDate})
		}
		out[i] = l
	}
	return out, nil
}
