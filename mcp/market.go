package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// MarketOrder is shared by list_character_market_orders, list_character_market_order_history,
// list_corporation_market_orders, and list_corporation_market_order_history -- the four
// underlying ESI models share 13 fields exactly. IsCorporation is populated only for the two
// character-scoped variants (character open orders + character order history both carry it --
// verified from source, not just the open-orders one); State is populated only for the two
// order-history variants; IssuedBy/WalletDivision are populated only for the two
// corporation-scoped variants. Only the applicable fields are populated -- same "shared model,
// zero-value/nil signals not-applicable" precedent as mcp/industry.go's IndustryJob.
type MarketOrder struct {
	Duration       int64     `json:"duration"`
	Escrow         *float64  `json:"escrow,omitempty"`
	IsBuyOrder     *bool     `json:"isBuyOrder,omitempty"`
	Issued         time.Time `json:"issued"`
	LocationID     int64     `json:"locationId"`
	MinVolume      *int64    `json:"minVolume,omitempty"`
	OrderID        int64     `json:"orderId"`
	Price          float64   `json:"price"`
	Range          string    `json:"range"`
	RegionID       int64     `json:"regionId"`
	RegionName     string    `json:"regionName,omitempty"`
	TypeID         int64     `json:"typeId"`
	TypeName       string    `json:"typeName,omitempty"`
	VolumeRemain   int64     `json:"volumeRemain"`
	VolumeTotal    int64     `json:"volumeTotal"`
	IsCorporation  *bool     `json:"isCorporation,omitempty"`
	State          string    `json:"state,omitempty"`
	IssuedBy       *int64    `json:"issuedBy,omitempty"`
	WalletDivision *int64    `json:"walletDivision,omitempty"`
}

func toMarketOrderFromCharacterOrder(o esiapi.CharactersCharacterIdOrdersGetInner) MarketOrder {
	isCorp := o.IsCorporation
	return MarketOrder{
		Duration: o.Duration, Escrow: o.Escrow, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued,
		LocationID: o.LocationId, MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price,
		Range: o.Range, RegionID: o.RegionId, TypeID: o.TypeId, VolumeRemain: o.VolumeRemain,
		VolumeTotal: o.VolumeTotal, IsCorporation: &isCorp,
	}
}

func toMarketOrderFromCharacterOrderHistory(o esiapi.CharactersCharacterIdOrdersHistoryGetInner) MarketOrder {
	isCorp := o.IsCorporation
	return MarketOrder{
		Duration: o.Duration, Escrow: o.Escrow, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued,
		LocationID: o.LocationId, MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price,
		Range: o.Range, RegionID: o.RegionId, TypeID: o.TypeId, VolumeRemain: o.VolumeRemain,
		VolumeTotal: o.VolumeTotal, IsCorporation: &isCorp, State: o.State,
	}
}

func toMarketOrderFromCorporationOrder(o esiapi.CorporationsCorporationIdOrdersGetInner) MarketOrder {
	issuedBy := o.IssuedBy
	walletDivision := o.WalletDivision
	return MarketOrder{
		Duration: o.Duration, Escrow: o.Escrow, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued,
		LocationID: o.LocationId, MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price,
		Range: o.Range, RegionID: o.RegionId, TypeID: o.TypeId, VolumeRemain: o.VolumeRemain,
		VolumeTotal: o.VolumeTotal, IssuedBy: &issuedBy, WalletDivision: &walletDivision,
	}
}

func toMarketOrderFromCorporationOrderHistory(o esiapi.CorporationsCorporationIdOrdersHistoryGetInner) MarketOrder {
	walletDivision := o.WalletDivision
	return MarketOrder{
		Duration: o.Duration, Escrow: o.Escrow, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued,
		LocationID: o.LocationId, MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price,
		Range: o.Range, RegionID: o.RegionId, TypeID: o.TypeId, VolumeRemain: o.VolumeRemain,
		VolumeTotal: o.VolumeTotal, State: o.State, IssuedBy: o.IssuedBy, WalletDivision: &walletDivision,
	}
}

