package postest_test

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/postest"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/receipt"
)

// Build a receipt, file it against the fake daemon, return it.
func Example() {
	var t fakeT // in a real test this is the *testing.T you already have
	defer t.close()
	pos := postest.New(&t)
	client := pos.Client()

	// The merchant's registration comes from registry.Taxpayer in production.
	merchant := receipt.Merchant{TIN: "37900846788", VATPayer: true, CityTaxPayer: true}
	req, err := receipt.Build(receipt.Sale{
		Type: receipt.B2CReceipt, PosNo: "10000001", BranchNo: "001", DistrictCode: "2501",
		Items: []receipt.Item{{
			Name: "Cola 0.5l", ClassificationCode: "2349010", MeasureUnit: "ш",
			TaxType: receipt.TaxVATable, HasCityTax: true, Quantity: decimal.NewFromInt(1), Total: money.FromTugrik(2800),
		}},
		Payments: []receipt.Payment{{Code: receipt.PaymentCard, Amount: money.FromTugrik(2800)}},
	}, merchant)
	if err != nil {
		panic(err)
	}

	issued, err := client.IssueReceipt(context.Background(), req)
	if err != nil {
		panic(err)
	}
	fmt.Println("total:", issued.TotalAmount, "vat:", issued.TotalVAT, "city tax:", issued.TotalCityTax)
	fmt.Println("id digits:", len(issued.ID))
	fmt.Println("returned:", client.ReturnReceipt(context.Background(), issued.ID, issued.Date))
	// Output:
	// total: 2800 vat: 250 city tax: 50
	// id digits: 33
	// returned: <nil>
}
