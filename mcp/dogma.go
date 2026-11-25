package mcp

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// dogmaESI is the ESI calls the dogma tools make; all are public.
type dogmaESI interface {
	DogmaAttributes(ctx context.Context) ([]int64, error)
	DogmaAttributeDetail(ctx context.Context, attributeID int64) (*esiapi.DogmaAttributesAttributeIdGet, error)
	DogmaDynamicItemDetail(ctx context.Context, typeID int64, itemID int64) (*esiapi.DogmaDynamicItemsTypeIdItemIdGet, error)
	DogmaEffects(ctx context.Context) ([]int64, error)
	DogmaEffectDetail(ctx context.Context, effectID int64) (*esiapi.DogmaEffectsEffectIdGet, error)
}

// --- list_dogma_attributes (public, no input) ---

func listDogmaAttributes(ctx context.Context, client dogmaESI) ([]int64, error) {
	return client.DogmaAttributes(ctx)
}

type ListDogmaAttributesInput struct{}
type ListDogmaAttributesOutput struct {
	AttributeIDs []int64 `json:"attributeIds"`
}

// AddListDogmaAttributes registers list_dogma_attributes. open builds the ESI client; any client with the
// dogmaESI methods will do.
func AddListDogmaAttributes[C dogmaESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_dogma_attributes",
		Description: "List every dogma attribute ID currently defined (public data, no character required).",
	}, listDogmaAttributesHandler(open))
}

func listDogmaAttributesHandler[C dogmaESI](open func() C) mcpsdk.ToolHandlerFor[ListDogmaAttributesInput, ListDogmaAttributesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListDogmaAttributesInput) (*mcpsdk.CallToolResult, ListDogmaAttributesOutput, error) {
		ids, err := listDogmaAttributes(ctx, open())
		if err != nil {
			return nil, ListDogmaAttributesOutput{}, fmt.Errorf("list dogma attributes: %w", err)
		}
		return nil, ListDogmaAttributesOutput{AttributeIDs: ids}, nil
	}
}

// --- get_dogma_attribute (public) ---

type GetDogmaAttributeOutput struct {
	AttributeID  int64    `json:"attributeId"`
	DefaultValue *float64 `json:"defaultValue,omitempty"`
	Description  *string  `json:"description,omitempty"`
	DisplayName  *string  `json:"displayName,omitempty"`
	HighIsGood   *bool    `json:"highIsGood,omitempty"`
	IconID       *int64   `json:"iconId,omitempty"`
	Name         *string  `json:"name,omitempty"`
	Published    *bool    `json:"published,omitempty"`
	Stackable    *bool    `json:"stackable,omitempty"`
	UnitID       *int64   `json:"unitId,omitempty"`
}

func getDogmaAttribute(ctx context.Context, client dogmaESI, attributeID int64) (GetDogmaAttributeOutput, error) {
	attr, err := client.DogmaAttributeDetail(ctx, attributeID)
	if err != nil {
		return GetDogmaAttributeOutput{}, err
	}
	return GetDogmaAttributeOutput{
		AttributeID: attr.AttributeId, DefaultValue: attr.DefaultValue, Description: attr.Description,
		DisplayName: attr.DisplayName, HighIsGood: attr.HighIsGood, IconID: attr.IconId,
		Name: attr.Name, Published: attr.Published, Stackable: attr.Stackable, UnitID: attr.UnitId,
	}, nil
}

type GetDogmaAttributeInput struct {
	AttributeID int64 `json:"attributeId" jsonschema:"the dogma attribute ID to look up"`
}

// AddGetDogmaAttribute registers get_dogma_attribute. open builds the ESI client; any client with the
// dogmaESI methods will do.
func AddGetDogmaAttribute[C dogmaESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_dogma_attribute",
		Description: "Get detail for one dogma attribute (public data, no character required).",
	}, getDogmaAttributeHandler(open))
}

func getDogmaAttributeHandler[C dogmaESI](open func() C) mcpsdk.ToolHandlerFor[GetDogmaAttributeInput, GetDogmaAttributeOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetDogmaAttributeInput) (*mcpsdk.CallToolResult, GetDogmaAttributeOutput, error) {
		out, err := getDogmaAttribute(ctx, open(), in.AttributeID)
		if err != nil {
			return nil, GetDogmaAttributeOutput{}, fmt.Errorf("get dogma attribute: %w", err)
		}
		return nil, out, nil
	}
}

// --- get_dogma_dynamic_item (public) ---

type DynamicItemAttribute struct {
	AttributeID int64   `json:"attributeId"`
	Value       float64 `json:"value"`
}
type DynamicItemEffect struct {
	EffectID  int64 `json:"effectId"`
	IsDefault bool  `json:"isDefault"`
}

