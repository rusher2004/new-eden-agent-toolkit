package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// --- list_character_market_orders ---

func TestListCharacterMarketOrdersHandler_MapsFields(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	issued := time.Unix(1000, 0)
	escrow := 100.5
	isBuyOrder := true
	minVolume := int64(1)
	fake := fakeCharacterMarketOrdersESI{characterMarketOrders: func(ctx context.Context) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		return []esiapi.CharactersCharacterIdOrdersGetInner{{
			Duration: 90, Escrow: &escrow, IsBuyOrder: &isBuyOrder, IsCorporation: false,
			Issued: issued, LocationId: 60003760, MinVolume: &minVolume, OrderId: 123456,
			Price: 5.55, Range: "region", RegionId: 10000002, TypeId: 34,
			VolumeRemain: 500, VolumeTotal: 1000,
		}}, nil
	}}

	handler := listCharacterMarketOrdersHandler("v1", openNarrow[characterMarketOrdersESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMarketOrdersInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 with one order", out)
	}
	o := out.Orders[0]
	if o.Duration != 90 || o.Escrow == nil || *o.Escrow != escrow || o.IsBuyOrder == nil || !*o.IsBuyOrder ||
		o.IsCorporation == nil || *o.IsCorporation || !o.Issued.Equal(issued) || o.LocationID != 60003760 ||
		o.MinVolume == nil || *o.MinVolume != minVolume || o.OrderID != 123456 || o.Price != 5.55 ||
		o.Range != "region" || o.RegionID != 10000002 || o.TypeID != 34 || o.VolumeRemain != 500 ||
		o.VolumeTotal != 1000 || o.State != "" || o.IssuedBy != nil || o.WalletDivision != nil {
		t.Errorf("order = %+v, want Duration=90 Escrow=%v IsBuyOrder=true IsCorporation=false Issued=%v LocationID=60003760 MinVolume=%v OrderID=123456 Price=5.55 Range=region RegionID=10000002 TypeID=34 VolumeRemain=500 VolumeTotal=1000 State=\"\" IssuedBy=nil WalletDivision=nil",
			o, escrow, issued, minVolume)
	}
}

func TestListCharacterMarketOrdersHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharacterMarketOrdersHandler("v1", noNarrow[characterMarketOrdersESI](t), store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterMarketOrdersInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_character_market_order_history ---

func TestListCharacterMarketOrderHistoryHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	issued := time.Unix(2000, 0)
	fake := fakeCharacterMarketOrderHistoryESI{characterMarketOrderHistory: func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdOrdersHistoryGetInner{{
			Duration: 90, IsCorporation: true, Issued: issued, LocationId: 60003760,
			OrderId: 654321, Price: 5.55, Range: "region", RegionId: 10000002,
			State: "expired", TypeId: 34, VolumeRemain: 0, VolumeTotal: 1000,
		}}, 3, nil
	}}

	handler := listCharacterMarketOrderHistoryHandler("v1", openNarrow[characterMarketOrderHistoryESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMarketOrderHistoryInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.Page != 1 || out.TotalPages != 3 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 Page=1 TotalPages=3 with one order", out)
	}
	o := out.Orders[0]
	if o.Duration != 90 || o.IsCorporation == nil || !*o.IsCorporation || !o.Issued.Equal(issued) ||
		o.LocationID != 60003760 || o.OrderID != 654321 || o.Price != 5.55 || o.Range != "region" ||
		o.RegionID != 10000002 || o.State != "expired" || o.TypeID != 34 || o.VolumeRemain != 0 ||
		o.VolumeTotal != 1000 || o.Escrow != nil || o.IsBuyOrder != nil || o.MinVolume != nil ||
		o.IssuedBy != nil || o.WalletDivision != nil {
		t.Errorf("order = %+v, want Duration=90 IsCorporation=true Issued=%v LocationID=60003760 OrderID=654321 Price=5.55 Range=region RegionID=10000002 State=expired TypeID=34 VolumeRemain=0 VolumeTotal=1000",
			o, issued)
	}
}

func TestListCharacterMarketOrderHistoryHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCharacterMarketOrderHistoryHandler("v1", noNarrow[characterMarketOrderHistoryESI](t), store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterMarketOrderHistoryInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_corporation_market_orders ---

func TestListCorporationMarketOrdersHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	issued := time.Unix(3000, 0)
	fake := fakeCorporationMarketOrdersESI{corporationMarketOrders: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersGetInner{{
			Duration: 90, Issued: issued, IssuedBy: 998877, LocationId: 60003760,
			OrderId: 111222, Price: 5.55, Range: "region", RegionId: 10000002,
			TypeId: 34, VolumeRemain: 500, VolumeTotal: 1000, WalletDivision: 1,
		}}, 2, nil
	}}

	handler := listCorporationMarketOrdersHandler("v1", openNarrow[corporationMarketOrdersESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMarketOrdersInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 2 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=2 with one order", out)
	}
	o := out.Orders[0]
	if o.Duration != 90 || !o.Issued.Equal(issued) || o.IssuedBy == nil || *o.IssuedBy != 998877 ||
		o.LocationID != 60003760 || o.OrderID != 111222 || o.Price != 5.55 || o.Range != "region" ||
		o.RegionID != 10000002 || o.TypeID != 34 || o.VolumeRemain != 500 || o.VolumeTotal != 1000 ||
		o.WalletDivision == nil || *o.WalletDivision != 1 ||
		o.IsCorporation != nil || o.State != "" || o.Escrow != nil || o.IsBuyOrder != nil || o.MinVolume != nil {
		t.Errorf("order = %+v, want Duration=90 Issued=%v IssuedBy=998877 LocationID=60003760 OrderID=111222 Price=5.55 Range=region RegionID=10000002 TypeID=34 VolumeRemain=500 VolumeTotal=1000 WalletDivision=1",
			o, issued)
	}
}

func TestListCorporationMarketOrdersHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationMarketOrdersHandler("v1", noNarrow[corporationMarketOrdersESI](t), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationMarketOrdersInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_corporation_market_order_history ---

func TestListCorporationMarketOrderHistoryHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	issued := time.Unix(4000, 0)
	fake := fakeCorporationMarketOrderHistoryESI{corporationMarketOrderHistory: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersHistoryGetInner{{
			Duration: 90, Issued: issued, IssuedBy: nil, LocationId: 60003760,
			OrderId: 333444, Price: 5.55, Range: "region", RegionId: 10000002,
			State: "cancelled", TypeId: 34, VolumeRemain: 0, VolumeTotal: 1000, WalletDivision: 2,
		}}, 1, nil
	}}

	handler := listCorporationMarketOrderHistoryHandler("v1", openNarrow[corporationMarketOrderHistoryESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.CorporationID != 98000001 || out.Page != 1 || out.TotalPages != 1 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 CorporationID=98000001 Page=1 TotalPages=1 with one order", out)
	}
	o := out.Orders[0]
	// IssuedBy is optional (*int64) for corp order history, unlike the required int64 on
	// list_corporation_market_orders -- this fixture leaves it nil to confirm the handler
	// passes that nil through rather than synthesizing a zero-value pointer.
	if o.Duration != 90 || !o.Issued.Equal(issued) || o.IssuedBy != nil ||
		o.LocationID != 60003760 || o.OrderID != 333444 || o.Price != 5.55 || o.Range != "region" ||
		o.RegionID != 10000002 || o.State != "cancelled" || o.TypeID != 34 || o.VolumeRemain != 0 ||
		o.VolumeTotal != 1000 || o.WalletDivision == nil || *o.WalletDivision != 2 ||
		o.IsCorporation != nil || o.Escrow != nil || o.IsBuyOrder != nil || o.MinVolume != nil {
		t.Errorf("order = %+v, want Duration=90 Issued=%v IssuedBy=nil LocationID=60003760 OrderID=333444 Price=5.55 Range=region RegionID=10000002 State=cancelled TypeID=34 VolumeRemain=0 VolumeTotal=1000 WalletDivision=2",
			o, issued)
	}
}

func TestListCorporationMarketOrderHistoryHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listCorporationMarketOrderHistoryHandler("v1", noNarrow[corporationMarketOrderHistoryESI](t), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_market_groups (public) ---

func TestListMarketGroupsHandler_MapsFields(t *testing.T) {
	fake := fakeMarketGroupsESI{marketGroups: func(ctx context.Context) ([]int64, error) {
		return []int64{1, 2, 3}, nil
	}}

	handler := listMarketGroupsHandler(publicNarrow[marketGroupsESI](fake))
	_, out, err := handler(context.Background(), nil, ListMarketGroupsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.MarketGroupIDs) != 3 || out.MarketGroupIDs[0] != 1 || out.MarketGroupIDs[1] != 2 || out.MarketGroupIDs[2] != 3 {
		t.Errorf("out.MarketGroupIDs = %+v, want [1 2 3]", out.MarketGroupIDs)
	}
}

// --- get_market_group (public) ---

func TestGetMarketGroupHandler_MapsFields(t *testing.T) {
	parentGroupID := int64(1)
	fake := fakeMarketGroupESI{marketGroup: func(_ context.Context, marketGroupID int64) (*esiapi.MarketsGroupsMarketGroupIdGet, error) {
		return &esiapi.MarketsGroupsMarketGroupIdGet{
			MarketGroupId: marketGroupID, Name: "Ammo & Charges", Description: "Ammo & Charges",
			ParentGroupId: &parentGroupID, Types: []int64{10, 11},
		}, nil
	}}

	handler := getMarketGroupHandler(publicNarrow[marketGroupESI](fake))
	_, out, err := handler(context.Background(), nil, GetMarketGroupInput{MarketGroupID: 9})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.MarketGroupID != 9 || out.Name != "Ammo & Charges" || out.Description != "Ammo & Charges" ||
		out.ParentGroupID == nil || *out.ParentGroupID != 1 || len(out.Types) != 2 || out.Types[0] != 10 || out.Types[1] != 11 {
		t.Errorf("out = %+v, want MarketGroupID=9 Name/Description=\"Ammo & Charges\" ParentGroupID=1 Types=[10 11]", out)
	}
}

// --- list_market_prices (public) ---

func TestListMarketPricesHandler_MapsFields(t *testing.T) {
	adjustedPrice := 306.11
	averagePrice := 306.29
	fake := fakeMarketPricesESI{marketPrices: func(_ context.Context) ([]esiapi.MarketsPricesGetInner, error) {
		return []esiapi.MarketsPricesGetInner{{
			AdjustedPrice: &adjustedPrice, AveragePrice: &averagePrice, TypeId: 32772,
		}}, nil
	}}

	handler := listMarketPricesHandler("v1", publicNarrow[marketPricesESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListMarketPricesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Prices) != 1 {
		t.Fatalf("out = %+v, want one price", out)
	}
	p := out.Prices[0]
	if p.TypeID != 32772 || p.AdjustedPrice == nil || *p.AdjustedPrice != adjustedPrice ||
		p.AveragePrice == nil || *p.AveragePrice != averagePrice {
		t.Errorf("price = %+v, want TypeID=32772 AdjustedPrice=%v AveragePrice=%v", p, adjustedPrice, averagePrice)
	}
}

// --- get_market_history (public) ---

func TestGetMarketHistoryHandler_MapsFields(t *testing.T) {
	fake := fakeMarketHistoryESI{marketHistory: func(ctx context.Context, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error) {
		return []esiapi.MarketsRegionIdHistoryGetInner{{
			Average: 5.5, Date: "2026-07-20", Highest: 6.0, Lowest: 5.0, OrderCount: 10, Volume: 1000,
		}}, nil
	}}

	handler := getMarketHistoryHandler(publicNarrow[marketHistoryESI](fake))
	_, out, err := handler(context.Background(), nil, GetMarketHistoryInput{RegionID: 10000002, TypeID: 34})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.RegionID != 10000002 || out.TypeID != 34 || len(out.History) != 1 {
		t.Fatalf("out = %+v, want RegionID=10000002 TypeID=34 with one history entry", out)
	}
	h := out.History[0]
	if h.Average != 5.5 || h.Date != "2026-07-20" || h.Highest != 6.0 || h.Lowest != 5.0 || h.OrderCount != 10 || h.Volume != 1000 {
		t.Errorf("history[0] = %+v, want Average=5.5 Date=2026-07-20 Highest=6.0 Lowest=5.0 OrderCount=10 Volume=1000", h)
	}
}

// --- list_market_orders (public, paginated) ---

func TestListMarketOrdersHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	issued := time.Unix(5000, 0)
	var gotRegionID, gotTypeID int64
	var gotOrderType string
	var gotPage int32
	fake := fakeMarketOrdersESI{marketOrders: func(_ context.Context, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		gotRegionID, gotOrderType, gotTypeID, gotPage = regionID, orderType, typeID, page
		return []esiapi.MarketsRegionIdOrdersGetInner{{
			Duration: 90, IsBuyOrder: true, Issued: issued, LocationId: 60003760,
			MinVolume: 1, OrderId: 555, Price: 5.55, Range: "region", SystemId: 30000142,
			TypeId: 34, VolumeRemain: 500, VolumeTotal: 1000,
		}}, 5, nil
	}}

	handler := listMarketOrdersHandler("v1", publicNarrow[marketOrdersESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListMarketOrdersInput{RegionID: 10000002, OrderType: "buy", TypeID: 34})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotRegionID != 10000002 || gotOrderType != "buy" || gotTypeID != 34 || gotPage != 0 {
		t.Errorf("captured args = regionID=%d orderType=%q typeID=%d page=%d, want 10000002/buy/34/0", gotRegionID, gotOrderType, gotTypeID, gotPage)
	}
	if out.RegionID != 10000002 || out.OrderType != "buy" || out.Page != 1 || out.TotalPages != 5 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want RegionID=10000002 OrderType=buy Page=1 TotalPages=5 with one order", out)
	}
	o := out.Orders[0]
	if o.Duration != 90 || !o.IsBuyOrder || !o.Issued.Equal(issued) || o.LocationID != 60003760 ||
		o.MinVolume != 1 || o.OrderID != 555 || o.Price != 5.55 || o.Range != "region" ||
		o.SystemID != 30000142 || o.TypeID != 34 || o.VolumeRemain != 500 || o.VolumeTotal != 1000 {
		t.Errorf("order = %+v, want Duration=90 IsBuyOrder=true Issued=%v LocationID=60003760 MinVolume=1 OrderID=555 Price=5.55 Range=region SystemID=30000142 TypeID=34 VolumeRemain=500 VolumeTotal=1000",
			o, issued)
	}
}

// --- list_market_types (public, paginated) ---

func TestListMarketTypesHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	fake := fakeMarketTypesESI{marketTypes: func(ctx context.Context, regionID int64, page int32) ([]int64, int32, error) {
		return []int64{34, 35, 36}, 7, nil
	}}

	handler := listMarketTypesHandler("v1", publicNarrow[marketTypesESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListMarketTypesInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.RegionID != 10000002 || out.Page != 1 || out.TotalPages != 7 || len(out.TypeIDs) != 3 ||
		out.TypeIDs[0] != 34 || out.TypeIDs[1] != 35 || out.TypeIDs[2] != 36 {
		t.Errorf("out = %+v, want RegionID=10000002 Page=1 TotalPages=7 TypeIDs=[34 35 36]", out)
	}
}

// --- list_structure_market_orders (private, paginated) ---

func TestListStructureMarketOrdersHandler_MapsFieldsAndDefaultsPageTo1(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	issued := time.Unix(6000, 0)
	fake := fakeStructureMarketOrdersESI{structureMarketOrders: func(_ context.Context, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		return []esiapi.MarketsStructuresStructureIdGetInner{{
			Duration: 90, IsBuyOrder: false, Issued: issued, LocationId: 1000000000001,
			MinVolume: 1, OrderId: 777, Price: 12.5, Range: "station",
			TypeId: 34, VolumeRemain: 250, VolumeTotal: 1000,
		}}, 2, nil
	}}

	handler := listStructureMarketOrdersHandler("v1", openNarrow[structureMarketOrdersESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListStructureMarketOrdersInput{CharacterID: 7, StructureID: 1000000000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.StructureID != 1000000000001 || out.Page != 1 || out.TotalPages != 2 || len(out.Orders) != 1 {
		t.Fatalf("out = %+v, want CharacterID=7 StructureID=1000000000001 Page=1 TotalPages=2 with one order", out)
	}
	o := out.Orders[0]
	if o.Duration != 90 || o.IsBuyOrder || !o.Issued.Equal(issued) || o.LocationID != 1000000000001 ||
		o.MinVolume != 1 || o.OrderID != 777 || o.Price != 12.5 || o.Range != "station" ||
		o.SystemID != 0 || o.TypeID != 34 || o.VolumeRemain != 250 || o.VolumeTotal != 1000 {
		t.Errorf("order = %+v, want Duration=90 IsBuyOrder=false Issued=%v LocationID=1000000000001 MinVolume=1 OrderID=777 Price=12.5 Range=station SystemID=0 TypeID=34 VolumeRemain=250 VolumeTotal=1000",
			o, issued)
	}
}

func TestListStructureMarketOrdersHandler_UnknownCharacter(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	handler := listStructureMarketOrdersHandler("v1", noNarrow[structureMarketOrdersESI](t), store, nil)
	_, _, err := handler(context.Background(), nil, ListStructureMarketOrdersInput{CharacterID: 999, StructureID: 1000000000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterMarketOrdersHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeCharacterMarketOrdersESI{characterMarketOrders: func(ctx context.Context) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		return nil, wantErr
	}}

	handler := listCharacterMarketOrdersHandler("v1", openNarrow[characterMarketOrdersESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterMarketOrdersInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCharacterMarketOrderHistoryHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeCharacterMarketOrderHistoryESI{characterMarketOrderHistory: func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCharacterMarketOrderHistoryHandler("v1", openNarrow[characterMarketOrderHistoryESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCharacterMarketOrderHistoryInput{CharacterID: 7})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationMarketOrdersHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeCorporationMarketOrdersESI{corporationMarketOrders: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationMarketOrdersHandler("v1", openNarrow[corporationMarketOrdersESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationMarketOrdersInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListCorporationMarketOrderHistoryHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeCorporationMarketOrderHistoryESI{corporationMarketOrderHistory: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listCorporationMarketOrderHistoryHandler("v1", openNarrow[corporationMarketOrderHistoryESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListMarketGroupsHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeMarketGroupsESI{marketGroups: func(ctx context.Context) ([]int64, error) {
		return nil, wantErr
	}}

	handler := listMarketGroupsHandler(publicNarrow[marketGroupsESI](fake))
	_, _, err := handler(context.Background(), nil, ListMarketGroupsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetMarketGroupHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeMarketGroupESI{marketGroup: func(context.Context, int64) (*esiapi.MarketsGroupsMarketGroupIdGet, error) {
		return nil, wantErr
	}}

	handler := getMarketGroupHandler(publicNarrow[marketGroupESI](fake))
	_, _, err := handler(context.Background(), nil, GetMarketGroupInput{MarketGroupID: 9})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListMarketPricesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeMarketPricesESI{marketPrices: func(_ context.Context) ([]esiapi.MarketsPricesGetInner, error) {
		return nil, wantErr
	}}

	handler := listMarketPricesHandler("v1", publicNarrow[marketPricesESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListMarketPricesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetMarketHistoryHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeMarketHistoryESI{marketHistory: func(ctx context.Context, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error) {
		return nil, wantErr
	}}

	handler := getMarketHistoryHandler(publicNarrow[marketHistoryESI](fake))
	_, _, err := handler(context.Background(), nil, GetMarketHistoryInput{RegionID: 10000002, TypeID: 34})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListMarketOrdersHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeMarketOrdersESI{marketOrders: func(_ context.Context, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listMarketOrdersHandler("v1", publicNarrow[marketOrdersESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListMarketOrdersInput{RegionID: 10000002, OrderType: "buy", TypeID: 34})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListMarketTypesHandler_PropagatesError(t *testing.T) {
	wantErr := errors.New("upstream failure")
	fake := fakeMarketTypesESI{marketTypes: func(ctx context.Context, regionID int64, page int32) ([]int64, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listMarketTypesHandler("v1", publicNarrow[marketTypesESI](fake), nil)
	_, _, err := handler(context.Background(), nil, ListMarketTypesInput{RegionID: 10000002})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListStructureMarketOrdersHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	wantErr := errors.New("upstream failure")
	fake := fakeStructureMarketOrdersESI{structureMarketOrders: func(_ context.Context, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		return nil, 0, wantErr
	}}

	handler := listStructureMarketOrdersHandler("v1", openNarrow[structureMarketOrdersESI](fake), store, nil)
	_, _, err := handler(context.Background(), nil, ListStructureMarketOrdersInput{CharacterID: 7, StructureID: 1000000000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

// --- type-name enrichment ---

func TestListCharacterMarketOrdersHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCharacterMarketOrdersESI{characterMarketOrders: func(ctx context.Context) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		return []esiapi.CharactersCharacterIdOrdersGetInner{{TypeId: 587}}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCharacterMarketOrdersHandler("v1", openNarrow[characterMarketOrdersESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMarketOrdersInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].TypeName != "Rifter" {
		t.Errorf("out.Orders = %+v, want one order with TypeName=Rifter", out.Orders)
	}
}

func TestListCharacterMarketOrderHistoryHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCharacterMarketOrderHistoryESI{characterMarketOrderHistory: func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdOrdersHistoryGetInner{{TypeId: 587}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCharacterMarketOrderHistoryHandler("v1", openNarrow[characterMarketOrderHistoryESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMarketOrderHistoryInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].TypeName != "Rifter" {
		t.Errorf("out.Orders = %+v, want one order with TypeName=Rifter", out.Orders)
	}
}

func TestListCorporationMarketOrdersHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCorporationMarketOrdersESI{corporationMarketOrders: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersGetInner{{TypeId: 587}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationMarketOrdersHandler("v1", openNarrow[corporationMarketOrdersESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMarketOrdersInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].TypeName != "Rifter" {
		t.Errorf("out.Orders = %+v, want one order with TypeName=Rifter", out.Orders)
	}
}

func TestListCorporationMarketOrderHistoryHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCorporationMarketOrderHistoryESI{corporationMarketOrderHistory: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersHistoryGetInner{{TypeId: 587}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationMarketOrderHistoryHandler("v1", openNarrow[corporationMarketOrderHistoryESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].TypeName != "Rifter" {
		t.Errorf("out.Orders = %+v, want one order with TypeName=Rifter", out.Orders)
	}
}

func TestListMarketPricesHandler_ResolvesTypeName(t *testing.T) {
	fake := fakeMarketPricesESI{marketPrices: func(_ context.Context) ([]esiapi.MarketsPricesGetInner, error) {
		return []esiapi.MarketsPricesGetInner{{TypeId: 587}}, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listMarketPricesHandler("v1", publicNarrow[marketPricesESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListMarketPricesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Prices) != 1 || out.Prices[0].TypeName != "Rifter" {
		t.Errorf("out.Prices = %+v, want one price with TypeName=Rifter", out.Prices)
	}
}

func TestListMarketOrdersHandler_ResolvesTypeName(t *testing.T) {
	fake := fakeMarketOrdersESI{marketOrders: func(_ context.Context, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		return []esiapi.MarketsRegionIdOrdersGetInner{{TypeId: 587}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listMarketOrdersHandler("v1", publicNarrow[marketOrdersESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListMarketOrdersInput{RegionID: 10000002, OrderType: "buy"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].TypeName != "Rifter" {
		t.Errorf("out.Orders = %+v, want one order with TypeName=Rifter", out.Orders)
	}
}

func TestListMarketTypesHandler_ResolvesTypeNames(t *testing.T) {
	fake := fakeMarketTypesESI{marketTypes: func(ctx context.Context, regionID int64, page int32) ([]int64, int32, error) {
		return []int64{587, 588}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 588: "Slasher"}
	}

	handler := listMarketTypesHandler("v1", publicNarrow[marketTypesESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListMarketTypesInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.TypeNames) != 2 || out.TypeNames[0] != "Rifter" || out.TypeNames[1] != "Slasher" {
		t.Errorf("out.TypeNames = %v, want [Rifter Slasher]", out.TypeNames)
	}
}

func TestListMarketTypesHandler_TypeNamesSurvivesPartialMiss(t *testing.T) {
	fake := fakeMarketTypesESI{marketTypes: func(ctx context.Context, regionID int64, page int32) ([]int64, int32, error) {
		return []int64{587, 999, 588}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		// 999 is deliberately omitted -- a cache miss on the middle ID must
		// not shift the surrounding names out of position.
		return map[int64]string{587: "Rifter", 588: "Slasher"}
	}

	handler := listMarketTypesHandler("v1", publicNarrow[marketTypesESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListMarketTypesInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	want := []string{"Rifter", "", "Slasher"}
	if len(out.TypeNames) != len(want) {
		t.Fatalf("out.TypeNames = %v, want %v", out.TypeNames, want)
	}
	for i, w := range want {
		if out.TypeNames[i] != w {
			t.Errorf("out.TypeNames[%d] = %q, want %q", i, out.TypeNames[i], w)
		}
	}
}

func TestListStructureMarketOrdersHandler_ResolvesTypeName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeStructureMarketOrdersESI{structureMarketOrders: func(_ context.Context, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		return []esiapi.MarketsStructuresStructureIdGetInner{{TypeId: 587}}, 1, nil
	}}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listStructureMarketOrdersHandler("v1", openNarrow[structureMarketOrdersESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListStructureMarketOrdersInput{CharacterID: 7, StructureID: 1000000000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].TypeName != "Rifter" {
		t.Errorf("out.Orders = %+v, want one order with TypeName=Rifter", out.Orders)
	}
}

// --- region-name / system-name enrichment ---

func TestListCharacterMarketOrdersHandler_ResolvesRegionName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCharacterMarketOrdersESI{characterMarketOrders: func(ctx context.Context) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		return []esiapi.CharactersCharacterIdOrdersGetInner{{RegionId: 10000002}}, nil
	}}

	origResolve := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origResolve })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}

	handler := listCharacterMarketOrdersHandler("v1", openNarrow[characterMarketOrdersESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMarketOrdersInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].RegionName != "The Forge" {
		t.Errorf("out.Orders = %+v, want one order with RegionName=The Forge", out.Orders)
	}
}

func TestListCharacterMarketOrderHistoryHandler_ResolvesRegionName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCharacterMarketOrderHistoryESI{characterMarketOrderHistory: func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdOrdersHistoryGetInner{{RegionId: 10000002}}, 1, nil
	}}

	origResolve := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origResolve })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}

	handler := listCharacterMarketOrderHistoryHandler("v1", openNarrow[characterMarketOrderHistoryESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCharacterMarketOrderHistoryInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].RegionName != "The Forge" {
		t.Errorf("out.Orders = %+v, want one order with RegionName=The Forge", out.Orders)
	}
}

func TestListCorporationMarketOrdersHandler_ResolvesRegionName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCorporationMarketOrdersESI{corporationMarketOrders: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersGetInner{{RegionId: 10000002}}, 1, nil
	}}

	origResolve := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origResolve })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}

	handler := listCorporationMarketOrdersHandler("v1", openNarrow[corporationMarketOrdersESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMarketOrdersInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].RegionName != "The Forge" {
		t.Errorf("out.Orders = %+v, want one order with RegionName=The Forge", out.Orders)
	}
}

func TestListCorporationMarketOrderHistoryHandler_ResolvesRegionName(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeCorporationMarketOrderHistoryESI{corporationMarketOrderHistory: func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersHistoryGetInner{{RegionId: 10000002}}, 1, nil
	}}

	origResolve := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origResolve })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}

	handler := listCorporationMarketOrderHistoryHandler("v1", openNarrow[corporationMarketOrderHistoryESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].RegionName != "The Forge" {
		t.Errorf("out.Orders = %+v, want one order with RegionName=The Forge", out.Orders)
	}
}

func TestListMarketOrdersHandler_ResolvesSystemName(t *testing.T) {
	fake := fakeMarketOrdersESI{marketOrders: func(_ context.Context, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		return []esiapi.MarketsRegionIdOrdersGetInner{{SystemId: 30000142}}, 1, nil
	}}

	origResolve := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origResolve })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listMarketOrdersHandler("v1", publicNarrow[marketOrdersESI](fake), nil)
	_, out, err := handler(context.Background(), nil, ListMarketOrdersInput{RegionID: 10000002, OrderType: "buy"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].SystemName != "Jita" {
		t.Errorf("out.Orders = %+v, want one order with SystemName=Jita", out.Orders)
	}
}

func TestListStructureMarketOrdersHandler_SystemNameEmptyWhenSystemIDZero(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	fake := fakeStructureMarketOrdersESI{structureMarketOrders: func(_ context.Context, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		return []esiapi.MarketsStructuresStructureIdGetInner{{OrderId: 1}}, 1, nil // SystemID left zero, matching the real mapper
	}}

	origResolve := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origResolve })
	calls := 0
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		calls++
		// Deliberately omits key 0 -- 0 is never a real EVE system ID, so the
		// real resolver would never have a match for it either. This proves
		// SystemName stays empty because the lookup naturally misses, not
		// because of a special-cased guard in the handler.
		return map[int64]string{}
	}

	handler := listStructureMarketOrdersHandler("v1", openNarrow[structureMarketOrdersESI](fake), store, nil)
	_, out, err := handler(context.Background(), nil, ListStructureMarketOrdersInput{CharacterID: 7, StructureID: 1})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].SystemName != "" {
		t.Errorf("out.Orders = %+v, want one order with SystemName empty (SystemID is 0 for structure orders)", out.Orders)
	}
	if calls != 1 {
		t.Errorf("resolveSystemNames calls = %d, want 1", calls)
	}
}

// fakeCharacterMarketOrdersESI fakes characterMarketOrdersESI; a nil characterMarketOrders fails as an unfaked call.
type fakeCharacterMarketOrdersESI struct {
	characterMarketOrders func(ctx context.Context) ([]esiapi.CharactersCharacterIdOrdersGetInner, error)
}

func (f fakeCharacterMarketOrdersESI) CharacterMarketOrders(ctx context.Context) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
	if f.characterMarketOrders == nil {
		return nil, errNotFaked
	}
	return f.characterMarketOrders(ctx)
}

// fakeCharacterMarketOrderHistoryESI fakes characterMarketOrderHistoryESI; a nil characterMarketOrderHistory fails as an unfaked call.
type fakeCharacterMarketOrderHistoryESI struct {
	characterMarketOrderHistory func(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error)
}

func (f fakeCharacterMarketOrderHistoryESI) CharacterMarketOrderHistory(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
	if f.characterMarketOrderHistory == nil {
		return nil, 0, errNotFaked
	}
	return f.characterMarketOrderHistory(ctx, page)
}

// fakeCorporationMarketOrdersESI fakes corporationMarketOrdersESI; a nil corporationMarketOrders fails as an unfaked call.
type fakeCorporationMarketOrdersESI struct {
	corporationMarketOrders func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error)
}

func (f fakeCorporationMarketOrdersESI) CorporationMarketOrders(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
	if f.corporationMarketOrders == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationMarketOrders(ctx, corporationID, page)
}

// fakeCorporationMarketOrderHistoryESI fakes corporationMarketOrderHistoryESI; a nil corporationMarketOrderHistory fails as an unfaked call.
type fakeCorporationMarketOrderHistoryESI struct {
	corporationMarketOrderHistory func(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error)
}

func (f fakeCorporationMarketOrderHistoryESI) CorporationMarketOrderHistory(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
	if f.corporationMarketOrderHistory == nil {
		return nil, 0, errNotFaked
	}
	return f.corporationMarketOrderHistory(ctx, corporationID, page)
}

// fakeMarketGroupsESI fakes marketGroupsESI; a nil marketGroups fails as an unfaked call.
type fakeMarketGroupsESI struct {
	marketGroups func(ctx context.Context) ([]int64, error)
}

func (f fakeMarketGroupsESI) MarketGroups(ctx context.Context) ([]int64, error) {
	if f.marketGroups == nil {
		return nil, errNotFaked
	}
	return f.marketGroups(ctx)
}

// fakeMarketGroupESI fakes marketGroupESI; a nil marketGroup fails as an unfaked call.
type fakeMarketGroupESI struct {
	marketGroup func(ctx context.Context, marketGroupID int64) (*esiapi.MarketsGroupsMarketGroupIdGet, error)
}

func (f fakeMarketGroupESI) MarketGroup(ctx context.Context, marketGroupID int64) (*esiapi.MarketsGroupsMarketGroupIdGet, error) {
	if f.marketGroup == nil {
		return nil, errNotFaked
	}
	return f.marketGroup(ctx, marketGroupID)
}

// fakeMarketHistoryESI fakes marketHistoryESI; a nil marketHistory fails as an unfaked call.
type fakeMarketHistoryESI struct {
	marketHistory func(ctx context.Context, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error)
}

func (f fakeMarketHistoryESI) MarketHistory(ctx context.Context, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error) {
	if f.marketHistory == nil {
		return nil, errNotFaked
	}
	return f.marketHistory(ctx, regionID, typeID)
}

// fakeMarketTypesESI fakes marketTypesESI; a nil marketTypes fails as an unfaked call.
type fakeMarketTypesESI struct {
	marketTypes func(ctx context.Context, regionID int64, page int32) ([]int64, int32, error)
}

func (f fakeMarketTypesESI) MarketTypes(ctx context.Context, regionID int64, page int32) ([]int64, int32, error) {
	if f.marketTypes == nil {
		return nil, 0, errNotFaked
	}
	return f.marketTypes(ctx, regionID, page)
}

// fakeMarketPricesESI fakes marketPricesESI; a nil marketPrices fails as an unfaked call.
type fakeMarketPricesESI struct {
	marketPrices func(ctx context.Context) ([]esiapi.MarketsPricesGetInner, error)
}

func (f fakeMarketPricesESI) MarketPrices(ctx context.Context) ([]esiapi.MarketsPricesGetInner, error) {
	if f.marketPrices == nil {
		return nil, errNotFaked
	}
	return f.marketPrices(ctx)
}

// fakeMarketOrdersESI fakes marketOrdersESI; a nil marketOrders fails as an unfaked call.
type fakeMarketOrdersESI struct {
	marketOrders func(ctx context.Context, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error)
}

func (f fakeMarketOrdersESI) MarketOrders(ctx context.Context, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
	if f.marketOrders == nil {
		return nil, 0, errNotFaked
	}
	return f.marketOrders(ctx, regionID, orderType, typeID, page)
}

// fakeStructureMarketOrdersESI fakes structureMarketOrdersESI; a nil structureMarketOrders fails as an unfaked call.
type fakeStructureMarketOrdersESI struct {
	structureMarketOrders func(ctx context.Context, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error)
}

func (f fakeStructureMarketOrdersESI) StructureMarketOrders(ctx context.Context, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
	if f.structureMarketOrders == nil {
		return nil, 0, errNotFaked
	}
	return f.structureMarketOrders(ctx, structureID, page)
}

// Each handler reports a client that can't be opened, wrapping the cause.
func TestStructureMarketOrdersHandlers_PropagateOpenError(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	wantErr := errors.New("no refresh token")
	for name, call := range map[string]func() error{
		"list_structure_market_orders": func() error {
			_, _, err := listStructureMarketOrdersHandler("v1", failNarrow[structureMarketOrdersESI](wantErr), store, nil)(context.Background(), nil, ListStructureMarketOrdersInput{CharacterID: 7, StructureID: 1000000000001})
			return err
		},
	} {
		if err := call(); !errors.Is(err, wantErr) {
			t.Errorf("%s: err = %v, want wrapping %v", name, err, wantErr)
		}
	}
}

// Each owner-scoped order handler reports a client that can't be opened,
// wrapping the cause.
func TestOwnerMarketOrdersHandlers_PropagateOpenError(t *testing.T) {
	store := characterStore(t, config.Character{ID: 7, Name: "Test Pilot"})
	wantErr := errors.New("no refresh token")
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"list_character_market_orders": func() error {
			_, _, err := listCharacterMarketOrdersHandler("v1", failNarrow[characterMarketOrdersESI](wantErr), store, nil)(ctx, nil, ListCharacterMarketOrdersInput{CharacterID: 7})
			return err
		},
		"list_character_market_order_history": func() error {
			_, _, err := listCharacterMarketOrderHistoryHandler("v1", failNarrow[characterMarketOrderHistoryESI](wantErr), store, nil)(ctx, nil, ListCharacterMarketOrderHistoryInput{CharacterID: 7})
			return err
		},
		"list_corporation_market_orders": func() error {
			_, _, err := listCorporationMarketOrdersHandler("v1", failNarrow[corporationMarketOrdersESI](wantErr), store, nil)(ctx, nil, ListCorporationMarketOrdersInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
		"list_corporation_market_order_history": func() error {
			_, _, err := listCorporationMarketOrderHistoryHandler("v1", failNarrow[corporationMarketOrderHistoryESI](wantErr), store, nil)(ctx, nil, ListCorporationMarketOrderHistoryInput{CharacterID: 7, CorporationID: 98000001})
			return err
		},
	} {
		if err := call(); !errors.Is(err, wantErr) {
			t.Errorf("%s: err = %v, want wrapping %v", name, err, wantErr)
		}
	}
}