// MarketOrderBookEntry is shared by list_market_orders (region order book) and
// list_structure_market_orders (structure order book) -- the two underlying ESI models are
// identical except SystemID, present only for region orders (a structure pins its own system, so
// ESI doesn't repeat it per-order there). Named for the data shape (an anonymous order-book
// entry with no owner/issuer identity fields), not for auth: list_structure_market_orders is
// privately scoped despite sharing this shape -- see its own doc comment below.
type MarketOrderBookEntry struct {
	Duration     int64     `json:"duration"`
	IsBuyOrder   bool      `json:"isBuyOrder"`
	Issued       time.Time `json:"issued"`
	LocationID   int64     `json:"locationId"`
	MinVolume    int64     `json:"minVolume"`
	OrderID      int64     `json:"orderId"`
	Price        float64   `json:"price"`
	Range        string    `json:"range"`
	SystemID     int64     `json:"systemId,omitempty"`
	SystemName   string    `json:"systemName,omitempty"`
	TypeID       int64     `json:"typeId"`
	TypeName     string    `json:"typeName,omitempty"`
	VolumeRemain int64     `json:"volumeRemain"`
	VolumeTotal  int64     `json:"volumeTotal"`
}

func toMarketOrderBookEntryFromRegion(o esiapi.MarketsRegionIdOrdersGetInner) MarketOrderBookEntry {
	return MarketOrderBookEntry{
		Duration: o.Duration, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued, LocationID: o.LocationId,
		MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price, Range: o.Range,
		SystemID: o.SystemId, TypeID: o.TypeId, VolumeRemain: o.VolumeRemain, VolumeTotal: o.VolumeTotal,
	}
}

func toMarketOrderBookEntryFromStructure(o esiapi.MarketsStructuresStructureIdGetInner) MarketOrderBookEntry {
	return MarketOrderBookEntry{
		Duration: o.Duration, IsBuyOrder: o.IsBuyOrder, Issued: o.Issued, LocationID: o.LocationId,
		MinVolume: o.MinVolume, OrderID: o.OrderId, Price: o.Price, Range: o.Range,
		TypeID: o.TypeId, VolumeRemain: o.VolumeRemain, VolumeTotal: o.VolumeTotal,
	}
}

// --- list_character_market_orders (private) ---

var listCharacterMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]esiapi.CharactersCharacterIdOrdersGetInner, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterMarketOrders(ctx)
}

type ListCharacterMarketOrdersInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterMarketOrdersOutput struct {
	CharacterID int32         `json:"characterId"`
	Orders      []MarketOrder `json:"orders"`
}

func listCharacterMarketOrdersHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCharacterMarketOrdersInput, ListCharacterMarketOrdersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMarketOrdersInput) (*mcpsdk.CallToolResult, ListCharacterMarketOrdersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMarketOrdersOutput{}, err
		}
		if !found {
			return nil, ListCharacterMarketOrdersOutput{}, fmt.Errorf("list character market orders for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		orders, err := listCharacterMarketOrders(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterMarketOrdersOutput{}, fmt.Errorf("list character market orders: %w", err)
		}
		out := make([]MarketOrder, len(orders))
		for i, o := range orders {
			out[i] = toMarketOrderFromCharacterOrder(o)
		}
		ids := make([]int64, len(out))
		for i, o := range out {
			ids[i] = o.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		regionIDs := make([]int64, len(out))
		for i, o := range out {
			regionIDs[i] = o.RegionID
		}
		regionNames := resolveRegionNames(ctx, sdeStore, sso.UserAgent(version), regionIDs)
		for i := range out {
			out[i].RegionName = regionNames[out[i].RegionID]
		}
		return nil, ListCharacterMarketOrdersOutput{CharacterID: in.CharacterID, Orders: out}, nil
	}
}

// --- list_character_market_order_history (private, paginated) ---

var listCharacterMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]esiapi.CharactersCharacterIdOrdersHistoryGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CharacterMarketOrderHistory(ctx, page)
}

type ListCharacterMarketOrderHistoryInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterMarketOrderHistoryOutput struct {
	CharacterID int32         `json:"characterId"`
	Page        int32         `json:"page"`
	TotalPages  int32         `json:"totalPages"`
	Orders      []MarketOrder `json:"orders"`
}

func listCharacterMarketOrderHistoryHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCharacterMarketOrderHistoryInput, ListCharacterMarketOrderHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMarketOrderHistoryInput) (*mcpsdk.CallToolResult, ListCharacterMarketOrderHistoryOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMarketOrderHistoryOutput{}, err
		}
		if !found {
			return nil, ListCharacterMarketOrderHistoryOutput{}, fmt.Errorf("list character market order history for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		history, totalPages, err := listCharacterMarketOrderHistory(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Page)
		if err != nil {
			return nil, ListCharacterMarketOrderHistoryOutput{}, fmt.Errorf("list character market order history: %w", err)
		}
		out := make([]MarketOrder, len(history))
		for i, o := range history {
			out[i] = toMarketOrderFromCharacterOrderHistory(o)
		}
		ids := make([]int64, len(out))
		for i, o := range out {
			ids[i] = o.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		regionIDs := make([]int64, len(out))
		for i, o := range out {
			regionIDs[i] = o.RegionID
		}
		regionNames := resolveRegionNames(ctx, sdeStore, sso.UserAgent(version), regionIDs)
		for i := range out {
			out[i].RegionName = regionNames[out[i].RegionID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCharacterMarketOrderHistoryOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Orders: out}, nil
	}
}

// --- list_corporation_market_orders (private, corp-scoped, paginated) ---

var listCorporationMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMarketOrders(ctx, corporationID, page)
}

type ListCorporationMarketOrdersInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMarketOrdersOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Orders        []MarketOrder `json:"orders"`
}

func listCorporationMarketOrdersHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCorporationMarketOrdersInput, ListCorporationMarketOrdersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMarketOrdersInput) (*mcpsdk.CallToolResult, ListCorporationMarketOrdersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMarketOrdersOutput{}, err
		}
		if !found {
			return nil, ListCorporationMarketOrdersOutput{}, fmt.Errorf("list corporation market orders for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		orders, totalPages, err := listCorporationMarketOrders(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMarketOrdersOutput{}, fmt.Errorf("list corporation market orders: %w", err)
		}
		out := make([]MarketOrder, len(orders))
		for i, o := range orders {
			out[i] = toMarketOrderFromCorporationOrder(o)
		}
		ids := make([]int64, len(out))
		for i, o := range out {
			ids[i] = o.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		regionIDs := make([]int64, len(out))
		for i, o := range out {
			regionIDs[i] = o.RegionID
		}
		regionNames := resolveRegionNames(ctx, sdeStore, sso.UserAgent(version), regionIDs)
		for i := range out {
			out[i].RegionName = regionNames[out[i].RegionID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationMarketOrdersOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Orders: out}, nil
	}
}

// --- list_corporation_market_order_history (private, corp-scoped, paginated) ---

var listCorporationMarketOrderHistory = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdOrdersHistoryGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.CorporationMarketOrderHistory(ctx, corporationID, page)
}

type ListCorporationMarketOrderHistoryInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationMarketOrderHistoryOutput struct {
	CharacterID   int32         `json:"characterId"`
	CorporationID int64         `json:"corporationId"`
	Page          int32         `json:"page"`
	TotalPages    int32         `json:"totalPages"`
	Orders        []MarketOrder `json:"orders"`
}

func listCorporationMarketOrderHistoryHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCorporationMarketOrderHistoryInput, ListCorporationMarketOrderHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationMarketOrderHistoryInput) (*mcpsdk.CallToolResult, ListCorporationMarketOrderHistoryOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationMarketOrderHistoryOutput{}, err
		}
		if !found {
			return nil, ListCorporationMarketOrderHistoryOutput{}, fmt.Errorf("list corporation market order history for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		history, totalPages, err := listCorporationMarketOrderHistory(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationMarketOrderHistoryOutput{}, fmt.Errorf("list corporation market order history: %w", err)
		}
		out := make([]MarketOrder, len(history))
		for i, o := range history {
			out[i] = toMarketOrderFromCorporationOrderHistory(o)
		}
		ids := make([]int64, len(out))
		for i, o := range out {
			ids[i] = o.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		regionIDs := make([]int64, len(out))
		for i, o := range out {
			regionIDs[i] = o.RegionID
		}
		regionNames := resolveRegionNames(ctx, sdeStore, sso.UserAgent(version), regionIDs)
		for i := range out {
			out[i].RegionName = regionNames[out[i].RegionID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationMarketOrderHistoryOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Orders: out}, nil
	}
}

// --- list_market_groups (public, no input) ---

var listMarketGroups = func(ctx context.Context, userAgent string) ([]int64, error) {
	client := esi.NewPublicClient(userAgent)
	return client.MarketGroups(ctx)
}

type ListMarketGroupsInput struct{}
type ListMarketGroupsOutput struct {
	MarketGroupIDs []int64 `json:"marketGroupIds"`
}

func listMarketGroupsHandler(version string) mcpsdk.ToolHandlerFor[ListMarketGroupsInput, ListMarketGroupsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMarketGroupsInput) (*mcpsdk.CallToolResult, ListMarketGroupsOutput, error) {
		ids, err := listMarketGroups(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListMarketGroupsOutput{}, fmt.Errorf("list market groups: %w", err)
		}
		return nil, ListMarketGroupsOutput{MarketGroupIDs: ids}, nil
	}
}

// --- get_market_group (public) ---

