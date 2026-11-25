package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// --- list_character_skinr_licenses (private) ---

type SkinrLicense struct {
	SkinrID     string `json:"skinrId"`
	Activated   bool   `json:"activated"`
	Unactivated int64  `json:"unactivated"`
}

// characterSkinrLicensesESI is the ESI call list_character_skinr_licenses makes.
type characterSkinrLicensesESI interface {
	CharacterSkinrLicenses(ctx context.Context) ([]esiapi.CharactersCosmeticsSkinrItem, error)
}

type ListCharacterSkinrLicensesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterSkinrLicensesOutput struct {
	CharacterID int32          `json:"characterId"`
	Licenses    []SkinrLicense `json:"licenses"`
}

// AddListCharacterSkinrLicenses registers list_character_skinr_licenses. open builds the ESI client; any client with the
// characterSkinrLicensesESI methods will do.
func AddListCharacterSkinrLicenses[C characterSkinrLicensesESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_skinr_licenses",
		Description: "List the SKINR (ship skin) licenses a logged-in character owns, with activated/unactivated counts.",
	}, listCharacterSkinrLicensesHandler(open, store))
}

func listCharacterSkinrLicensesHandler[C characterSkinrLicensesESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterSkinrLicensesInput, ListCharacterSkinrLicensesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterSkinrLicensesInput) (*mcpsdk.CallToolResult, ListCharacterSkinrLicensesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterSkinrLicensesOutput{}, err
		}
		if !found {
			return nil, ListCharacterSkinrLicensesOutput{}, fmt.Errorf("list character skinr licenses for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterSkinrLicensesOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		licenses, err := client.CharacterSkinrLicenses(ctx)
		if err != nil {
			return nil, ListCharacterSkinrLicensesOutput{}, fmt.Errorf("list character skinr licenses: %w", err)
		}
		out := make([]SkinrLicense, len(licenses))
		for i, l := range licenses {
			out[i] = SkinrLicense{SkinrID: l.SkinrId, Activated: l.Activated, Unactivated: l.Unactivated}
		}
		return nil, ListCharacterSkinrLicensesOutput{CharacterID: in.CharacterID, Licenses: out}, nil
	}
}

// --- list_character_skinr_component_licenses (private, SDE-enriched) ---

// SkinrComponentLicense flattens CharactersCosmeticsSkinrComponentsItem. Its Runs field is a
// 2-variant oneOf in ESI ({"remaining": n} or {"unlimited": true}); Unlimited and RunsRemaining
// carry whichever variant ESI returned.
type SkinrComponentLicense struct {
	ComponentID   int64  `json:"componentId"`
	ComponentName string `json:"componentName,omitempty"`
	Type          string `json:"type"`
	Unlimited     bool   `json:"unlimited"`
	RunsRemaining *int64 `json:"runsRemaining,omitempty"`
}

// characterSkinrComponentLicensesESI is the ESI call list_character_skinr_component_licenses makes.
type characterSkinrComponentLicensesESI interface {
	CharacterSkinrComponentLicenses(ctx context.Context) ([]esiapi.CharactersCosmeticsSkinrComponentsItem, error)
}

type ListCharacterSkinrComponentLicensesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterSkinrComponentLicensesOutput struct {
	CharacterID int32                   `json:"characterId"`
	Licenses    []SkinrComponentLicense `json:"licenses"`
}

// AddListCharacterSkinrComponentLicenses registers list_character_skinr_component_licenses. open builds the ESI client; any client with the
// characterSkinrComponentLicensesESI methods will do.
func AddListCharacterSkinrComponentLicenses[C characterSkinrComponentLicensesESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore skinrComponentNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_skinr_component_licenses",
		Description: "List the SKINR component licenses (nanocoatings, patterns) a logged-in character owns, with names and remaining runs.",
	}, listCharacterSkinrComponentLicensesHandler(userAgent, open, store, sdeStore))
}

