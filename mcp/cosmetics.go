package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- list_character_skinr_licenses (private) ---

type SkinrLicense struct {
	SkinrID     string `json:"skinrId"`
	Activated   bool   `json:"activated"`
	Unactivated int64  `json:"unactivated"`
}

var listCharacterSkinrLicenses = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCosmeticsSkinrItem, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterSkinrLicenses(ctx)
}

type ListCharacterSkinrLicensesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterSkinrLicensesOutput struct {
	CharacterID int32          `json:"characterId"`
	Licenses    []SkinrLicense `json:"licenses"`
}

func listCharacterSkinrLicensesHandler(version, clientID string, store ConfigStore) mcpsdk.ToolHandlerFor[ListCharacterSkinrLicensesInput, ListCharacterSkinrLicensesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterSkinrLicensesInput) (*mcpsdk.CallToolResult, ListCharacterSkinrLicensesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterSkinrLicensesOutput{}, err
		}
		if !found {
			return nil, ListCharacterSkinrLicensesOutput{}, fmt.Errorf("list character skinr licenses for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		licenses, err := listCharacterSkinrLicenses(ctx, clientID, in.CharacterID, sso.UserAgent(version))
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

var listCharacterSkinrComponentLicenses = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCosmeticsSkinrComponentsItem, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterSkinrComponentLicenses(ctx)
}

type ListCharacterSkinrComponentLicensesInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterSkinrComponentLicensesOutput struct {
	CharacterID int32                   `json:"characterId"`
	Licenses    []SkinrComponentLicense `json:"licenses"`
}

func listCharacterSkinrComponentLicensesHandler(version, clientID string, store ConfigStore, sdeStore SDEStore) mcpsdk.ToolHandlerFor[ListCharacterSkinrComponentLicensesInput, ListCharacterSkinrComponentLicensesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterSkinrComponentLicensesInput) (*mcpsdk.CallToolResult, ListCharacterSkinrComponentLicensesOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterSkinrComponentLicensesOutput{}, err
		}
		if !found {
			return nil, ListCharacterSkinrComponentLicensesOutput{}, fmt.Errorf("list character skinr component licenses for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		licenses, err := listCharacterSkinrComponentLicenses(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterSkinrComponentLicensesOutput{}, fmt.Errorf("list character skinr component licenses: %w", err)
		}
		ids := make([]int64, len(licenses))
		for i, l := range licenses {
			ids[i] = l.ComponentId
		}
		names := resolveSkinrComponentNames(ctx, sdeStore, sso.UserAgent(version), ids)
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

var getSkinr = func(ctx context.Context, userAgent, skinrID string) (*esiapi.CosmeticsSkinr, error) {
	return esi.NewPublicClient(userAgent).SkinrDetail(ctx, skinrID)
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

func getSkinrHandler(version string, sdeStore SDEStore) mcpsdk.ToolHandlerFor[GetSkinrInput, GetSkinrOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetSkinrInput) (*mcpsdk.CallToolResult, GetSkinrOutput, error) {
		userAgent := sso.UserAgent(version)
		s, err := getSkinr(ctx, userAgent, in.SkinrID)
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
