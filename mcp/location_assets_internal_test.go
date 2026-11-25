package mcp

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

func stubTypeBasics(t *testing.T, basics map[int64]sde.TypeBasic) {
	t.Helper()
	orig := resolveTypeBasics
	t.Cleanup(func() { resolveTypeBasics = orig })
	resolveTypeBasics = func(context.Context, typeBasicsCache, string, []int64) map[int64]sde.TypeBasic { return basics }
}

func TestListLocationAssets_CombinesCharactersPagesAndTotals(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Hauler"}, config.Character{ID: 2, Name: "Alt"})
	stubTypeBasics(t, map[int64]sde.TypeBasic{
		34:  {TypeID: 34, Volume: 0.01, AssembledVolume: 0.01},
		587: {TypeID: 587, CategoryID: 6, GroupID: 25, Volume: 2500, AssembledVolume: 27289},
	})
	assets := openLocationAssets(func(characterID, page int32) ([]AssetItem, int32, error) {
		switch {
		case characterID == 1 && page == 1:
			return []AssetItem{{ItemID: 10, TypeID: 587, LocationID: 60003760, LocationFlag: "Hangar", Quantity: 1, IsSingleton: true}}, 2, nil
		case characterID == 1 && page == 2:
			return []AssetItem{{ItemID: 11, TypeID: 34, LocationID: 10, LocationFlag: "Cargo", Quantity: 1000}}, 2, nil
		case characterID == 2:
			return []AssetItem{
				{ItemID: 20, TypeID: 34, LocationID: 60003760, LocationFlag: "Hangar", Quantity: 500},
				{ItemID: 21, TypeID: 34, LocationID: 60008494, LocationFlag: "Hangar", Quantity: 5},
			}, 1, nil
		}
		return nil, 0, errors.New("unexpected page")
	})

	_, out, err := listLocationAssetsHandler("v1", assets, store, nil)(context.Background(), nil, ListLocationAssetsInput{LocationID: 60003760})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(out.Items) != 3 || out.Items[0].ItemID != 10 || out.Items[1].ParentItemID != 10 {
		t.Errorf("items = %+v", out.Items)
	}
	if out.Totals.AssembledShips != 1 || !near(out.Totals.ItemsM3, 5) || !near(out.Totals.ShipContentsM3, 10) || !near(out.Totals.ShipsPackagedM3, 2500) {
		t.Errorf("totals = %+v", out.Totals)
	}
	if len(out.Sources) != 2 || out.Sources[0].Owner != "Hauler" || out.Sources[0].Items != 2 || out.Sources[1].Owner != "Alt" || out.Sources[1].Items != 1 {
		t.Errorf("sources = %+v", out.Sources)
	}
	if h, a := out.Sources[0].Totals, out.Sources[1].Totals; h.AssembledShips != 1 || !near(h.ShipContentsM3, 10) || !near(h.ItemsM3, 0) || a.AssembledShips != 0 || !near(a.ItemsM3, 5) {
		t.Errorf("per-owner totals: Hauler %+v, Alt %+v", h, a)
	}
}

func TestListLocationAssets_OneOwnerFailingIsAWarning(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Hauler"}, config.Character{ID: 2, Name: "Alt"})
	stubTypeBasics(t, map[int64]sde.TypeBasic{34: {TypeID: 34, Volume: 0.01, AssembledVolume: 0.01}})
	assets := openLocationAssets(func(characterID, page int32) ([]AssetItem, int32, error) {
		if characterID == 2 {
			return nil, 0, errors.New("missing scope")
		}
		return []AssetItem{{ItemID: 10, TypeID: 34, LocationID: 60003760, Quantity: 1}}, 1, nil
	})
	_, out, err := listLocationAssetsHandler("v1", assets, store, nil)(context.Background(), nil, ListLocationAssetsInput{LocationID: 60003760})
	if err != nil || len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "Alt") || len(out.Items) != 1 {
		t.Errorf("out = %+v, err %v", out, err)
	}
}

func TestListLocationAssets_WarnsWhenPagesAreCapped(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Hauler"})
	stubTypeBasics(t, map[int64]sde.TypeBasic{34: {TypeID: 34, Volume: 0.01, AssembledVolume: 0.01}})
	calls := 0
	assets := openLocationAssets(func(characterID, page int32) ([]AssetItem, int32, error) {
		calls++
		return []AssetItem{{ItemID: int64(page), TypeID: 34, LocationID: 60003760, LocationFlag: "Hangar", Quantity: 1}}, maxAssetPages + 10, nil
	})
	_, out, err := listLocationAssetsHandler("v1", assets, store, nil)(context.Background(), nil, ListLocationAssetsInput{LocationID: 60003760})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if calls != maxAssetPages || len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "Hauler") {
		t.Errorf("calls = %d, warnings = %q, want %d calls and one warning naming the owner", calls, out.Warnings, maxAssetPages)
	}
}