func listCharacterSkinrComponentLicensesHandler[C characterSkinrComponentLicensesESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore skinrComponentNameCache) mcpsdk.ToolHandlerFor[ListCharacterSkinrComponentLicensesInput, ListCharacterSkinrComponentLicensesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterSkinrComponentLicensesInput) (*mcpsdk.CallToolResult, ListCharacterSkinrComponentLicensesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterSkinrComponentLicensesOutput{}, err
		}
		if !found {
			return nil, ListCharacterSkinrComponentLicensesOutput{}, fmt.Errorf("list character skinr component licenses for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterSkinrComponentLicensesOutput{}, fmt.Errorf("build ESI client: %w", err)
		}
		licenses, err := client.CharacterSkinrComponentLicenses(ctx)
		if err != nil {
			return nil, ListCharacterSkinrComponentLicensesOutput{}, fmt.Errorf("list character skinr component licenses: %w", err)
		}
		ids := make([]int64, len(licenses))
		for i, l := range licenses {
			ids[i] = l.ComponentId
		}
		names := resolveSkinrComponentNames(ctx, sdeStore, userAgent, ids)
		out := make([]SkinrComponentLicense, len(licenses))
		for i, l := range licenses {
			out[i] = SkinrComponentLicense{ComponentID: l.ComponentId, ComponentName: names[l.ComponentId], Type: l.Type}
			if l.Runs.Unlimited != nil && l.Runs.Unlimited.Unlimited != nil {
				out[i].Unlimited = *l.Runs.Unlimited.Unlimited
			}
			if l.Runs.Remaining != nil {
				out[i].RunsRemaining = l.Runs.Remaining.Remaining
			}
		}
		return nil, ListCharacterSkinrComponentLicensesOutput{CharacterID: in.CharacterID, Licenses: out}, nil
	}
}

// --- get_skinr (public, SDE-enriched) ---

type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Vector4 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	W float64 `json:"w"`
}

type SkinrPatternProjection struct {
	Slot1 bool `json:"slot1"`
	Slot2 bool `json:"slot2"`
	Slot3 bool `json:"slot3"`
	Slot4 bool `json:"slot4"`
}

type SkinrPatternConfiguration struct {
	Mirrored   bool                   `json:"mirrored"`
	Projection SkinrPatternProjection `json:"projection"`
	Position   Vector3                `json:"position"`
	Rotation   Vector4                `json:"rotation"`
	Scaling    Vector3                `json:"scaling"`
}

// SkinrLayoutSlot flattens CosmeticsSkinrLayoutslot, whose Configuration is a 2-variant oneOf
// (nanocoating or pattern). Kind names the active variant; ComponentID is that variant's
// component either way, and Pattern is populated only for Kind "pattern".
type SkinrLayoutSlot struct {
	SlotID        int64                      `json:"slotId"`
	SlotName      string                     `json:"slotName,omitempty"`
	Kind          string                     `json:"kind"`
	ComponentID   int64                      `json:"componentId"`
	ComponentName string                     `json:"componentName,omitempty"`
	Pattern       *SkinrPatternConfiguration `json:"pattern,omitempty"`
}

func toSkinrLayoutSlot(s esiapi.CosmeticsSkinrLayoutslot) SkinrLayoutSlot {
	out := SkinrLayoutSlot{SlotID: s.Id}
	switch cfg := s.Configuration; {
	case cfg.Nanocoating != nil && cfg.Nanocoating.Nanocoating != nil:
		out.Kind, out.ComponentID = "nanocoating", cfg.Nanocoating.Nanocoating.Id
	case cfg.Pattern != nil && cfg.Pattern.Pattern != nil:
		p := cfg.Pattern.Pattern
		c := p.Configuration
		out.Kind, out.ComponentID = "pattern", p.Id
		out.Pattern = &SkinrPatternConfiguration{
			Mirrored:   c.Mirrored,
			Projection: SkinrPatternProjection{Slot1: c.Projection.Slot1, Slot2: c.Projection.Slot2, Slot3: c.Projection.Slot3, Slot4: c.Projection.Slot4},
			Position:   Vector3{X: c.Transform.Position.X, Y: c.Transform.Position.Y, Z: c.Transform.Position.Z},
			Rotation:   Vector4{X: c.Transform.Rotation.X, Y: c.Transform.Rotation.Y, Z: c.Transform.Rotation.Z, W: c.Transform.Rotation.W},
			Scaling:    Vector3{X: c.Transform.Scaling.X, Y: c.Transform.Scaling.Y, Z: c.Transform.Scaling.Z},
		}
	}
	return out
}

