# Migrating from v1 to v2

v1 (`github.com/techpartners-asia/ebarimt-pos3-go`, repo root) stays untouched and keeps
working. v2 is a separate module, `github.com/techpartners-asia/ebarimt-pos3-go/v2`, so both
can be imported side by side while you move over. v2 is a rewrite from the PosAPI 3.0 docs,
not a patch: the shapes differ, and it fixes #3, #10-#33 and #36 by construction.

## What changed in shape

- One tiny package per kind of knowledge (`receipt`, `posapi`, `registry`, `oidc`, `money`) instead of one `EbarimtClient`.
- Building a request is pure (`receipt.Build`) and separate from sending it (`posapi.Client`).
- Amounts are `money.Amount`, not `float64`.
- Every call takes a `context.Context` first and returns `*ebarimt.Error`.
- The PosAPI daemon address, merchant TIN and POS number are no longer client configuration: the
  address is `posapi.Config.Endpoint`; TIN comes from `receipt.Merchant`, POS number from `receipt.Sale.PosNo`.
- **Removed, on purpose:** database persistence (GORM), email, QR images. Storing `lottery`/`qrData`
  is forbidden by the receipt spec. Persist your own sale and the receipt `ID` if you need to.
- `IsDev` is gone: pick `ebarimt.Production` or `ebarimt.Staging`.

## Rename table

| v1 | v2 | Why |
|---|---|---|
| `Pos3` | `posapi.Client` | Names the system, not a version |
| `ReceiptSend` / `ReceiptDelete` | `IssueReceipt` / `ReturnReceipt` | The docs' verbs: хадгалах / буцаах |
| `SendData` | `SendToTaxAuthority` | Says where the data goes |
| `Info`, `LastSendDate` | `Status`, `LastSentAt` (`time.Time`) | Fixes #25 |
| `GetInfo` / `GetTinInfo` / `GetBranchInfo` | `registry.Taxpayer` / `TINByRegisterNumber` / `Districts` | Say what comes back |
| `GetSalesTotalData`, `GetSalesListERP`, `GEtSalesListERPResponse`, `RecieptBuyModel` | `taxauthority.SalesBreakdown`, `SubsidiaryPurchases`, `Purchase` (v2.1) | Typos; names the service |
| `SaveOprMerchants` | `taxauthority.RegisterMerchants` (v2.1) | Plain verb |
| `ConsumerInfo` / `GetProfile` / `ApproveQr` | `easyregister.Consumer` / `ConsumerByPhoneOrNumber` / `ConfirmReceipt` (v2.1) | Say what each does |
| `Foriegner*` | `Foreigner*` (v2.1) | Spelling |
| `TAX_NO_VAT` | `receipt.TaxOutsideVAT` | The wire value is `NOT_VAT` |
| `TAX_VAT_ABLE` / `TAX_VAT_FREE` / `TAX_VAT_ZERO` | `TaxVATable` / `TaxVATExempt` / `TaxVATZeroRated` | Names say what they mean |
| `TaxProductCode` | `TaxReliefCode` | It is the legal basis for relief, not a product code |
| `InActiveID` | `ReplacesReceiptID` | Says what the field means |
| `NumberPrecision`, `GetVat...` | `money.Amount`, `receipt.SplitTaxes` | Exact math; one entry point |
| `PAYMENT_CASH`, `PAYMENT_CARD` | `receipt.PaymentCash`, `PaymentCard`, plus `PaymentBankTransfer`, `PaymentQPay` | |
| `RECEIPT_B2C_RECEIPT` ... | `receipt.B2CReceipt`, `B2BReceipt`, `B2CInvoice`, `B2BInvoice` | |
| `BARCODE_GS1` ... | `receipt.BarcodeGS1`, `BarcodeISBN`, `BarcodeUndefined` | |

Wire field names (`totalVat`, `inActiveId`) are unchanged: v2 keeps the spelling production has proven.

