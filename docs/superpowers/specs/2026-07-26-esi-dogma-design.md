# ESI Dogma (Twelfth Batch) Design

## Status

Approved (autonomous execution). Twelfth batch: the "Dogma" tag alone (5 endpoints, all public —
Dogma is EVE's live attribute/effect reference data, no character auth required). A simple, small
batch, in contrast to the outsized complexity of the prior Corporation Projects batch.

## Scope

All 5 endpoints are **public** (`esi.NewPublicClient`, no existence check, no `characterId` input)
— confirmed from the generated request builders, none of which take a `characterId` at all:
`GetDogmaAttributes` (list of attribute IDs), `GetDogmaAttributesAttributeId` (attribute detail),
`GetDogmaDynamicItemsTypeIdItemId` (dynamic/mutated item detail), `GetDogmaEffects` (list of effect
IDs), `GetDogmaEffectsEffectId` (effect detail).

No pagination on any of the 5 (verified — none of the request builders have a `.Page()` method).

## `esi/dogma.go` (5 methods)

```go
func (c *Client) DogmaAttributes(ctx context.Context) ([]int64, error)
func (c *Client) DogmaAttributeDetail(ctx context.Context, attributeID int64) (*esiapi.DogmaAttributesAttributeIdGet, error)
func (c *Client) DogmaDynamicItemDetail(ctx context.Context, typeID int64, itemID int64) (*esiapi.DogmaDynamicItemsTypeIdItemIdGet, error)
func (c *Client) DogmaEffects(ctx context.Context) ([]int64, error)
func (c *Client) DogmaEffectDetail(ctx context.Context, effectID int64) (*esiapi.DogmaEffectsEffectIdGet, error)
```

Note `DogmaDynamicItemDetail`'s parameter order: neat's facade keeps `(typeID, itemID)` — matching
the URL path's own ordering (`/dogma/dynamic/items/{type_id}/{item_id}/`) and the generated
function's own positional order (`GetDogmaDynamicItemsTypeIdItemId(ctx, itemId, typeId)` takes
`itemId` first in the generated signature — verified from source; neat's facade normalizes to the
URL's `(typeID, itemID)` order for caller clarity, since ESI's own generated order here is the
reverse of its own path).

Verified model shapes: `DogmaAttributesAttributeIdGet` — 10 fields, only `AttributeId` required,
the other 9 (`DefaultValue`, `Description`, `DisplayName`, `HighIsGood`, `IconId`, `Name`,
`Published`, `Stackable`, `UnitId`) all optional/pointer. `DogmaEffectsEffectIdGet` — 21 fields,
only `EffectId` required, the rest optional/pointer, including an optional `Modifiers
[]DogmaEffectsEffectIdGetModifiersInner` slice (`{Domain, Func string, EffectId,
ModifiedAttributeId, ModifyingAttributeId, Operator *int64}` — `Func` is the only required field on
a modifier). `DogmaDynamicItemsTypeIdItemIdGet` — `CreatedBy`, `MutatorTypeId`, `SourceTypeId` int64
all required, plus two non-optional slices: `DogmaAttributes
[]DogmaDynamicItemsTypeIdItemIdGetDogmaAttributesInner` (`{AttributeId int64, Value float64}`, both
required) and `DogmaEffects []DogmaDynamicItemsTypeIdItemIdGetDogmaEffectsInner` (`{EffectId int64,
IsDefault bool}`, both required).

## `mcp/dogma.go` (5 tools)

All public (no existence check):

- `list_dogma_attributes` — input: none, output `{attributeIds []int64}`
- `get_dogma_attribute` — input `{attributeId int64}`, output `{attributeId int64, defaultValue *float64, description, displayName, name *string, highIsGood, published, stackable *bool, iconId, unitId *int64}`
- `get_dogma_dynamic_item` — input `{typeId, itemId int64}`, output `{typeId, itemId, createdBy, mutatorTypeId, sourceTypeId int64, dogmaAttributes []DynamicItemAttribute, dogmaEffects []DynamicItemEffect}` where `DynamicItemAttribute{AttributeID int64, Value float64}` and `DynamicItemEffect{EffectID int64, IsDefault bool}`
- `list_dogma_effects` — input: none, output `{effectIds []int64}`
- `get_dogma_effect` — input `{effectId int64}`, output `{effectId int64, description, displayName, name *string, disallowAutoRepeat, electronicChance, isAssistance, isOffensive, isWarpSafe, published, rangeChance *bool, dischargeAttributeId, durationAttributeId, effectCategory, falloffAttributeId, iconId, postExpression, preExpression, rangeAttributeId, trackingSpeedAttributeId *int64, modifiers []EffectModifier}` where `EffectModifier{Func string, Domain *string, EffectID, ModifiedAttributeID, ModifyingAttributeID, Operator *int64}`

## Testing

Same isolation as every prior batch. `esi/dogma_test.go` covers all 5 methods (parsing + at least
one error path). `mcp/dogma_internal_test.go` covers all 5 tools (no unknown-character tests needed
— every tool is public). `mcp/server_test.go` grows from 83 to 88.

## Out of scope

- No SDE cross-referencing (e.g. resolving `attribute_id`/`effect_id`/`unit_id` to human-readable
  names beyond what ESI itself returns) — that's `sde/`'s job if ever needed, not this tag's.
