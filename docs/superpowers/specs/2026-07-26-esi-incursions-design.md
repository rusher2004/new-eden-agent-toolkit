# ESI Incursions (Seventeenth Batch) Design

## Status

Approved (autonomous execution). Seventeenth batch: the "Incursions" tag — a single public
endpoint, `GetIncursions`. The smallest batch in this sweep so far.

## Scope

**1 public** (`esi.NewPublicClient`, no existence check, no pagination):
`GetIncursions(ctx) ([]IncursionsGetInner, error)`.

Verified model shape: `IncursionsGetInner{ConstellationId, FactionId, StagingSolarSystemId int64,
HasBoss bool, InfestedSolarSystems []int64, Influence float64, State, Type string}` — all fields
required, no pointers.

## `esi/incursions.go` (1 method)

```go
func (c *Client) Incursions(ctx context.Context) ([]esiapi.IncursionsGetInner, error)
```

## `mcp/incursions.go` (1 tool)

- `list_incursions` — public, input: none, output `{incursions []Incursion}` where
  `Incursion{ConstellationID, FactionID, StagingSolarSystemID int64, HasBoss bool,
  InfestedSolarSystemIDs []int64, Influence float64, State, Type string}`

## Testing

`esi/incursions_test.go` covers parsing + one error path. `mcp/incursions_internal_test.go`
covers the one tool's happy path (no unknown-character test needed — fully public).
`mcp/server_test.go` grows from 119 to 120.
