# ebarimt-pos3-go v2

Go client for the Mongolian eBarimt **PosAPI 3.0** system, rewritten from the
[official documentation](https://developer.itc.gov.mn/detail/proj-1787042993564).
Design and rationale: [issue #37](https://github.com/techpartners-asia/ebarimt-pos3-go/issues/37).
Coming from v1? Read [MIGRATION.md](MIGRATION.md).

```
go get github.com/techpartners-asia/ebarimt-pos3-go/v2
```

Go 1.25 or newer. Dependencies: `shopspring/decimal`, `golang.org/x/oauth2` (package `oidc` only).

## Packages

Each package owns one kind of knowledge.

| Package | Owns | I/O |
|---|---|---|
| `receipt` | Receipt law: tax split, grouping, rounding, required fields per receipt type | none |
| `posapi` | The local PosAPI daemon: issue, return, send, status, bank accounts | HTTP to your daemon |
| `registry` | Public registry and reference lists: taxpayer, TIN, districts, relief codes, БҮНА, barcodes, locations | HTTP |
| `oidc` | Token source for the authenticated tax-authority services | HTTP |
| `money` | Exact two-decimal `Amount`, exact `Quantity` | none |
| `postest` | In-memory fake PosAPI that enforces the docs' rules | in-process |
| `ebarimt` (module root) | The one `*Error` type and the `Environment` presets | none |

Not in v2.0: `taxauthority`, `easyregister`, `excise` (planned for v2.1, each to be
checked against staging first). Removed from v1 on purpose: database persistence,
email, QR images.

## File a receipt

```go
// 1. The seller's registration decides how tax is split. Read it once and cache it.
reg := registry.New(registry.Config{Environment: ebarimt.Production})
tp, err := reg.Taxpayer(ctx, merchantTIN) // errors.Is(err, ebarimt.ErrNotFound) when unknown

merchant := receipt.Merchant{
    TIN: merchantTIN, VATPayer: tp.VATPayer,
    CityTaxPayer: tp.CityTaxPayer, VATFreeProject: tp.VATFreeProject,
}

// 2. Build is pure. It returns every rule you broke at once, before anything is sent.
req, err := receipt.Build(receipt.Sale{
    Type:         receipt.B2CReceipt,
    PosNo:        "10000001",
    BranchNo:     "001",
    DistrictCode: "2501", // registry.Districts
    Items: []receipt.Item{{
        Name: "Cola 0.5l", ClassificationCode: "2349010", MeasureUnit: "ш",
        TaxType: receipt.TaxVATable, HasCityTax: true,
        Quantity: decimal.NewFromInt(1), Total: money.FromTugrik(2800), // all taxes included
    }},
    Payments: []receipt.Payment{{Code: receipt.PaymentCard, Amount: money.FromTugrik(2800)}},
}, merchant)

// 3. Send it to your daemon (its address is always yours to supply).
pos := posapi.New(posapi.Config{Endpoint: "http://10.0.0.5:7080"})
issued, err := pos.IssueReceipt(ctx, req)
// Print issued.Lottery and issued.QRData on the receipt. Do not store or log them:
// the spec forbids keeping them in any other form.
```

### When the answer is lost

`IssueReceipt` is not idempotent and is never retried. Branch on the error's `Kind`:

```go
var e *ebarimt.Error
if errors.As(err, &e) {
    switch e.Kind {
    case ebarimt.Invalid:       // refused locally; e.Violations lists every rule broken
    case ebarimt.Rejected:      // PosAPI said no; e.Message is its own text. Nothing was filed.
    case ebarimt.Transport:     // never fully sent. Safe to retry.
    case ebarimt.Indeterminate: // sent in full, answer lost. A receipt MAY exist.
                                // Do not retry blindly: reconcile first, or the sale is filed twice.
    }
}
```

An error never contains a response body.

## Tax rules `Build` owns

`totalAmount = base + VAT + city tax`
([spec](https://developer.itc.gov.mn/detail/proj-1787042993564?item=api-1787209763945):
1000 + 100 + 20 = 1120), so the divisor depends on which taxes are actually added:

```
addVAT  = merchant.VATPayer && item.TaxType == VAT_ABLE
addCity = merchant.CityTaxPayer && item.HasCityTax && item.TaxType != NOT_VAT
base    = total / (1 + 0.10*addVAT + 0.02*addCity)
vat     = base * 0.10     city = base * 0.02     (each rounded half-up to 2 places)
```

For a 2800 line: VAT and city tax 250 / 50; VAT only 254.55 / 0; city tax only 0 / 54.90; neither 0 / 0.

- One sub-receipt per tax type, ordered by tax type. Item VAT and city tax are rounded
  to 2 places first; sub-receipt and header are exact sums of those, and the payments
  must equal the total.
- `unitPrice` is total/quantity truncated to 2 places.
- A merchant with `VATFreeProject` sends every item as `VAT_FREE` with relief code `304`.
- Violations reported together: no items; quantity or total not above 0; a
  `ClassificationCode` that is not 7 digits (every item); a missing `TaxReliefCode` on
  `VAT_FREE`/`VAT_ZERO`/`NOT_VAT`, or one on `VAT_ABLE`; payments missing or not summing
  to the total; B2B types without `Buyer.TIN`, B2C types with one; `ConsumerNo` outside
  `B2C_RECEIPT`; `ReportMonth` on `B2C_RECEIPT`; an unknown type; a missing merchant TIN.
- `Sale.BillIDSuffix` is sent as given and omitted when empty. The SDK does not derive it.

## Money

`money.Amount` is a decimal underneath, always rounded half-up to 2 places, and never a
`float64`. It marshals as a JSON number in shortest form (`2800`, `457.4`, `1572.32`).
457.40 + 1114.92 is exactly 1572.32.

## Environments

```go
reg := registry.New(registry.Config{Environment: ebarimt.Staging}) // st-api.ebarimt.mn
tok := oidc.Config{TokenURL: ebarimt.Staging.TokenURL(), ClientID: oidc.ClientVATPS, Username: u, Password: p}
```

`ebarimt.Production` (the zero value) and `ebarimt.Staging` name every documented host.
Every URL can be overridden in the client's `Config`. The staging service host
(`service.itc.gov.mn` counterpart) is not published, so `Staging.ServiceURL()` is empty.

## Testing your code

```go
pos := postest.New(t)          // httptest server, closed with the test
client := pos.Client()         // a posapi.Client pointed at it
issued, err := client.IssueReceipt(ctx, req)
pos.Requests()                 // what it received, in order
pos.FailNext(503, "maintenance")
```

The fake refuses what PosAPI refuses: totals that are not exact sums, a missing or
non-7-digit `classificationCode`, a missing relief code, a B2B receipt without
`customerTin`, a wrong city tax (with PosAPI's own message `НХАТ-г 2%-р тооцоолоогүй
байна. Шалгуур дүн: …`), and a `DELETE` without a body.

## Rules every package follows

- `ctx` is the first argument of every call.
- One error type, `*ebarimt.Error`; `errors.Is(err, ebarimt.ErrNotFound)` for lookups that came back empty.
- No package globals, no logging, no hidden state; clients are safe for concurrent use.
- No retries in `posapi`. `registry` can retry its idempotent GETs (`Config.Retries`, off by default).
- Redirects are never followed.

## Known limits (staging checks pending)

- The wire keeps the production-proven spelling `totalVat` / `inActiveId`; the docs say `totalVAT` / `inactiveId`.
- Item VAT and city tax are sent rounded to 2 places, as in the docs' examples; staging has not confirmed it yet.
- `Sale.ReportMonth` is sent as the first day of the month (`2026-09-01`); the docs give no format.
- `registry.ProductClassification` follows the docs' description of the БҮНА walk, not their URL template; not yet tried live.
