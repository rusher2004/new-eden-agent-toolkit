package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

//go:fix inline
func ptr[T any](v T) *T { return new(v) }

func TestListCharacterSkinrLicensesHandler_MapsFields(t *testing.T) {
	store := storeWithCharacter(t)
	orig := listCharacterSkinrLicenses
	t.Cleanup(func() { listCharacterSkinrLicenses = orig })
	listCharacterSkinrLicenses = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCosmeticsSkinrItem, error) {
		return []esiapi.CharactersCosmeticsSkinrItem{{SkinrId: "abc", Activated: true, Unactivated: 2}}, nil
	}

	_, out, err := listCharacterSkinrLicensesHandler("v1", "client-id", store)(context.Background(), nil, ListCharacterSkinrLicensesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Licenses) != 1 || out.Licenses[0] != (SkinrLicense{SkinrID: "abc", Activated: true, Unactivated: 2}) {
		t.Errorf("out = %+v, want character 7 with one activated abc license, 2 unactivated", out)
	}
}

func TestListCharacterSkinrLicensesHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	orig := listCharacterSkinrLicenses
	t.Cleanup(func() { listCharacterSkinrLicenses = orig })
	listCharacterSkinrLicenses = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCosmeticsSkinrItem, error) {
		t.Fatal("listCharacterSkinrLicenses should not be called for an unknown character")
		return nil, nil
	}

	_, _, err := listCharacterSkinrLicensesHandler("v1", "client-id", store)(context.Background(), nil, ListCharacterSkinrLicensesInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterSkinrComponentLicensesHandler_FlattensRunsAndEnrichesNames(t *testing.T) {
	store := storeWithCharacter(t)
	origList, origNames := listCharacterSkinrComponentLicenses, resolveSkinrComponentNames
	t.Cleanup(func() { listCharacterSkinrComponentLicenses, resolveSkinrComponentNames = origList, origNames })
	listCharacterSkinrComponentLicenses = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCosmeticsSkinrComponentsItem, error) {
		return []esiapi.CharactersCosmeticsSkinrComponentsItem{
			{ComponentId: 1254, Type: "material", Runs: esiapi.CharactersCosmeticsSkinrComponentsItemRuns{Remaining: &esiapi.Remaining{Remaining: new(int64(3))}}},
			{ComponentId: 1328, Type: "pattern", Runs: esiapi.CharactersCosmeticsSkinrComponentsItemRuns{Unlimited: &esiapi.Unlimited{Unlimited: new(true)}}},
		}, nil
	}
	resolveSkinrComponentNames = func(ctx context.Context, sdeStore skinrComponentNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1254: "Charcoal Covenant Satin"} // 1328 deliberately unresolved
	}

	_, out, err := listCharacterSkinrComponentLicensesHandler("v1", "client-id", store, nil)(context.Background(), nil, ListCharacterSkinrComponentLicensesInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Licenses) != 2 {
		t.Fatalf("out = %+v, want 2 licenses", out)
	}
	if l := out.Licenses[0]; l.ComponentName != "Charcoal Covenant Satin" || l.Unlimited || l.RunsRemaining == nil || *l.RunsRemaining != 3 {
		t.Errorf("licenses[0] = %+v, want named, 3 runs remaining, not unlimited", l)
	}
	if l := out.Licenses[1]; l.ComponentName != "" || !l.Unlimited || l.RunsRemaining != nil {
		t.Errorf("licenses[1] = %+v, want unnamed (unresolved), unlimited, no runs count", l)
	}
}

