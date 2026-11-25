package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_dogma_attributes (public, no input) ---

var listDogmaAttributes = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.DogmaAttributes(ctx)
}

type ListDogmaAttributesInput struct{}
type ListDogmaAttributesOutput struct {
	AttributeIDs []int64 `json:"attributeIds"`
}

func listDogmaAttributesHandler(version string) mcpsdk.ToolHandlerFor[ListDogmaAttributesInput, ListDogmaAttributesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListDogmaAttributesInput) (*mcpsdk.CallToolResult, ListDogmaAttributesOutput, error) {
		ids, err := listDogmaAttributes(ctx, sso.UserAgent(version))
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

var getDogmaAttribute = func(ctx context.Context, userAgent string, attributeID int64) (GetDogmaAttributeOutput, error) {
	client := esi.NewPublicClient(userAgent)
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

func getDogmaAttributeHandler(version string) mcpsdk.ToolHandlerFor[GetDogmaAttributeInput, GetDogmaAttributeOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetDogmaAttributeInput) (*mcpsdk.CallToolResult, GetDogmaAttributeOutput, error) {
		out, err := getDogmaAttribute(ctx, sso.UserAgent(version), in.AttributeID)
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

var getDogmaDynamicItem = func(ctx context.Context, userAgent string, typeID, itemID int64) (GetDogmaDynamicItemOutput, error) {
	client := esi.NewPublicClient(userAgent)
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

func getDogmaDynamicItemHandler(version string, sdeStore SDEStore) mcpsdk.ToolHandlerFor[GetDogmaDynamicItemInput, GetDogmaDynamicItemOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetDogmaDynamicItemInput) (*mcpsdk.CallToolResult, GetDogmaDynamicItemOutput, error) {
		out, err := getDogmaDynamicItem(ctx, sso.UserAgent(version), in.TypeID, in.ItemID)
		if err != nil {
			return nil, GetDogmaDynamicItemOutput{}, fmt.Errorf("get dogma dynamic item: %w", err)
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), []int64{out.TypeID, out.MutatorTypeID, out.SourceTypeID})
		out.TypeName = names[out.TypeID]
		out.MutatorTypeName = names[out.MutatorTypeID]
		out.SourceTypeName = names[out.SourceTypeID]
		return nil, out, nil
	}
}

// --- list_dogma_effects (public, no input) ---

var listDogmaEffects = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.DogmaEffects(ctx)
}

type ListDogmaEffectsInput struct{}
type ListDogmaEffectsOutput struct {
	EffectIDs []int64 `json:"effectIds"`
}

func listDogmaEffectsHandler(version string) mcpsdk.ToolHandlerFor[ListDogmaEffectsInput, ListDogmaEffectsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListDogmaEffectsInput) (*mcpsdk.CallToolResult, ListDogmaEffectsOutput, error) {
		ids, err := listDogmaEffects(ctx, sso.UserAgent(version))
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

var getDogmaEffect = func(ctx context.Context, userAgent string, effectID int64) (GetDogmaEffectOutput, error) {
	client := esi.NewPublicClient(userAgent)
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

func getDogmaEffectHandler(version string) mcpsdk.ToolHandlerFor[GetDogmaEffectInput, GetDogmaEffectOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetDogmaEffectInput) (*mcpsdk.CallToolResult, GetDogmaEffectOutput, error) {
		out, err := getDogmaEffect(ctx, sso.UserAgent(version), in.EffectID)
		if err != nil {
			return nil, GetDogmaEffectOutput{}, fmt.Errorf("get dogma effect: %w", err)
		}
		return nil, out, nil
	}
}