type GetDogmaDynamicItemOutput struct {
	TypeID          int64                  `json:"typeId"`
	TypeName        string                 `json:"typeName,omitempty"`
	ItemID          int64                  `json:"itemId"`
	CreatedBy       int64                  `json:"createdBy"`
	MutatorTypeID   int64                  `json:"mutatorTypeId"`
	MutatorTypeName string                 `json:"mutatorTypeName,omitempty"`
	SourceTypeID    int64                  `json:"sourceTypeId"`
	SourceTypeName  string                 `json:"sourceTypeName,omitempty"`
	DogmaAttributes []DynamicItemAttribute `json:"dogmaAttributes"`
	DogmaEffects    []DynamicItemEffect    `json:"dogmaEffects"`
}

func getDogmaDynamicItem(ctx context.Context, client dogmaESI, typeID, itemID int64) (GetDogmaDynamicItemOutput, error) {
	item, err := client.DogmaDynamicItemDetail(ctx, typeID, itemID)
	if err != nil {
		return GetDogmaDynamicItemOutput{}, err
	}
	attrs := make([]DynamicItemAttribute, len(item.DogmaAttributes))
	for i, a := range item.DogmaAttributes {
		attrs[i] = DynamicItemAttribute{AttributeID: a.AttributeId, Value: a.Value}
	}
	effects := make([]DynamicItemEffect, len(item.DogmaEffects))
	for i, e := range item.DogmaEffects {
		effects[i] = DynamicItemEffect{EffectID: e.EffectId, IsDefault: e.IsDefault}
	}
	return GetDogmaDynamicItemOutput{
		TypeID: typeID, ItemID: itemID, CreatedBy: item.CreatedBy, MutatorTypeID: item.MutatorTypeId,
		SourceTypeID: item.SourceTypeId, DogmaAttributes: attrs, DogmaEffects: effects,
	}, nil
}

type GetDogmaDynamicItemInput struct {
	TypeID int64 `json:"typeId" jsonschema:"the mutated item's type ID"`
	ItemID int64 `json:"itemId" jsonschema:"the mutated item's item ID"`
}

// AddGetDogmaDynamicItem registers get_dogma_dynamic_item. open builds the ESI client; any client with the
// dogmaESI methods will do.
func AddGetDogmaDynamicItem[C dogmaESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_dogma_dynamic_item",
		Description: "Get detail for one mutated (\"abyssal\") item, identified by its source type and item ID (public data, no character required).",
	}, getDogmaDynamicItemHandler(userAgent, open, sdeStore))
}

func getDogmaDynamicItemHandler[C dogmaESI](userAgent string, open func() C, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[GetDogmaDynamicItemInput, GetDogmaDynamicItemOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetDogmaDynamicItemInput) (*mcpsdk.CallToolResult, GetDogmaDynamicItemOutput, error) {
		out, err := getDogmaDynamicItem(ctx, open(), in.TypeID, in.ItemID)
		if err != nil {
			return nil, GetDogmaDynamicItemOutput{}, fmt.Errorf("get dogma dynamic item: %w", err)
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, []int64{out.TypeID, out.MutatorTypeID, out.SourceTypeID})
		out.TypeName = names[out.TypeID]
		out.MutatorTypeName = names[out.MutatorTypeID]
		out.SourceTypeName = names[out.SourceTypeID]
		return nil, out, nil
	}
}

// --- list_dogma_effects (public, no input) ---

func listDogmaEffects(ctx context.Context, client dogmaESI) ([]int64, error) {
	return client.DogmaEffects(ctx)
}

type ListDogmaEffectsInput struct{}
type ListDogmaEffectsOutput struct {
	EffectIDs []int64 `json:"effectIds"`
}

// AddListDogmaEffects registers list_dogma_effects. open builds the ESI client; any client with the
// dogmaESI methods will do.
func AddListDogmaEffects[C dogmaESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_dogma_effects",
		Description: "List every dogma effect ID currently defined (public data, no character required).",
	}, listDogmaEffectsHandler(open))
}

func listDogmaEffectsHandler[C dogmaESI](open func() C) mcpsdk.ToolHandlerFor[ListDogmaEffectsInput, ListDogmaEffectsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListDogmaEffectsInput) (*mcpsdk.CallToolResult, ListDogmaEffectsOutput, error) {
		ids, err := listDogmaEffects(ctx, open())
		if err != nil {
			return nil, ListDogmaEffectsOutput{}, fmt.Errorf("list dogma effects: %w", err)
		}
		return nil, ListDogmaEffectsOutput{EffectIDs: ids}, nil
	}
}

// --- get_dogma_effect (public) ---

