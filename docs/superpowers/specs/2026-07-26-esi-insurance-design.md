# ESI Insurance (Nineteenth Batch) Design

## Status

Approved (autonomous execution). Nineteenth batch: the "Insurance" tag — a single public
endpoint, `GetInsurancePrices`. Same size class as Incursions.

## Scope

**1 public** (`esi.NewPublicClient`, no existence check, no pagination):
`GetInsurancePrices(ctx) ([]InsurancePricesGetInner, error)`.

Verified model shape: `InsurancePricesGetInner{TypeId int64, Levels
[]InsurancePricesGetInnerLevelsInner{Cost, Payout float64, Name string}}` — all fields required.

## `esi/insurance.go` (1 method)

```go
func (c *Client) InsurancePrices(ctx context.Context) ([]esiapi.InsurancePricesGetInner, error)
```

## `mcp/insurance.go` (1 tool)

- `list_insurance_prices` — public, input: none, output `{prices []InsurancePrice}` where
  `InsurancePrice{TypeID int64, Levels []InsuranceLevel}` and `InsuranceLevel{Cost, Payout
  float64, Name string}`

## Testing

`esi/insurance_test.go` covers parsing (including a non-empty nested `Levels` array) + one error
path. `mcp/insurance_internal_test.go` covers the one tool's happy path (no unknown-character test
needed — fully public). `mcp/server_test.go` grows from 128 to 129.