// skinrDetailESI is the ESI call get_skinr makes.
type skinrDetailESI interface {
	SkinrDetail(ctx context.Context, skinrID string) (*esiapi.CosmeticsSkinr, error)
}

// skinrNameCache is the SDE lookups get_skinr makes: slot and component names, plus the ship's type name.
type skinrNameCache interface {
	typeNameCache
	skinrComponentNameCache
	skinrSlotNameCache
}

type GetSkinrInput struct {
	SkinrID string `json:"skinrId" jsonschema:"the SKINR ID (a hex string, from list_character_skinr_licenses or a Paragon Hub listing)"`
}
type GetSkinrOutput struct {
	SkinrID          string            `json:"skinrId"`
	Name             string            `json:"name"`
	Line             *string           `json:"line,omitempty"`
	CreatorID        int64             `json:"creatorId"`
	ShipTypeID       int64             `json:"shipTypeId"`
	ShipTypeName     string            `json:"shipTypeName,omitempty"`
	TierLevel        int64             `json:"tierLevel"`
	PatternBlendMode string            `json:"patternBlendMode"`
	Slots            []SkinrLayoutSlot `json:"slots"`
}

// AddGetSkinr registers get_skinr. open builds the ESI client; any client with the
// skinrDetailESI methods will do.
func AddGetSkinr[C skinrDetailESI](s *mcpsdk.Server, userAgent string, open func() C, sdeStore skinrNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_skinr",
		Description: "Get a SKINR's full design: ship type, tier, and each cosmetic slot's nanocoating or pattern, with names (public data, no character required).",
	}, getSkinrHandler(userAgent, open, sdeStore))
}

func getSkinrHandler[C skinrDetailESI](userAgent string, open func() C, sdeStore skinrNameCache) mcpsdk.ToolHandlerFor[GetSkinrInput, GetSkinrOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetSkinrInput) (*mcpsdk.CallToolResult, GetSkinrOutput, error) {
		client := open()
		s, err := client.SkinrDetail(ctx, in.SkinrID)
		if err != nil {
			return nil, GetSkinrOutput{}, fmt.Errorf("get skinr: %w", err)
		}
		slots := make([]SkinrLayoutSlot, len(s.Layout.Slots))
		slotIDs := make([]int64, len(slots))
		componentIDs := make([]int64, 0, len(slots))
		for i, raw := range s.Layout.Slots {
			slots[i] = toSkinrLayoutSlot(raw)
			slotIDs[i] = slots[i].SlotID
			if slots[i].Kind != "" {
				componentIDs = append(componentIDs, slots[i].ComponentID)
			}
		}
		slotNames := resolveSkinrSlotNames(ctx, sdeStore, userAgent, slotIDs)
		componentNames := resolveSkinrComponentNames(ctx, sdeStore, userAgent, componentIDs)
		for i := range slots {
			slots[i].SlotName = slotNames[slots[i].SlotID]
			if slots[i].Kind != "" {
				slots[i].ComponentName = componentNames[slots[i].ComponentID]
			}
		}
		typeNames := resolveTypeNames(ctx, sdeStore, userAgent, []int64{s.ShipTypeId})
		return nil, GetSkinrOutput{
			SkinrID: s.Id, Name: s.Name, Line: s.Line, CreatorID: s.CreatorId,
			ShipTypeID: s.ShipTypeId, ShipTypeName: typeNames[s.ShipTypeId], TierLevel: s.Tier.Level,
			PatternBlendMode: s.Layout.PatternBlendMode, Slots: slots,
		}, nil
	}
}