type GetMarketGroupOutput struct {
	MarketGroupID int64   `json:"marketGroupId"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	ParentGroupID *int64  `json:"parentGroupId,omitempty"`
	Types         []int64 `json:"types"`
}

var getMarketGroup = func(ctx context.Context, userAgent string, marketGroupID int64) (GetMarketGroupOutput, error) {
	client := esi.NewPublicClient(userAgent)
	group, err := client.MarketGroup(ctx, marketGroupID)
	if err != nil {
		return GetMarketGroupOutput{}, err
	}
	return GetMarketGroupOutput{
		MarketGroupID: group.MarketGroupId, Name: group.Name, Description: group.Description,
		ParentGroupID: group.ParentGroupId, Types: group.Types,
	}, nil
}

type GetMarketGroupInput struct {
	MarketGroupID int64 `json:"marketGroupId" jsonschema:"the market group ID to look up"`
}

func getMarketGroupHandler(version string) mcpsdk.ToolHandlerFor[GetMarketGroupInput, GetMarketGroupOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMarketGroupInput) (*mcpsdk.CallToolResult, GetMarketGroupOutput, error) {
		out, err := getMarketGroup(ctx, sso.UserAgent(version), in.MarketGroupID)
		if err != nil {
			return nil, GetMarketGroupOutput{}, fmt.Errorf("get market group: %w", err)
		}
		return nil, out, nil
	}
}

// --- list_market_prices (public, no input) ---

type MarketPrice struct {
	TypeID        int64    `json:"typeId"`
	TypeName      string   `json:"typeName,omitempty"`
	AdjustedPrice *float64 `json:"adjustedPrice,omitempty"`
	AveragePrice  *float64 `json:"averagePrice,omitempty"`
}

var listMarketPrices = func(ctx context.Context, userAgent string) ([]esiapi.MarketsPricesGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.MarketPrices(ctx)
}

type ListMarketPricesInput struct{}
type ListMarketPricesOutput struct {
	Prices []MarketPrice `json:"prices"`
}

func listMarketPricesHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListMarketPricesInput, ListMarketPricesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMarketPricesInput) (*mcpsdk.CallToolResult, ListMarketPricesOutput, error) {
		prices, err := listMarketPrices(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, ListMarketPricesOutput{}, fmt.Errorf("list market prices: %w", err)
		}
		out := make([]MarketPrice, len(prices))
		for i, p := range prices {
			out[i] = MarketPrice{TypeID: p.TypeId, AdjustedPrice: p.AdjustedPrice, AveragePrice: p.AveragePrice}
		}
		ids := make([]int64, len(out))
		for i, p := range out {
			ids[i] = p.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		return nil, ListMarketPricesOutput{Prices: out}, nil
	}
}

// --- get_market_history (public) ---

type MarketHistoryEntry struct {
	Average    float64 `json:"average"`
	Date       string  `json:"date"`
	Highest    float64 `json:"highest"`
	Lowest     float64 `json:"lowest"`
	OrderCount int64   `json:"orderCount"`
	Volume     int64   `json:"volume"`
}

var getMarketHistory = func(ctx context.Context, userAgent string, regionID, typeID int64) ([]esiapi.MarketsRegionIdHistoryGetInner, error) {
	client := esi.NewPublicClient(userAgent)
	return client.MarketHistory(ctx, regionID, typeID)
}

type GetMarketHistoryInput struct {
	RegionID int64 `json:"regionId" jsonschema:"the region ID to look up"`
	TypeID   int64 `json:"typeId" jsonschema:"the item type ID to get historical daily stats for"`
}
type GetMarketHistoryOutput struct {
	RegionID int64                `json:"regionId"`
	TypeID   int64                `json:"typeId"`
	History  []MarketHistoryEntry `json:"history"`
}

func getMarketHistoryHandler(version string) mcpsdk.ToolHandlerFor[GetMarketHistoryInput, GetMarketHistoryOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetMarketHistoryInput) (*mcpsdk.CallToolResult, GetMarketHistoryOutput, error) {
		history, err := getMarketHistory(ctx, sso.UserAgent(version), in.RegionID, in.TypeID)
		if err != nil {
			return nil, GetMarketHistoryOutput{}, fmt.Errorf("get market history: %w", err)
		}
		out := make([]MarketHistoryEntry, len(history))
		for i, h := range history {
			out[i] = MarketHistoryEntry{Average: h.Average, Date: h.Date, Highest: h.Highest, Lowest: h.Lowest, OrderCount: h.OrderCount, Volume: h.Volume}
		}
		return nil, GetMarketHistoryOutput{RegionID: in.RegionID, TypeID: in.TypeID, History: out}, nil
	}
}

// --- list_market_orders (public, paginated) ---

var listMarketOrders = func(ctx context.Context, userAgent string, regionID int64, orderType string, typeID int64, page int32) ([]esiapi.MarketsRegionIdOrdersGetInner, int32, error) {
	client := esi.NewPublicClient(userAgent)
	return client.MarketOrders(ctx, regionID, orderType, typeID, page)
}

type ListMarketOrdersInput struct {
	RegionID  int64  `json:"regionId" jsonschema:"the region ID to look up"`
	OrderType string `json:"orderType" jsonschema:"which order types to return: buy, sell, or all"`
	TypeID    int64  `json:"typeId,omitempty" jsonschema:"optionally filter to just this item type ID"`
	Page      int32  `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListMarketOrdersOutput struct {
	RegionID   int64                  `json:"regionId"`
	OrderType  string                 `json:"orderType"`
	Page       int32                  `json:"page"`
	TotalPages int32                  `json:"totalPages"`
	Orders     []MarketOrderBookEntry `json:"orders"`
}

func listMarketOrdersHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListMarketOrdersInput, ListMarketOrdersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMarketOrdersInput) (*mcpsdk.CallToolResult, ListMarketOrdersOutput, error) {
		orders, totalPages, err := listMarketOrders(ctx, sso.UserAgent(version), in.RegionID, in.OrderType, in.TypeID, in.Page)
		if err != nil {
			return nil, ListMarketOrdersOutput{}, fmt.Errorf("list market orders: %w", err)
		}
		out := make([]MarketOrderBookEntry, len(orders))
		for i, o := range orders {
			out[i] = toMarketOrderBookEntryFromRegion(o)
		}
		ids := make([]int64, len(out))
		for i, o := range out {
			ids[i] = o.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		systemIDs := make([]int64, len(out))
		for i, o := range out {
			systemIDs[i] = o.SystemID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		for i := range out {
			out[i].SystemName = systemNames[out[i].SystemID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListMarketOrdersOutput{RegionID: in.RegionID, OrderType: in.OrderType, Page: page, TotalPages: totalPages, Orders: out}, nil
	}
}

// --- list_market_types (public, paginated) ---

var listMarketTypes = func(ctx context.Context, userAgent string, regionID int64, page int32) ([]int64, int32, error) {
	client := esi.NewPublicClient(userAgent)
	return client.MarketTypes(ctx, regionID, page)
}

type ListMarketTypesInput struct {
	RegionID int64 `json:"regionId" jsonschema:"the region ID to look up"`
	Page     int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListMarketTypesOutput struct {
	RegionID   int64    `json:"regionId"`
	Page       int32    `json:"page"`
	TotalPages int32    `json:"totalPages"`
	TypeIDs    []int64  `json:"typeIds"`
	TypeNames  []string `json:"typeNames,omitempty"`
}

func listMarketTypesHandler(version string, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListMarketTypesInput, ListMarketTypesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListMarketTypesInput) (*mcpsdk.CallToolResult, ListMarketTypesOutput, error) {
		types, totalPages, err := listMarketTypes(ctx, sso.UserAgent(version), in.RegionID, in.Page)
		if err != nil {
			return nil, ListMarketTypesOutput{}, fmt.Errorf("list market types: %w", err)
		}
		nameMap := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), types)
		typeNames := make([]string, len(types))
		for i, id := range types {
			typeNames[i] = nameMap[id]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListMarketTypesOutput{RegionID: in.RegionID, Page: page, TotalPages: totalPages, TypeIDs: types, TypeNames: typeNames}, nil
	}
}

// --- list_structure_market_orders (private, paginated) ---
//
// Despite resembling list_market_orders (a public region order book), this tool requires a
// character auth gate identical to every other private tool in this file: ESI requires OAuth2
// scope esi-markets.structure_markets.v1 for GetMarketsStructuresStructureId (verified in
// esi/api/openapi.yaml), meaning the querying character must have docking access to the
// structure. There's no separate ownerId/corporationId input -- the structure itself is the
// resource, and ESI's own 403 is the sole authorization gate, same as every corp-scoped tool.

var listStructureMarketOrders = func(ctx context.Context, clientID string, characterID int32, userAgent string, structureID int64, page int32) ([]esiapi.MarketsStructuresStructureIdGetInner, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
	return client.StructureMarketOrders(ctx, structureID, page)
}

type ListStructureMarketOrdersInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check -- must have docking access to the structure"`
	StructureID int64 `json:"structureId" jsonschema:"the structure ID to look up"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListStructureMarketOrdersOutput struct {
	CharacterID int32                  `json:"characterId"`
	StructureID int64                  `json:"structureId"`
	Page        int32                  `json:"page"`
	TotalPages  int32                  `json:"totalPages"`
	Orders      []MarketOrderBookEntry `json:"orders"`
}

func listStructureMarketOrdersHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListStructureMarketOrdersInput, ListStructureMarketOrdersOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListStructureMarketOrdersInput) (*mcpsdk.CallToolResult, ListStructureMarketOrdersOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListStructureMarketOrdersOutput{}, err
		}
		if !found {
			return nil, ListStructureMarketOrdersOutput{}, fmt.Errorf("list structure market orders for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		orders, totalPages, err := listStructureMarketOrders(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.StructureID, in.Page)
		if err != nil {
			return nil, ListStructureMarketOrdersOutput{}, fmt.Errorf("list structure market orders: %w", err)
		}
		out := make([]MarketOrderBookEntry, len(orders))
		for i, o := range orders {
			out[i] = toMarketOrderBookEntryFromStructure(o)
		}
		ids := make([]int64, len(out))
		for i, o := range out {
			ids[i] = o.TypeID
		}
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
		for i := range out {
			out[i].TypeName = names[out[i].TypeID]
		}
		systemIDs := make([]int64, len(out))
		for i, o := range out {
			systemIDs[i] = o.SystemID
		}
		systemNames := resolveSystemNames(ctx, sdeStore, sso.UserAgent(version), systemIDs)
		for i := range out {
			out[i].SystemName = systemNames[out[i].SystemID]
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListStructureMarketOrdersOutput{CharacterID: in.CharacterID, StructureID: in.StructureID, Page: page, TotalPages: totalPages, Orders: out}, nil
	}
}
