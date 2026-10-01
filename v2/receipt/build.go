// Package receipt builds a valid PosAPI receipt request from a sale. It is
// pure: no I/O, no clock, no globals.
//
// It owns the receipt law: how a line total splits into base, VAT and city tax,
// how lines group into sub-receipts, how amounts round, and which fields each
// receipt type requires. Everything PosAPI would otherwise reject after the
// round trip is reported by Build up front, all violations at once.
//
// Specs:
// https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787209763945 (receipt)
// https://developer.itc.gov.mn/detail/proj-1787042993564?item=doc-1787215296174 (system requirements)
package receipt

import (
	"fmt"
	"sort"

	"github.com/shopspring/decimal"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/posapi"
)

// FreeProjectReliefCode is the relief code of a VAT-exempt project: when the
// merchant's freeProject flag is set, every item is sent as VAT_FREE with it
// (https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787213583668).
const FreeProjectReliefCode = "304"

// Build turns a sale into the request for posapi.Client.IssueReceipt.
//
// The request has one sub-receipt per tax type, ordered by tax type; each
// sub-receipt's total, VAT and city tax are the exact sums of its items', and
// the header's are the exact sums of the sub-receipts'. The payments must add up
// to the same total. Item VAT and city tax are rounded to 2 decimals before
// summing, so header = sum of sub-receipts = sum of items, exactly.
//
// If anything is wrong, Build returns one *ebarimt.Error of Kind Invalid whose
// Violations list every broken rule, and a zero request.
func Build(s Sale, m Merchant) (posapi.ReceiptRequest, error) {
	const op = "receipt.Build"
	var v violations

	if m.TIN == "" {
		v.add("merchant.tin", "required")
	}

	// Buyer, report month and type.
	isB2B := false
	switch s.Type {
	case B2BReceipt, B2BInvoice:
		isB2B = true
	case B2CReceipt, B2CInvoice:
	default:
		v.add("type", fmt.Sprintf("must be one of B2C_RECEIPT, B2B_RECEIPT, B2C_INVOICE, B2B_INVOICE, got %q", s.Type))
	}
	if s.Type != "" {
		switch {
		case isB2B && s.Buyer.TIN == "":
			v.add("buyer.tin", "required for "+string(s.Type))
		case !isB2B && s.Buyer.TIN != "":
			v.add("buyer.tin", "must be empty for "+string(s.Type)+": only B2B types carry a buyer TIN")
		}
	}
	if s.Buyer.ConsumerNo != "" && s.Type != B2CReceipt {
		v.add("buyer.consumerNo", "allowed only for B2C_RECEIPT")
	}
	var reportMonth *string
	if s.ReportMonth != nil {
		if s.Type == B2CReceipt {
			v.add("reportMonth", "allowed only for B2B_RECEIPT, B2C_INVOICE and B2B_INVOICE")
		}
		if s.ReportMonth.Year < 1 || s.ReportMonth.Month < 1 || s.ReportMonth.Month > 12 {
			v.add("reportMonth", "must be a real calendar month")
		}
		str := s.ReportMonth.String()
		reportMonth = &str
	}

	// Items.
	if len(s.Items) == 0 {
		v.add("items", "at least one item is required")
	}
	type line struct {
		item     posapi.Item
		taxType  TaxType
		validQty bool
	}
	lines := make([]line, 0, len(s.Items))
	for i, it := range s.Items {
		f := func(field string) string { return fmt.Sprintf("items[%d].%s", i, field) }

		switch it.TaxType {
		case TaxVATable, TaxVATExempt, TaxVATZeroRated, TaxOutsideVAT:
		default:
			v.add(f("taxType"), fmt.Sprintf("must be one of VAT_ABLE, VAT_FREE, VAT_ZERO, NOT_VAT, got %q", it.TaxType))
		}
		qtyOK := it.Quantity.Sign() > 0
		if !qtyOK {
			v.add(f("quantity"), "must be greater than 0")
		}
		if it.Total.Sign() <= 0 {
			v.add(f("total"), "must be greater than 0")
		}
		if !isSevenDigits(it.ClassificationCode) {
			v.add(f("classificationCode"), "must be a 7-digit БҮНА code, required for every item")
		}

		// A VAT-exempt project sends every item as VAT_FREE with relief 304.
		taxType, relief := it.TaxType, it.TaxReliefCode
		if m.VATFreeProject {
			taxType, relief = TaxVATExempt, FreeProjectReliefCode
		}
		switch taxType {
		case TaxVATExempt, TaxVATZeroRated, TaxOutsideVAT:
			if relief == "" {
				v.add(f("taxReliefCode"), "required for "+string(taxType))
			}
		case TaxVATable:
			if relief != "" {
				v.add(f("taxReliefCode"), "must be empty for VAT_ABLE")
			}
		}

		barcodeType := it.BarcodeType
		if barcodeType == "" {
			barcodeType = BarcodeUndefined
		}
		var data *posapi.ItemData
		if len(it.ExciseStampQRs) > 0 || it.MedicineLotNo != "" {
			data = &posapi.ItemData{LotNo: it.MedicineLotNo, StockQR: it.ExciseStampQRs}
		}
		wire := posapi.Item{
			Name:               it.Name,
			BarCode:            it.Barcode,
			BarCodeType:        barcodeType,
			ClassificationCode: it.ClassificationCode,
			TaxProductCode:     relief,
			MeasureUnit:        it.MeasureUnit,
			Qty:                money.QuantityFromDecimal(it.Quantity),
			TotalAmount:        it.Total,
			Data:               data,
		}
		if qtyOK {
			wire.UnitPrice = unitPrice(it.Total, it.Quantity)
		}
		wire.TotalVat, wire.TotalCityTax = SplitTaxes(it.Total, taxType, it.HasCityTax, m)
		lines = append(lines, line{item: wire, taxType: taxType})
	}

	// Payments.
	var paid money.Amount
	if len(s.Payments) == 0 {
		v.add("payments", "at least one payment is required")
	}
	for _, p := range s.Payments {
		paid = paid.Add(p.Amount)
	}

	// Group and sum, in decimal.
	byType := map[TaxType]*posapi.Receipt{}
	var total, totalVat, totalCity money.Amount
	for _, l := range lines {
		r := byType[l.taxType]
		if r == nil {
			r = &posapi.Receipt{TaxType: l.taxType, MerchantTin: m.TIN}
			if s.Location != nil {
				loc := *s.Location
				r.Data = &posapi.ReceiptData{Location: []posapi.Location{loc}}
			}
			byType[l.taxType] = r
		}
		r.Items = append(r.Items, l.item)
		r.TotalAmount = r.TotalAmount.Add(l.item.TotalAmount)
		r.TotalVat = r.TotalVat.Add(l.item.TotalVat)
		r.TotalCityTax = r.TotalCityTax.Add(l.item.TotalCityTax)
	}
	receipts := make([]posapi.Receipt, 0, len(byType))
	for _, r := range byType {
		receipts = append(receipts, *r)
	}
	sort.Slice(receipts, func(i, j int) bool { return receipts[i].TaxType < receipts[j].TaxType })
	for _, r := range receipts {
		total = total.Add(r.TotalAmount)
		totalVat = totalVat.Add(r.TotalVat)
		totalCity = totalCity.Add(r.TotalCityTax)
	}

	if len(s.Payments) > 0 && len(lines) > 0 && !paid.Equal(total) {
		v.add("payments", fmt.Sprintf("must sum to the sale total %s, got %s", total, paid))
	}

	if len(v) > 0 {
		return posapi.ReceiptRequest{}, &ebarimt.Error{Kind: ebarimt.Invalid, Op: op, Violations: v}
	}

	payments := make([]posapi.Payment, len(s.Payments))
	for i, p := range s.Payments {
		payments[i] = posapi.Payment{Code: p.Code, ExchangeCode: p.ExchangeCode, Status: posapi.PaymentPaid, PaidAmount: p.Amount}
	}
	return posapi.ReceiptRequest{
		TotalAmount:  total,
		TotalVat:     totalVat,
		TotalCityTax: totalCity,
		BranchNo:     s.BranchNo,
		DistrictCode: s.DistrictCode,
		MerchantTin:  m.TIN,
		PosNo:        s.PosNo,
		CustomerTin:  s.Buyer.TIN,
		ConsumerNo:   s.Buyer.ConsumerNo,
		Type:         s.Type,
		InActiveID:   s.ReplacesReceiptID,
		ReportMonth:  reportMonth,
		BillIDSuffix: s.BillIDSuffix,
		Receipts:     receipts,
		Payments:     payments,
	}, nil
}

// unitPrice is total/qty truncated toward zero to 2 decimals. Truncating the
// exact quotient (not a rounded or float one) never loses or gains a cent.
func unitPrice(total money.Amount, qty decimal.Decimal) money.Amount {
	q, _ := total.Decimal().QuoRem(qty, 2)
	return money.FromDecimal(q)
}

func isSevenDigits(s string) bool {
	if len(s) != 7 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

type violations []ebarimt.Violation

func (v *violations) add(field, rule string) {
	*v = append(*v, ebarimt.Violation{Field: field, Rule: rule})
}