func TestGetSkinrHandler_FlattensSlotsAndEnriches(t *testing.T) {
	origGet, origTypes, origSlots, origComponents := getSkinr, resolveTypeNames, resolveSkinrSlotNames, resolveSkinrComponentNames
	t.Cleanup(func() {
		getSkinr, resolveTypeNames, resolveSkinrSlotNames, resolveSkinrComponentNames = origGet, origTypes, origSlots, origComponents
	})
	getSkinr = func(ctx context.Context, userAgent, skinrID string) (*esiapi.CosmeticsSkinr, error) {
		return &esiapi.CosmeticsSkinr{
			Id: skinrID, Name: "Semper Fi", Line: new("Combat"), CreatorId: 2119410889, ShipTypeId: 78576,
			Tier: esiapi.CosmeticsSkinrTier{Level: 13},
			Layout: esiapi.CosmeticsSkinrLayout{PatternBlendMode: "normal", Slots: []esiapi.CosmeticsSkinrLayoutslot{
				{Id: 1, Configuration: esiapi.CosmeticsSkinrLayoutslotConfiguration{Nanocoating: &esiapi.Nanocoating{Nanocoating: &esiapi.CosmeticsSkinrSlotnanocoating{Id: 1254}}}},
				{Id: 5, Configuration: esiapi.CosmeticsSkinrLayoutslotConfiguration{Pattern: &esiapi.Pattern{Pattern: &esiapi.CosmeticsSkinrSlotpattern{
					Id: 1328,
					Configuration: esiapi.CosmeticsSkinrPatternconfiguration{
						Mirrored:   true,
						Projection: esiapi.CosmeticsSkinrPatternprojection{Slot2: true, Slot4: true},
						Transform: esiapi.CosmeticsSkinrPatterntransform{
							Position: esiapi.CosmeticsSkinrVector3{X: 1, Y: 2, Z: 3},
							Rotation: esiapi.CosmeticsSkinrVector4{X: 0.1, Y: 0.2, Z: 0.3, W: 0.4},
							Scaling:  esiapi.CosmeticsSkinrVector3{X: 9, Y: 9, Z: 9},
						},
					},
				}}}},
			}},
		}, nil
	}
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{78576: "Some Ship"}
	}
	resolveSkinrSlotNames = func(ctx context.Context, sdeStore skinrSlotNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1: "Primary Slot", 5: "Primary Pattern Slot"}
	}
	resolveSkinrComponentNames = func(ctx context.Context, sdeStore skinrComponentNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{1254: "Charcoal Covenant Satin", 1328: "Lines of Convergence"}
	}

	_, out, err := getSkinrHandler("v1", nil)(context.Background(), nil, GetSkinrInput{SkinrID: "381156f5"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.SkinrID != "381156f5" || out.Name != "Semper Fi" || out.Line == nil || *out.Line != "Combat" ||
		out.ShipTypeID != 78576 || out.ShipTypeName != "Some Ship" || out.TierLevel != 13 || out.PatternBlendMode != "normal" {
		t.Errorf("out = %+v, want Semper Fi / Combat / Some Ship (78576) / tier 13 / normal", out)
	}
	if len(out.Slots) != 2 {
		t.Fatalf("slots = %+v, want 2", out.Slots)
	}
	if s := out.Slots[0]; s.SlotID != 1 || s.SlotName != "Primary Slot" || s.Kind != "nanocoating" || s.ComponentID != 1254 || s.ComponentName != "Charcoal Covenant Satin" || s.Pattern != nil {
		t.Errorf("slots[0] = %+v, want named nanocoating 1254 in Primary Slot, no pattern", s)
	}
	s := out.Slots[1]
	if s.SlotID != 5 || s.SlotName != "Primary Pattern Slot" || s.Kind != "pattern" || s.ComponentID != 1328 || s.ComponentName != "Lines of Convergence" || s.Pattern == nil {
		t.Fatalf("slots[1] = %+v, want named pattern 1328 in Primary Pattern Slot", s)
	}
	want := SkinrPatternConfiguration{
		Mirrored: true, Projection: SkinrPatternProjection{Slot2: true, Slot4: true},
		Position: Vector3{X: 1, Y: 2, Z: 3}, Rotation: Vector4{X: 0.1, Y: 0.2, Z: 0.3, W: 0.4}, Scaling: Vector3{X: 9, Y: 9, Z: 9},
	}
	if *s.Pattern != want {
		t.Errorf("slots[1].Pattern = %+v, want %+v", *s.Pattern, want)
	}
}

func TestGetSkinrHandler_PropagatesError(t *testing.T) {
	orig := getSkinr
	t.Cleanup(func() { getSkinr = orig })
	getSkinr = func(ctx context.Context, userAgent, skinrID string) (*esiapi.CosmeticsSkinr, error) {
		return nil, errors.New("boom")
	}
	if _, _, err := getSkinrHandler("v1", nil)(context.Background(), nil, GetSkinrInput{SkinrID: "x"}); err == nil {
		t.Fatal("handler returned nil error, want propagated error")
	}
}
