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

	orig := listCharacterMarketOrders
	t.Cleanup(func() { listCharacterMarketOrders = orig })
	issued := time.Unix(1000, 0)
	escrow := 100.5
	isBuyOrder := true
	minVolume := int64(1)
	listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		return []esiapi.CharactersCharacterIdOrdersGetInner{{
			Duration: 90, Escrow: &escrow, IsBuyOrder: &isBuyOrder, IsCorporation: false,
			Issued: issued, LocationId: 60003760, MinVolume: &minVolume, OrderId: 123456,
			Price: 5.55, Range: "region", RegionId: 10000002, TypeId: 34,
			VolumeRemain: 500, VolumeTotal: 1000,
		}}, nil
	}

	handler := listCharacterMarketOrdersHandler("v1", "client-id", store, nil)
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

	orig := listCharacterMarketOrders
	t.Cleanup(func() { listCharacterMarketOrders = orig })
	listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		t.Fatal("listCharacterMarketOrders should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterMarketOrdersHandler("v1", "client-id", store, nil)
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

	orig := listCharacterMarketOrderHistory
	t.Cleanup(func() { listCharacterMarketOrderHistory = orig })
	issued := time.Unix(2000, 0)
	listCharacterMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdOrdersHistoryGetInner{{
			Duration: 90, IsCorporation: true, Issued: issued, LocationId: 60003760,
			OrderId: 654321, Price: 5.55, Range: "region", RegionId: 10000002,
			State: "expired", TypeId: 34, VolumeRemain: 0, VolumeTotal: 1000,
		}}, 3, nil
	}

	handler := listCharacterMarketOrderHistoryHandler("v1", "client-id", store, nil)
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

	orig := listCharacterMarketOrderHistory
	t.Cleanup(func() { listCharacterMarketOrderHistory = orig })
	listCharacterMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		t.Fatal("listCharacterMarketOrderHistory should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCharacterMarketOrderHistoryHandler("v1", "client-id", store, nil)
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

	orig := listCorporationMarketOrders
	t.Cleanup(func() { listCorporationMarketOrders = orig })
	issued := time.Unix(3000, 0)
	listCorporationMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersGetInner{{
			Duration: 90, Issued: issued, IssuedBy: 998877, LocationId: 60003760,
			OrderId: 111222, Price: 5.55, Range: "region", RegionId: 10000002,
			TypeId: 34, VolumeRemain: 500, VolumeTotal: 1000, WalletDivision: 1,
		}}, 2, nil
	}

	handler := listCorporationMarketOrdersHandler("v1", "client-id", store, nil)
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

	orig := listCorporationMarketOrders
	t.Cleanup(func() { listCorporationMarketOrders = orig })
	listCorporationMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		t.Fatal("listCorporationMarketOrders should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationMarketOrdersHandler("v1", "client-id", store, nil)
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

	orig := listCorporationMarketOrderHistory
	t.Cleanup(func() { listCorporationMarketOrderHistory = orig })
	issued := time.Unix(4000, 0)
	listCorporationMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersHistoryGetInner{{
			Duration: 90, Issued: issued, IssuedBy: nil, LocationId: 60003760,
			OrderId: 333444, Price: 5.55, Range: "region", RegionId: 10000002,
			State: "cancelled", TypeId: 34, VolumeRemain: 0, VolumeTotal: 1000, WalletDivision: 2,
		}}, 1, nil
	}

	handler := listCorporationMarketOrderHistoryHandler("v1", "client-id", store, nil)
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

	orig := listCorporationMarketOrderHistory
	t.Cleanup(func() { listCorporationMarketOrderHistory = orig })
	listCorporationMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		t.Fatal("listCorporationMarketOrderHistory should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listCorporationMarketOrderHistoryHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

// --- list_market_groups (public) ---

func TestListMarketGroupsHandler_MapsFields(t *testing.T) {
	orig := listMarketGroups
	t.Cleanup(func() { listMarketGroups = orig })
	listMarketGroups = func(ctx context.Context, userAgent string) ([]int64, error) {
		return []int64{1, 2, 3}, nil
	}

	handler := listMarketGroupsHandler("v1")
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
	orig := getMarketGroup
	t.Cleanup(func() { getMarketGroup = orig })
	parentGroupID := int64(1)
	getMarketGroup = func(ctx context.Context, userAgent string, marketGroupID int64) (GetMarketGroupOutput, error) {
		return GetMarketGroupOutput{
			MarketGroupID: marketGroupID, Name: "Ammo & Charges", Description: "Ammo & Charges",
			ParentGroupID: &parentGroupID, Types: []int64{10, 11},
		}, nil
	}

	handler := getMarketGroupHandler("v1")
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
	orig := listMarketPrices
	t.Cleanup(func() { listMarketPrices = orig })
	adjustedPrice := 306.11
	averagePrice := 306.29
	listMarketPrices = func(ctx context.Context, userAgent string) ([]esiapi.MarketsPricesGetInner, error) {
		return []esiapi.MarketsPricesGetInner{{
			AdjustedPrice: &adjustedPrice, AveragePrice: &averagePrice, TypeId: 32772,
		}}, nil
	}

	handler := listMarketPricesHandler("v1", nil)
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
	orig := getMarketHistory
	t.Cleanup(func() { getMarketHistory = orig })
	getMarketHistory = func(ctx context.Context, userAgent string, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error) {
		return []esiapi.MarketsRegionIdHistoryGetInner{{
			Average: 5.5, Date: "2026-07-20", Highest: 6.0, Lowest: 5.0, OrderCount: 10, Volume: 1000,
		}}, nil
	}

	handler := getMarketHistoryHandler("v1")
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
	orig := listMarketOrders
	t.Cleanup(func() { listMarketOrders = orig })
	issued := time.Unix(5000, 0)
	var gotRegionID, gotTypeID int64
	var gotOrderType string
	var gotPage int32
	listMarketOrders = func(ctx context.Context, userAgent string, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		gotRegionID, gotOrderType, gotTypeID, gotPage = regionID, orderType, typeID, page
		return []esiapi.MarketsRegionIdOrdersGetInner{{
			Duration: 90, IsBuyOrder: true, Issued: issued, LocationId: 60003760,
			MinVolume: 1, OrderId: 555, Price: 5.55, Range: "region", SystemId: 30000142,
			TypeId: 34, VolumeRemain: 500, VolumeTotal: 1000,
		}}, 5, nil
	}

	handler := listMarketOrdersHandler("v1", nil)
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
	orig := listMarketTypes
	t.Cleanup(func() { listMarketTypes = orig })
	listMarketTypes = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]int64, int32, error) {
		return []int64{34, 35, 36}, 7, nil
	}

	handler := listMarketTypesHandler("v1", nil)
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

	orig := listStructureMarketOrders
	t.Cleanup(func() { listStructureMarketOrders = orig })
	issued := time.Unix(6000, 0)
	listStructureMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		return []esiapi.MarketsStructuresStructureIdGetInner{{
			Duration: 90, IsBuyOrder: false, Issued: issued, LocationId: 1000000000001,
			MinVolume: 1, OrderId: 777, Price: 12.5, Range: "station",
			TypeId: 34, VolumeRemain: 250, VolumeTotal: 1000,
		}}, 2, nil
	}

	handler := listStructureMarketOrdersHandler("v1", "client-id", store, nil)
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

	orig := listStructureMarketOrders
	t.Cleanup(func() { listStructureMarketOrders = orig })
	listStructureMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		t.Fatal("listStructureMarketOrders should not be called for an unknown character")
		return nil, 0, nil
	}

	handler := listStructureMarketOrdersHandler("v1", "client-id", store, nil)
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

	orig := listCharacterMarketOrders
	t.Cleanup(func() { listCharacterMarketOrders = orig })
	wantErr := errors.New("upstream failure")
	listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		return nil, wantErr
	}

	handler := listCharacterMarketOrdersHandler("v1", "client-id", store, nil)
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

	orig := listCharacterMarketOrderHistory
	t.Cleanup(func() { listCharacterMarketOrderHistory = orig })
	wantErr := errors.New("upstream failure")
	listCharacterMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCharacterMarketOrderHistoryHandler("v1", "client-id", store, nil)
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

	orig := listCorporationMarketOrders
	t.Cleanup(func() { listCorporationMarketOrders = orig })
	wantErr := errors.New("upstream failure")
	listCorporationMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationMarketOrdersHandler("v1", "client-id", store, nil)
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

	orig := listCorporationMarketOrderHistory
	t.Cleanup(func() { listCorporationMarketOrderHistory = orig })
	wantErr := errors.New("upstream failure")
	listCorporationMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listCorporationMarketOrderHistoryHandler("v1", "client-id", store, nil)
	_, _, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListMarketGroupsHandler_PropagatesError(t *testing.T) {
	orig := listMarketGroups
	t.Cleanup(func() { listMarketGroups = orig })
	wantErr := errors.New("upstream failure")
	listMarketGroups = func(ctx context.Context, userAgent string) ([]int64, error) {
		return nil, wantErr
	}

	handler := listMarketGroupsHandler("v1")
	_, _, err := handler(context.Background(), nil, ListMarketGroupsInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetMarketGroupHandler_PropagatesError(t *testing.T) {
	orig := getMarketGroup
	t.Cleanup(func() { getMarketGroup = orig })
	wantErr := errors.New("upstream failure")
	getMarketGroup = func(ctx context.Context, userAgent string, marketGroupID int64) (GetMarketGroupOutput, error) {
		return GetMarketGroupOutput{}, wantErr
	}

	handler := getMarketGroupHandler("v1")
	_, _, err := handler(context.Background(), nil, GetMarketGroupInput{MarketGroupID: 9})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListMarketPricesHandler_PropagatesError(t *testing.T) {
	orig := listMarketPrices
	t.Cleanup(func() { listMarketPrices = orig })
	wantErr := errors.New("upstream failure")
	listMarketPrices = func(ctx context.Context, userAgent string) ([]esiapi.MarketsPricesGetInner, error) {
		return nil, wantErr
	}

	handler := listMarketPricesHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListMarketPricesInput{})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestGetMarketHistoryHandler_PropagatesError(t *testing.T) {
	orig := getMarketHistory
	t.Cleanup(func() { getMarketHistory = orig })
	wantErr := errors.New("upstream failure")
	getMarketHistory = func(ctx context.Context, userAgent string, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error) {
		return nil, wantErr
	}

	handler := getMarketHistoryHandler("v1")
	_, _, err := handler(context.Background(), nil, GetMarketHistoryInput{RegionID: 10000002, TypeID: 34})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListMarketOrdersHandler_PropagatesError(t *testing.T) {
	orig := listMarketOrders
	t.Cleanup(func() { listMarketOrders = orig })
	wantErr := errors.New("upstream failure")
	listMarketOrders = func(ctx context.Context, userAgent string, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listMarketOrdersHandler("v1", nil)
	_, _, err := handler(context.Background(), nil, ListMarketOrdersInput{RegionID: 10000002, OrderType: "buy", TypeID: 34})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}

func TestListMarketTypesHandler_PropagatesError(t *testing.T) {
	orig := listMarketTypes
	t.Cleanup(func() { listMarketTypes = orig })
	wantErr := errors.New("upstream failure")
	listMarketTypes = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]int64, int32, error) {
		return nil, 0, wantErr
	}

	handler := listMarketTypesHandler("v1", nil)
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

	orig := listStructureMarketOrders
	t.Cleanup(func() { listStructureMarketOrders = orig })
	wantErr := errors.New("upstream failure")
	listStructureMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		return nil, 0, wantErr
	}

	handler := listStructureMarketOrdersHandler("v1", "client-id", store, nil)
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

	origFetch := listCharacterMarketOrders
	t.Cleanup(func() { listCharacterMarketOrders = origFetch })
	listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		return []esiapi.CharactersCharacterIdOrdersGetInner{{TypeId: 587}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCharacterMarketOrdersHandler("v1", "client-id", store, nil)
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

	origFetch := listCharacterMarketOrderHistory
	t.Cleanup(func() { listCharacterMarketOrderHistory = origFetch })
	listCharacterMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdOrdersHistoryGetInner{{TypeId: 587}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCharacterMarketOrderHistoryHandler("v1", "client-id", store, nil)
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

	origFetch := listCorporationMarketOrders
	t.Cleanup(func() { listCorporationMarketOrders = origFetch })
	listCorporationMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersGetInner{{TypeId: 587}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationMarketOrdersHandler("v1", "client-id", store, nil)
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

	origFetch := listCorporationMarketOrderHistory
	t.Cleanup(func() { listCorporationMarketOrderHistory = origFetch })
	listCorporationMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersHistoryGetInner{{TypeId: 587}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listCorporationMarketOrderHistoryHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].TypeName != "Rifter" {
		t.Errorf("out.Orders = %+v, want one order with TypeName=Rifter", out.Orders)
	}
}

func TestListMarketPricesHandler_ResolvesTypeName(t *testing.T) {
	origFetch := listMarketPrices
	t.Cleanup(func() { listMarketPrices = origFetch })
	listMarketPrices = func(ctx context.Context, userAgent string) ([]esiapi.MarketsPricesGetInner, error) {
		return []esiapi.MarketsPricesGetInner{{TypeId: 587}}, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listMarketPricesHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListMarketPricesInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Prices) != 1 || out.Prices[0].TypeName != "Rifter" {
		t.Errorf("out.Prices = %+v, want one price with TypeName=Rifter", out.Prices)
	}
}

func TestListMarketOrdersHandler_ResolvesTypeName(t *testing.T) {
	origFetch := listMarketOrders
	t.Cleanup(func() { listMarketOrders = origFetch })
	listMarketOrders = func(ctx context.Context, userAgent string, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		return []esiapi.MarketsRegionIdOrdersGetInner{{TypeId: 587}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listMarketOrdersHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListMarketOrdersInput{RegionID: 10000002, OrderType: "buy"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].TypeName != "Rifter" {
		t.Errorf("out.Orders = %+v, want one order with TypeName=Rifter", out.Orders)
	}
}

func TestListMarketTypesHandler_ResolvesTypeNames(t *testing.T) {
	origFetch := listMarketTypes
	t.Cleanup(func() { listMarketTypes = origFetch })
	listMarketTypes = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]int64, int32, error) {
		return []int64{587, 588}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter", 588: "Slasher"}
	}

	handler := listMarketTypesHandler("v1", nil)
	_, out, err := handler(context.Background(), nil, ListMarketTypesInput{RegionID: 10000002})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.TypeNames) != 2 || out.TypeNames[0] != "Rifter" || out.TypeNames[1] != "Slasher" {
		t.Errorf("out.TypeNames = %v, want [Rifter Slasher]", out.TypeNames)
	}
}

func TestListMarketTypesHandler_TypeNamesSurvivesPartialMiss(t *testing.T) {
	origFetch := listMarketTypes
	t.Cleanup(func() { listMarketTypes = origFetch })
	listMarketTypes = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]int64, int32, error) {
		return []int64{587, 999, 588}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		// 999 is deliberately omitted -- a cache miss on the middle ID must
		// not shift the surrounding names out of position.
		return map[int64]string{587: "Rifter", 588: "Slasher"}
	}

	handler := listMarketTypesHandler("v1", nil)
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

	origFetch := listStructureMarketOrders
	t.Cleanup(func() { listStructureMarketOrders = origFetch })
	listStructureMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		return []esiapi.MarketsStructuresStructureIdGetInner{{TypeId: 587}}, 1, nil
	}

	origResolve := resolveTypeNames
	t.Cleanup(func() { resolveTypeNames = origResolve })
	resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{587: "Rifter"}
	}

	handler := listStructureMarketOrdersHandler("v1", "client-id", store, nil)
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

	origFetch := listCharacterMarketOrders
	t.Cleanup(func() { listCharacterMarketOrders = origFetch })
	listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
		return []esiapi.CharactersCharacterIdOrdersGetInner{{RegionId: 10000002}}, nil
	}

	origResolve := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origResolve })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}

	handler := listCharacterMarketOrdersHandler("v1", "client-id", store, nil)
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

	origFetch := listCharacterMarketOrderHistory
	t.Cleanup(func() { listCharacterMarketOrderHistory = origFetch })
	listCharacterMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CharactersCharacterIdOrdersHistoryGetInner{{RegionId: 10000002}}, 1, nil
	}

	origResolve := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origResolve })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}

	handler := listCharacterMarketOrderHistoryHandler("v1", "client-id", store, nil)
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

	origFetch := listCorporationMarketOrders
	t.Cleanup(func() { listCorporationMarketOrders = origFetch })
	listCorporationMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersGetInner{{RegionId: 10000002}}, 1, nil
	}

	origResolve := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origResolve })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}

	handler := listCorporationMarketOrdersHandler("v1", "client-id", store, nil)
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

	origFetch := listCorporationMarketOrderHistory
	t.Cleanup(func() { listCorporationMarketOrderHistory = origFetch })
	listCorporationMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
		return []esiapi.CorporationsCorporationIdOrdersHistoryGetInner{{RegionId: 10000002}}, 1, nil
	}

	origResolve := resolveRegionNames
	t.Cleanup(func() { resolveRegionNames = origResolve })
	resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{10000002: "The Forge"}
	}

	handler := listCorporationMarketOrderHistoryHandler("v1", "client-id", store, nil)
	_, out, err := handler(context.Background(), nil, ListCorporationMarketOrderHistoryInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Orders) != 1 || out.Orders[0].RegionName != "The Forge" {
		t.Errorf("out.Orders = %+v, want one order with RegionName=The Forge", out.Orders)
	}
}

func TestListMarketOrdersHandler_ResolvesSystemName(t *testing.T) {
	origFetch := listMarketOrders
	t.Cleanup(func() { listMarketOrders = origFetch })
	listMarketOrders = func(ctx context.Context, userAgent string, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		return []esiapi.MarketsRegionIdOrdersGetInner{{SystemId: 30000142}}, 1, nil
	}

	origResolve := resolveSystemNames
	t.Cleanup(func() { resolveSystemNames = origResolve })
	resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
		return map[int64]string{30000142: "Jita"}
	}

	handler := listMarketOrdersHandler("v1", nil)
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

	origFetch := listStructureMarketOrders
	t.Cleanup(func() { listStructureMarketOrders = origFetch })
	listStructureMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		return []esiapi.MarketsStructuresStructureIdGetInner{{OrderId: 1}}, 1, nil // SystemID left zero, matching the real mapper
	}

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

	handler := listStructureMarketOrdersHandler("v1", "client-id", store, nil)
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
