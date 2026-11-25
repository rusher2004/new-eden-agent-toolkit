package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/market"
)

const (
	jitaHub     = int64(60003760)
	amarrHub    = int64(60008494)
	amamakeHub  = int64(1022167642188)
	forgeRegion = int64(10000002)
)

// stubHubMarkets fakes ESI market and contract reads. regionOrders holds
// every order by region; structure orders come from structOrders and need
// docking[character].
func stubHubMarkets(t *testing.T, regionOrders map[int64][]market.BookOrder, structOrders []market.BookOrder, docking map[int32]bool) {
	t.Helper()
	origM, origS, origC, origI, origT, origN, origF := listMarketOrders, listStructureMarketOrders, listPublicContracts, listPublicContractItems, today, now, fetchSellOrders
	t.Cleanup(func() {
		listMarketOrders, listStructureMarketOrders, listPublicContracts, listPublicContractItems, today, now, fetchSellOrders = origM, origS, origC, origI, origT, origN, origF
	})
	today = func() string { return "2026-11-20" }
	now = func() time.Time { return time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC) }
	listMarketOrders = func(_ context.Context, _ string, regionID int64, orderType string, typeID int64, _ int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
		var out []esiapi.MarketsRegionIdOrdersGetInner
		for _, o := range regionOrders[regionID] {
			if o.TypeID == typeID && (orderType == "all" || (orderType == "sell") == !o.IsBuy) {
				out = append(out, esiapi.MarketsRegionIdOrdersGetInner{TypeId: o.TypeID, Price: o.Price, VolumeRemain: o.Volume, LocationId: o.LocationID, IsBuyOrder: o.IsBuy})
			}
		}
		return out, 1, nil
	}
	fetchSellOrders = origF
	listStructureMarketOrders = func(_ context.Context, _ string, characterID int32, _ string, _ int64, _ int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
		if !docking[characterID] {
			return nil, 0, errors.New("403 Forbidden")
		}
		var out []esiapi.MarketsStructuresStructureIdGetInner
		for _, o := range structOrders {
			out = append(out, esiapi.MarketsStructuresStructureIdGetInner{TypeId: o.TypeID, Price: o.Price, VolumeRemain: o.Volume, LocationId: o.LocationID, IsBuyOrder: o.IsBuy})
		}
		return out, 1, nil
	}
	price := func(p float64) *float64 { return &p }
	loc := func(l int64) *int64 { return &l }
	listPublicContracts = func(_ context.Context, _ string, _ int64, _ int32) ([]Contract, int32, error) {
		return []Contract{
			{ContractID: 1, Type: "item_exchange", Price: price(12e9), StartLocationID: loc(jitaHub)},
			{ContractID: 2, Type: "item_exchange", Price: price(2e9), StartLocationID: loc(jitaHub)},
			{ContractID: 3, Type: "courier", Price: price(50e9), StartLocationID: loc(jitaHub)},
		}, 1, nil
	}
	listPublicContractItems = func(_ context.Context, _ string, contractID int64, _ int32) ([]ContractItem, int32, error) {
		if contractID == 1 {
			return []ContractItem{{TypeID: 28846, Quantity: 1, IsIncluded: true}}, 1, nil
		}
		return []ContractItem{{TypeID: 34, Quantity: 5, IsIncluded: true}}, 1, nil
	}
}

func TestListHubProfiles_BuiltinsAndStaleness(t *testing.T) {
	store, _ := industryTestEnv(t)
	stubHubMarkets(t, nil, nil, nil)
	_, out, err := listHubProfilesHandler(store)(context.Background(), nil, struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Hubs) != len(config.BuiltinHubs) || out.Hubs[0].StationID != jitaHub || !out.Hubs[0].ValidationStale {
		t.Errorf("hubs = %+v", out.Hubs)
	}
}

func TestValidateTradeHub_NPCBasketAndContracts(t *testing.T) {
	store, p := industryTestEnv(t)
	stubHubMarkets(t, map[int64][]market.BookOrder{forgeRegion: {
		{TypeID: 34, Price: 4, Volume: 1000, LocationID: jitaHub},
		{TypeID: 34, Price: 3.9, Volume: 500, LocationID: jitaHub, IsBuy: true},
		{TypeID: 35, Price: 9, Volume: 10, LocationID: 1}, // elsewhere in The Forge
	}}, nil, nil)
	_, out, err := validateTradeHubHandler("test", "client", store, p.sde)(context.Background(), nil, ValidateTradeHubInput{
		Hub: "Jita", Basket: []ItemInput{{Name: "Tritanium"}, {TypeID: 35}}, Contracts: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	v := out.Hub.Validation
	if v == nil || v.Date != "2026-11-20" || v.Basket.Stocked != 1 || v.Basket.Of != 2 || v.Basket.SellOrders != 1 || v.Basket.BuyOrders != 1 || len(v.Basket.Missing) != 1 {
		t.Fatalf("validation = %+v basket %+v", v, v.Basket)
	}
	if c := v.Contracts; c == nil || c.Public != 2 || c.Over1B != 2 || c.Over10B != 1 || len(c.ShipGroups) != 1 {
		t.Errorf("contracts = %+v", c)
	}
	if len(out.Changes) == 0 || out.Changes[0].Entry != "60003760" {
		t.Errorf("changes = %+v", out.Changes)
	}
	_, list, _ := listHubProfilesHandler(store)(context.Background(), nil, struct{}{})
	if list.Hubs[0].ValidationStale {
		t.Error("a hub validated today is marked stale")
	}
}

func TestValidateTradeHub_StructureNeedsACharacterAndRemembersIt(t *testing.T) {
	store, p := industryTestEnv(t)
	stubHubMarkets(t, nil, []market.BookOrder{
		{TypeID: 34, Price: 5, Volume: 10, LocationID: amamakeHub},
		{TypeID: 35, Price: 5, Volume: 10, LocationID: amamakeHub, IsBuy: true},
	}, map[int32]bool{testPilot: true})
	validate := validateTradeHubHandler("test", "client", store, p.sde)
	if _, _, err := validate(context.Background(), nil, ValidateTradeHubInput{Hub: "Amamake"}); err == nil || !strings.Contains(err.Error(), "docking access") {
		t.Errorf("a structure hub with no character: err = %v", err)
	}
	_, out, err := validate(context.Background(), nil, ValidateTradeHubInput{Hub: "Amamake", CharacterID: testPilot, Basket: []ItemInput{{TypeID: 34}, {TypeID: 35}}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Hub.ReadVia == nil || out.Hub.ReadVia.ID != int64(testPilot) || out.Hub.Validation.Basket.Stocked != 1 || out.Hub.Validation.Basket.BuyOrders != 1 {
		t.Errorf("hub = %+v validation %+v", out.Hub, out.Hub.Validation.Basket)
	}

	// The remembered character prices purchases there from now on.
	_, quote, err := quoteMarketPurchaseHandler("test", "client", store, p.sde)(context.Background(), nil, QuoteMarketPurchaseInput{Hub: "Amamake", Items: []ItemInput{{TypeID: 34, Quantity: 5}}})
	if err != nil || len(quote.Lines) != 1 || quote.Lines[0].Total != 25 {
		t.Errorf("quote = %+v, %v", quote, err)
	}
}

func TestCompareTradeHubs_RanksByCoverageThenPriceAndSkipsUnreadable(t *testing.T) {
	store, p := industryTestEnv(t)
	stubHubMarkets(t, map[int64][]market.BookOrder{
		forgeRegion: {{TypeID: 34, Price: 4, Volume: 1000, LocationID: jitaHub}, {TypeID: 35, Price: 10, Volume: 1000, LocationID: jitaHub}},
		10000043:    {{TypeID: 34, Price: 3, Volume: 1000, LocationID: amarrHub}},
	}, nil, nil)
	_, out, err := compareTradeHubsHandler("test", "client", store, p.sde)(context.Background(), nil, CompareTradeHubsInput{
		Items: []ItemInput{{TypeID: 34, Quantity: 100}, {TypeID: 35, Quantity: 10}}, Hubs: []string{"Amarr", "Jita", "Amamake"}, From: "Jita",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Hubs) != 2 || out.Hubs[0].HubID != jitaHub || out.Hubs[0].Covered != 2 || out.Hubs[0].Total != 500 || out.Hubs[0].Jumps != 0 || out.Hubs[1].HubID != amarrHub {
		t.Errorf("hubs = %+v", out.Hubs)
	}
	if out.Hubs[0].SecurityDisplay != "0.9" {
		t.Errorf("Jita security = %q", out.Hubs[0].SecurityDisplay)
	}
	if len(out.Warnings) != 1 || !strings.Contains(out.Warnings[0], "Amamake") {
		t.Errorf("warnings = %v, want Amamake skipped for lack of a character", out.Warnings)
	}
}

func TestSaveAndDeleteHubProfile(t *testing.T) {
	store, p := industryTestEnv(t)
	stubHubMarkets(t, nil, nil, nil)
	_, out, err := saveHubProfileHandler("test", "client", store, p.sde)(context.Background(), nil, SaveHubProfileInput{ID: jitaHub, Name: "Jita 4-4", Notes: new("my default")})
	if err != nil {
		t.Fatal(err)
	}
	if out.Hub.Name != "Jita 4-4" || out.Hub.Notes != "my default" || out.Hub.Region.ID != forgeRegion || len(out.Changes) != 2 {
		t.Errorf("out = %+v changes %+v", out.Hub, out.Changes)
	}
	_, del, err := deleteHubProfileHandler(store)(context.Background(), nil, DeleteHubProfileInput{Hub: "Amarr"})
	if err != nil || !strings.HasPrefix(del.Deleted, "Amarr VIII") {
		t.Errorf("delete = %+v, %v", del, err)
	}
	cfg, _ := store.Load()
	if _, ok := cfg.HubProfile("Amarr"); ok {
		t.Error("Amarr came back after deleting it")
	}
}