## Behaviour that differs

| v1 | v2 |
|---|---|
| `Create` sends up to two receipts (NOT_VAT first, then the rest) and prints responses | `Build` makes one request with one sub-receipt per tax type; nothing is printed |
| Totals summed in `float64` | Exact; header = sum of sub-receipts = sum of payments |
| City tax ignores whether the merchant is a city-tax payer | `Merchant.CityTaxPayer` gates it |
| Payments ignored | `Sale.Payments` required and checked against the total |
| Item without a БҮНА code or relief code silently dropped | `Build` returns `Invalid` listing every item and rule |
| Non-2xx or junk body read as success | `Rejected` / `Transport` |
| `DELETE` body never sent | `ReturnReceipt` always sends `{"id","date"}` |
| Auth never worked | `oidc` on `golang.org/x/oauth2` |
| Timeout after the receipt was issued looked like any other failure | `Indeterminate`: do not retry blindly |
| Item VAT / city tax unrounded (`254.54545454545453`) | Rounded to 2 places (`254.55`) |
| Sub-receipt order random | Ordered by tax type |

## Before and after: filing a receipt

v1:

```go
client := ebarimtv3.New(ebarimtv3.Input{
    Endpoint: "http://10.0.0.5:7080", PosNo: "10000001", MerchantTin: "37900846788",
})
res, err := client.Create(models.CreateInputModel{ /* items, payment type, branch ... */ })
if err != nil {
    // could not tell "refused" from "filed but the answer was lost"
}
fmt.Println(res.Lottery)
```

v2:

```go
reg := registry.New(registry.Config{})
tp, err := reg.Taxpayer(ctx, "37900846788")
if err != nil { /* ... */ }

req, err := receipt.Build(receipt.Sale{
    Type: receipt.B2CReceipt, PosNo: "10000001", BranchNo: "001", DistrictCode: "2501",
    Items:    []receipt.Item{{Name: "Cola", ClassificationCode: "2349010", MeasureUnit: "ш",
        TaxType: receipt.TaxVATable, HasCityTax: true,
        Quantity: decimal.NewFromInt(1), Total: money.FromTugrik(2800)}},
    Payments: []receipt.Payment{{Code: receipt.PaymentCard, Amount: money.FromTugrik(2800)}},
}, receipt.Merchant{TIN: "37900846788", VATPayer: tp.VATPayer, CityTaxPayer: tp.CityTaxPayer, VATFreeProject: tp.VATFreeProject})
if err != nil { /* *ebarimt.Error{Kind: Invalid}: every violation, nothing sent */ }

issued, err := posapi.New(posapi.Config{Endpoint: "http://10.0.0.5:7080"}).IssueReceipt(ctx, req)
var e *ebarimt.Error
if errors.As(err, &e) && e.Kind == ebarimt.Indeterminate {
    // the receipt may exist: reconcile before any retry
}
fmt.Println(issued.Lottery) // print it, do not keep it
```

## Before and after: other calls

```go
// v1                                      // v2
pos.GetInfo(tin)                           reg.Taxpayer(ctx, tin)             // ErrNotFound when found=false
pos.GetTinInfo(regNo)                      reg.TINByRegisterNumber(ctx, regNo) // ErrNotFound for the live "status 500, data null"
pos.GetBranchInfo()                        reg.Districts(ctx)
pos.ReceiptDelete(req{ID, Date: "..."})    pos.ReturnReceipt(ctx, id, issuedAt)
pos.SendData()                             pos.SendToTaxAuthority(ctx)
pos.Info()                                 pos.Status(ctx)                    // LastSentAt is a time.Time
pos.BankAccounts(tin)                      pos.BankAccounts(ctx, tin)
```

New in v2 with no v1 counterpart: `registry.TaxReliefCodes`, `ProductClassification`,
`RegisteredBarcodes`, `MerchantLocations`; `postest`.