func TestListLocationAssets_DuplicateCharacterCountedOnce(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Hauler"})
	stubTypeBasics(t, map[int64]sde.TypeBasic{34: {TypeID: 34, Volume: 0.01, AssembledVolume: 0.01}})
	assets := openLocationAssets(func(characterID, page int32) ([]AssetItem, int32, error) {
		return []AssetItem{{ItemID: 10, TypeID: 34, LocationID: 60003760, LocationFlag: "Hangar", Quantity: 100}}, 1, nil
	})
	_, out, err := listLocationAssetsHandler("v1", assets, store, nil)(context.Background(), nil, ListLocationAssetsInput{LocationID: 60003760, CharacterIDs: []int32{1, 1}})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(out.Items) != 1 || !near(out.Totals.ItemsM3, 1) {
		t.Errorf("items = %+v totals = %+v, want the character's one stack counted once", out.Items, out.Totals)
	}
}

func TestListLocationAssets_WarnsAboutTypesMissingFromSDE(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Hauler"})
	stubTypeBasics(t, nil)
	assets := openLocationAssets(func(characterID, page int32) ([]AssetItem, int32, error) {
		return []AssetItem{
			{ItemID: 10, TypeID: 99999, LocationID: 60003760, LocationFlag: "Hangar", Quantity: 1},
			{ItemID: 11, TypeID: 88888, LocationID: 60008494, LocationFlag: "Hangar", Quantity: 1},
		}, 1, nil
	})
	_, out, err := listLocationAssetsHandler("v1", assets, store, nil)(context.Background(), nil, ListLocationAssetsInput{LocationID: 60003760})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "99999") || strings.Contains(out.Warnings[0], "88888") {
		t.Errorf("warnings = %q, want one naming only the type at this location", out.Warnings)
	}
}

func TestListLocationAssets_RejectsBadInput(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Hauler"})
	h := listLocationAssetsHandler("v1", noNarrow[locationAssetsESI](t), store, nil)
	for name, in := range map[string]ListLocationAssetsInput{
		"no location":                   {},
		"unknown character":             {LocationID: 1, CharacterIDs: []int32{9}},
		"corporation without character": {LocationID: 1, CorporationID: 98000001},
	} {
		if _, _, err := h(context.Background(), nil, in); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestListAssets_AddsVolume(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Pilot"})
	stubTypeBasics(t, map[int64]sde.TypeBasic{587: {TypeID: 587, CategoryID: 6, Volume: 2500, AssembledVolume: 27289}})
	assets := openLocationAssets(func(characterID, page int32) ([]AssetItem, int32, error) {
		return []AssetItem{{ItemID: 1, TypeID: 587, Quantity: 1, IsSingleton: true}, {ItemID: 2, TypeID: 587, Quantity: 3}}, 1, nil
	})
	_, out, err := listAssetsHandler("v1", openAssets(assets), store, nil)(context.Background(), nil, ListAssetsInput{CharacterID: 7})
	if err != nil || !near(out.Assets[0].VolumeM3, 27289) || !near(out.Assets[1].VolumeM3, 7500) {
		t.Errorf("assets = %+v, err %v", out.Assets, err)
	}
}

// locationAssetsFake is the client list_location_assets opens.
type locationAssetsFake struct {
	fakeAssetsESI
	fakeCorporationAssetsESI
}

// openLocationAssets opens, for each character, a client whose asset pages
// come from pages(characterID, page).
func openLocationAssets(pages func(characterID, page int32) ([]AssetItem, int32, error)) func(context.Context, int32) (locationAssetsESI, error) {
	return func(_ context.Context, characterID int32) (locationAssetsESI, error) {
		return locationAssetsFake{fakeAssetsESI: fakeAssetsESI{assets: func(_ context.Context, page int32) ([]esiapi.CharactersCharacterIdAssetsGetInner, int32, error) {
			items, total, err := pages(characterID, page)
			return toESIAssets(items), total, err
		}}}, nil
	}
}

// openAssets narrows a list_location_assets opener to list_assets' client.
func openAssets(open func(context.Context, int32) (locationAssetsESI, error)) func(context.Context, int32) (assetsESI, error) {
	return func(ctx context.Context, characterID int32) (assetsESI, error) { return open(ctx, characterID) }
}

func TestListLocationAssets_UnopenableClient(t *testing.T) {
	store := characterStore(t, config.Character{ID: 1, Name: "Hauler"})
	stubTypeBasics(t, nil)
	_, _, err := listLocationAssetsHandler("v1", failNarrow[locationAssetsESI](errors.New("no refresh token")), store, nil)(context.Background(), nil, ListLocationAssetsInput{LocationID: 60003760})
	if err == nil || !strings.Contains(err.Error(), "couldn't read Hauler's assets: no refresh token") {
		t.Errorf("err = %v, want the open error", err)
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }
