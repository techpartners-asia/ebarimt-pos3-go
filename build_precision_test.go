package ebarimtv3

import (
	"testing"

	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"github.com/techpartners-asia/ebarimt-pos3-go/utils"
)

// POS API 3.0 rejects a receipt whose totalAmount differs from the sum of its
// items' totalAmount ("дэд баримтын totalAmount ... нийлбэртэй таарсангүй").
// Items with fractional amounts (discounts) must therefore be rounded to 2
// decimals and summed with rounding, or float noise breaks the equality.
func TestReceiptTotalsEqualSumOfRoundedItems(t *testing.T) {
	e := New(Input{Endpoint: "http://127.0.0.1:1", PosNo: "10015280", MerchantTin: "70201222345"})
	items := []structs.CreateItemInputModel{
		{Name: "a", TaxType: constants.TAX_VAT_ABLE, ClassificationCode: "5610101", Qty: 3, TotalAmount: 19426.0 / 3 * 3 * 0.9333333333},
		{Name: "b", TaxType: constants.TAX_VAT_ABLE, ClassificationCode: "5610101", Qty: 1, TotalAmount: 0.1},
		{Name: "c", TaxType: constants.TAX_VAT_ABLE, ClassificationCode: "5610101", Qty: 1, TotalAmount: 0.2},
		{Name: "d", TaxType: constants.TAX_VAT_ABLE, ClassificationCode: "5610101", Qty: 7, TotalAmount: 99300.0 * 0.85 / 7 * 7},
		{Name: "e", TaxType: constants.TAX_VAT_FREE, ClassificationCode: "5610101", Qty: 2, TotalAmount: 12345.678},
	}
	req := structs.ReceiptRequest{}
	m, err := e.buildReceiptItemMap(items, &req)
	if err != nil {
		t.Fatal(err)
	}
	e.buildReceipt(&req, m)

	var grand float64
	for _, r := range req.Receipts {
		var sum float64
		for _, it := range r.Items {
			if it.TotalAmount != utils.NumberPrecision(it.TotalAmount) {
				t.Errorf("item %s totalAmount %v not rounded to 2 decimals", it.Name, it.TotalAmount)
			}
			sum = utils.NumberPrecision(sum + it.TotalAmount)
		}
		if r.TotalAmount != sum {
			t.Errorf("receipt %s totalAmount %v != rounded sum of items %v", r.TaxType, r.TotalAmount, sum)
		}
		grand = utils.NumberPrecision(grand + r.TotalAmount)
	}
	if req.TotalAmount != grand {
		t.Errorf("request totalAmount %v != rounded sum of receipts %v", req.TotalAmount, grand)
	}
}