type EffectModifier struct {
	Func                 string  `json:"func"`
	Domain               *string `json:"domain,omitempty"`
	EffectID             *int64  `json:"effectId,omitempty"`
	ModifiedAttributeID  *int64  `json:"modifiedAttributeId,omitempty"`
	ModifyingAttributeID *int64  `json:"modifyingAttributeId,omitempty"`
	Operator             *int64  `json:"operator,omitempty"`
}

type GetDogmaEffectOutput struct {
	EffectID                 int64            `json:"effectId"`
	Description              *string          `json:"description,omitempty"`
	DisplayName              *string          `json:"displayName,omitempty"`
	Name                     *string          `json:"name,omitempty"`
	DisallowAutoRepeat       *bool            `json:"disallowAutoRepeat,omitempty"`
	ElectronicChance         *bool            `json:"electronicChance,omitempty"`
	IsAssistance             *bool            `json:"isAssistance,omitempty"`
	IsOffensive              *bool            `json:"isOffensive,omitempty"`
	IsWarpSafe               *bool            `json:"isWarpSafe,omitempty"`
	Published                *bool            `json:"published,omitempty"`
	RangeChance              *bool            `json:"rangeChance,omitempty"`
	DischargeAttributeID     *int64           `json:"dischargeAttributeId,omitempty"`
	DurationAttributeID      *int64           `json:"durationAttributeId,omitempty"`
	EffectCategory           *int64           `json:"effectCategory,omitempty"`
	FalloffAttributeID       *int64           `json:"falloffAttributeId,omitempty"`
	IconID                   *int64           `json:"iconId,omitempty"`
	PostExpression           *int64           `json:"postExpression,omitempty"`
	PreExpression            *int64           `json:"preExpression,omitempty"`
	RangeAttributeID         *int64           `json:"rangeAttributeId,omitempty"`
	TrackingSpeedAttributeID *int64           `json:"trackingSpeedAttributeId,omitempty"`
	Modifiers                []EffectModifier `json:"modifiers,omitempty"`
}

func getDogmaEffect(ctx context.Context, client dogmaESI, effectID int64) (GetDogmaEffectOutput, error) {
	effect, err := client.DogmaEffectDetail(ctx, effectID)
	if err != nil {
		return GetDogmaEffectOutput{}, err
	}
	modifiers := make([]EffectModifier, len(effect.Modifiers))
	for i, m := range effect.Modifiers {
		modifiers[i] = EffectModifier{
			Func: m.Func, Domain: m.Domain, EffectID: m.EffectId,
			ModifiedAttributeID: m.ModifiedAttributeId, ModifyingAttributeID: m.ModifyingAttributeId, Operator: m.Operator,
		}
	}
	return GetDogmaEffectOutput{
		EffectID: effect.EffectId, Description: effect.Description, DisplayName: effect.DisplayName, Name: effect.Name,
		DisallowAutoRepeat: effect.DisallowAutoRepeat, ElectronicChance: effect.ElectronicChance,
		IsAssistance: effect.IsAssistance, IsOffensive: effect.IsOffensive, IsWarpSafe: effect.IsWarpSafe,
		Published: effect.Published, RangeChance: effect.RangeChance, DischargeAttributeID: effect.DischargeAttributeId,
		DurationAttributeID: effect.DurationAttributeId, EffectCategory: effect.EffectCategory,
		FalloffAttributeID: effect.FalloffAttributeId, IconID: effect.IconId, PostExpression: effect.PostExpression,
		PreExpression: effect.PreExpression, RangeAttributeID: effect.RangeAttributeId,
		TrackingSpeedAttributeID: effect.TrackingSpeedAttributeId, Modifiers: modifiers,
	}, nil
}

type GetDogmaEffectInput struct {
	EffectID int64 `json:"effectId" jsonschema:"the dogma effect ID to look up"`
}

// AddGetDogmaEffect registers get_dogma_effect. open builds the ESI client; any client with the
// dogmaESI methods will do.
func AddGetDogmaEffect[C dogmaESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_dogma_effect",
		Description: "Get detail for one dogma effect (public data, no character required).",
	}, getDogmaEffectHandler(open))
}

func getDogmaEffectHandler[C dogmaESI](open func() C) mcpsdk.ToolHandlerFor[GetDogmaEffectInput, GetDogmaEffectOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetDogmaEffectInput) (*mcpsdk.CallToolResult, GetDogmaEffectOutput, error) {
		out, err := getDogmaEffect(ctx, open(), in.EffectID)
		if err != nil {
			return nil, GetDogmaEffectOutput{}, fmt.Errorf("get dogma effect: %w", err)
		}
		return nil, out, nil
	}
}
