package mcp

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// AssetItem is one asset entry, shared by character- and corporation-scoped listings.
type AssetItem struct {
	ItemID          int64  `json:"itemId"`
	TypeID          int64  `json:"typeId"`
	TypeName        string `json:"typeName,omitempty"`
	LocationID      int64  `json:"locationId"`
	LocationType    string `json:"locationType"`
	LocationFlag    string `json:"locationFlag"`
	Quantity        int64  `json:"quantity"`
	IsSingleton     bool   `json:"isSingleton"`
	IsBlueprintCopy *bool  `json:"isBlueprintCopy,omitempty"`
	// VolumeM3 is the stack's volume as it is now, from the SDE: assembled
	// for an assembled item (isSingleton), packaged otherwise.
	VolumeM3 float64 `json:"volumeM3,omitempty" jsonschema:"the whole stack's volume in m³ as it is now: assembled for an assembled item (isSingleton true), packaged otherwise"`
}

// cargoItem pairs an asset with its type's SDE details for cargo/.
func cargoItem(a AssetItem, b sde.TypeBasic, owner string) cargo.Item {
	assembled := b.AssembledVolume
	if assembled == 0 {
		assembled = b.Volume
	}
	return cargo.Item{
		ItemID: a.ItemID, TypeID: a.TypeID, Name: a.TypeName, GroupID: b.GroupID, CategoryID: b.CategoryID,
		LocationID: a.LocationID, Flag: a.LocationFlag, Quantity: a.Quantity, Singleton: a.IsSingleton,
		Volume: assembled, PackagedVolume: b.Volume, Owner: owner,
	}
}

// assetEnrichCache is the SDE lookups enrichAssets makes: each type's name and
// volume.
type assetEnrichCache interface {
	typeNameCache
	typeBasicsCache
}

// enrichAssets fills in each asset's type name and volume from the SDE.
func enrichAssets(ctx context.Context, sdeStore assetEnrichCache, userAgent string, assets []AssetItem) map[int64]sde.TypeBasic {
	ids := make([]int64, len(assets))
	for i, a := range assets {
		ids[i] = a.TypeID
	}
	names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
	basics := resolveTypeBasics(ctx, sdeStore, userAgent, ids)
	for i := range assets {
		assets[i].TypeName = names[assets[i].TypeID]
		if b, ok := basics[assets[i].TypeID]; ok {
			assets[i].VolumeM3 = cargoItem(assets[i], b, "").StackVolume()
		}
	}
	return basics
}

// assetsESI is the ESI call list_assets makes; the planner family reads a
// character's assets through it too.
type assetsESI interface {
	Assets(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdAssetsGetInner, int32, error)
}

// listAssets reads one page of a character's assets.
func listAssets(ctx context.Context, client assetsESI, page int32) ([]AssetItem, int32, error) {
	assets, totalPages, err := client.Assets(ctx, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]AssetItem, len(assets))
	for i, a := range assets {
		out[i] = AssetItem{
			ItemID: a.ItemId, TypeID: a.TypeId, LocationID: a.LocationId,
			LocationType: a.LocationType, LocationFlag: a.LocationFlag,
			Quantity: a.Quantity, IsSingleton: a.IsSingleton, IsBlueprintCopy: a.IsBlueprintCopy,
		}
	}
	return out, totalPages, nil
}

// ListAssetsInput is list_assets's tool input.
type ListAssetsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}

// ListAssetsOutput is list_assets's tool result.
type ListAssetsOutput struct {
	CharacterID int32       `json:"characterId"`
	Page        int32       `json:"page"`
	TotalPages  int32       `json:"totalPages"`
	Assets      []AssetItem `json:"assets"`
}

// AddListAssets registers list_assets. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListAssets[C assetsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore assetEnrichCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_assets",
		Description: "List a logged-in EVE Online character's assets.",
	}, listAssetsHandler(userAgent, open, store, sdeStore))
}

func listAssetsHandler[C assetsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore assetEnrichCache) mcpsdk.ToolHandlerFor[ListAssetsInput, ListAssetsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAssetsInput) (*mcpsdk.CallToolResult, ListAssetsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListAssetsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListAssetsOutput{}, fmt.Errorf("list assets for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListAssetsOutput{}, fmt.Errorf("list assets: %w", err)
		}
		assets, totalPages, err := listAssets(ctx, client, in.Page)
		if err != nil {
			return nil, ListAssetsOutput{}, fmt.Errorf("list assets: %w", err)
		}
		enrichAssets(ctx, sdeStore, userAgent, assets)
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListAssetsOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Assets: assets}, nil
	}
}

// AssetLocation is one resolved item's position.
type AssetLocation struct {
	ItemID int64   `json:"itemId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
}

// assetLocationsESI is the ESI call get_asset_locations makes.
type assetLocationsESI interface {
	AssetLocations(ctx context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, error)
}

// getAssetLocations reads the positions of a character's items.
func getAssetLocations(ctx context.Context, client assetLocationsESI, itemIDs []int64) ([]AssetLocation, error) {
	locs, err := client.AssetLocations(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	out := make([]AssetLocation, len(locs))
	for i, l := range locs {
		out[i] = AssetLocation{ItemID: l.ItemId, X: l.Position.X, Y: l.Position.Y, Z: l.Position.Z}
	}
	return out, nil
}

// GetAssetLocationsInput is get_asset_locations's tool input.
type GetAssetLocationsInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	ItemIDs     []int64 `json:"itemIds" jsonschema:"item IDs to resolve, from list_assets"`
}

// GetAssetLocationsOutput is get_asset_locations's tool result.
type GetAssetLocationsOutput struct {
	CharacterID int32           `json:"characterId"`
	Locations   []AssetLocation `json:"locations"`
}

// AddGetAssetLocations registers get_asset_locations. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetAssetLocations[C assetLocationsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_asset_locations",
		Description: "Resolve a set of a character's asset item IDs into their positions.",
	}, getAssetLocationsHandler(open, store))
}

func getAssetLocationsHandler[C assetLocationsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetAssetLocationsInput, GetAssetLocationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetAssetLocationsInput) (*mcpsdk.CallToolResult, GetAssetLocationsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetAssetLocationsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetAssetLocationsOutput{}, fmt.Errorf("get asset locations for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetAssetLocationsOutput{}, fmt.Errorf("get asset locations: %w", err)
		}
		locs, err := getAssetLocations(ctx, client, in.ItemIDs)
		if err != nil {
			return nil, GetAssetLocationsOutput{}, fmt.Errorf("get asset locations: %w", err)
		}
		return nil, GetAssetLocationsOutput{CharacterID: in.CharacterID, Locations: locs}, nil
	}
}

// AssetName is one item's player-assigned custom name.
type AssetName struct {
	ItemID int64  `json:"itemId"`
	Name   string `json:"name"`
}

// assetNamesESI is the ESI call get_asset_names makes; the planner family
// names a character's ships and containers through it too.
type assetNamesESI interface {
	AssetNames(ctx context.Context, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error)
}

// getAssetNames reads the custom names of a character's items.
func getAssetNames(ctx context.Context, client assetNamesESI, itemIDs []int64) ([]AssetName, error) {
	names, err := client.AssetNames(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	out := make([]AssetName, len(names))
	for i, n := range names {
		out[i] = AssetName{ItemID: n.ItemId, Name: n.Name}
	}
	return out, nil
}

// GetAssetNamesInput is get_asset_names's tool input.
type GetAssetNamesInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	ItemIDs     []int64 `json:"itemIds" jsonschema:"item IDs to resolve, from list_assets"`
}

// GetAssetNamesOutput is get_asset_names's tool result.
type GetAssetNamesOutput struct {
	CharacterID int32       `json:"characterId"`
	Names       []AssetName `json:"names"`
}

// AddGetAssetNames registers get_asset_names. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetAssetNames[C assetNamesESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_asset_names",
		Description: "Resolve a set of a character's asset item IDs into their player-assigned custom names.",
	}, getAssetNamesHandler(open, store))
}

func getAssetNamesHandler[C assetNamesESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetAssetNamesInput, GetAssetNamesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetAssetNamesInput) (*mcpsdk.CallToolResult, GetAssetNamesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetAssetNamesOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetAssetNamesOutput{}, fmt.Errorf("get asset names for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetAssetNamesOutput{}, fmt.Errorf("get asset names: %w", err)
		}
		names, err := getAssetNames(ctx, client, in.ItemIDs)
		if err != nil {
			return nil, GetAssetNamesOutput{}, fmt.Errorf("get asset names: %w", err)
		}
		return nil, GetAssetNamesOutput{CharacterID: in.CharacterID, Names: names}, nil
	}
}

// corporationAssetsESI is the ESI call list_corporation_assets makes; the
// planner family reads a corporation's assets through it too.
type corporationAssetsESI interface {
	CorporationAssets(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdAssetsGetInner, int32, error)
}

// listCorporationAssets reads one page of a corporation's assets.
func listCorporationAssets(ctx context.Context, client corporationAssetsESI, corporationID int64, page int32) ([]AssetItem, int32, error) {
	assets, totalPages, err := client.CorporationAssets(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]AssetItem, len(assets))
	for i, a := range assets {
		out[i] = AssetItem{
			ItemID: a.ItemId, TypeID: a.TypeId, LocationID: a.LocationId,
			LocationType: a.LocationType, LocationFlag: a.LocationFlag,
			Quantity: a.Quantity, IsSingleton: a.IsSingleton, IsBlueprintCopy: a.IsBlueprintCopy,
		}
	}
	return out, totalPages, nil
}

// ListCorporationAssetsInput is list_corporation_assets's tool input.
type ListCorporationAssetsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}

// ListCorporationAssetsOutput is list_corporation_assets's tool result.
type ListCorporationAssetsOutput struct {
	CharacterID   int32       `json:"characterId"`
	CorporationID int64       `json:"corporationId"`
	Page          int32       `json:"page"`
	TotalPages    int32       `json:"totalPages"`
	Assets        []AssetItem `json:"assets"`
}

// AddListCorporationAssets registers list_corporation_assets. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCorporationAssets[C corporationAssetsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore assetEnrichCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_assets",
		Description: "List a corporation's assets, using a logged-in character's roles to authorize.",
	}, listCorporationAssetsHandler(userAgent, open, store, sdeStore))
}

func listCorporationAssetsHandler[C corporationAssetsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore assetEnrichCache) mcpsdk.ToolHandlerFor[ListCorporationAssetsInput, ListCorporationAssetsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationAssetsInput) (*mcpsdk.CallToolResult, ListCorporationAssetsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListCorporationAssetsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListCorporationAssetsOutput{}, fmt.Errorf("list corporation assets for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationAssetsOutput{}, fmt.Errorf("list corporation assets: %w", err)
		}
		assets, totalPages, err := listCorporationAssets(ctx, client, in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationAssetsOutput{}, fmt.Errorf("list corporation assets: %w", err)
		}
		enrichAssets(ctx, sdeStore, userAgent, assets)
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationAssetsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Assets: assets}, nil
	}
}

// corporationAssetLocationsESI is the ESI call get_corporation_asset_locations makes.
type corporationAssetLocationsESI interface {
	CorporationAssetLocations(ctx context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsLocationsPostInner, error)
}

// getCorporationAssetLocations reads the positions of a corporation's items.
func getCorporationAssetLocations(ctx context.Context, client corporationAssetLocationsESI, corporationID int64, itemIDs []int64) ([]AssetLocation, error) {
	locs, err := client.CorporationAssetLocations(ctx, corporationID, itemIDs)
	if err != nil {
		return nil, err
	}
	out := make([]AssetLocation, len(locs))
	for i, l := range locs {
		out[i] = AssetLocation{ItemID: l.ItemId, X: l.Position.X, Y: l.Position.Y, Z: l.Position.Z}
	}
	return out, nil
}

// GetCorporationAssetLocationsInput is get_corporation_asset_locations's tool input.
type GetCorporationAssetLocationsInput struct {
	CharacterID   int32   `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64   `json:"corporationId" jsonschema:"the corporation ID to look up"`
	ItemIDs       []int64 `json:"itemIds" jsonschema:"item IDs to resolve, from list_corporation_assets"`
}

// GetCorporationAssetLocationsOutput is get_corporation_asset_locations's tool result.
type GetCorporationAssetLocationsOutput struct {
	CharacterID   int32           `json:"characterId"`
	CorporationID int64           `json:"corporationId"`
	Locations     []AssetLocation `json:"locations"`
}

// AddGetCorporationAssetLocations registers get_corporation_asset_locations. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCorporationAssetLocations[C corporationAssetLocationsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_corporation_asset_locations",
		Description: "Resolve a set of a corporation's asset item IDs into their positions.",
	}, getCorporationAssetLocationsHandler(open, store))
}

func getCorporationAssetLocationsHandler[C corporationAssetLocationsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCorporationAssetLocationsInput, GetCorporationAssetLocationsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationAssetLocationsInput) (*mcpsdk.CallToolResult, GetCorporationAssetLocationsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetCorporationAssetLocationsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetCorporationAssetLocationsOutput{}, fmt.Errorf("get corporation asset locations for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCorporationAssetLocationsOutput{}, fmt.Errorf("get corporation asset locations: %w", err)
		}
		locs, err := getCorporationAssetLocations(ctx, client, in.CorporationID, in.ItemIDs)
		if err != nil {
			return nil, GetCorporationAssetLocationsOutput{}, fmt.Errorf("get corporation asset locations: %w", err)
		}
		return nil, GetCorporationAssetLocationsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Locations: locs}, nil
	}
}

// corporationAssetNamesESI is the ESI call get_corporation_asset_names
// makes; the planner family names a corporation's containers through it too.
type corporationAssetNamesESI interface {
	CorporationAssetNames(ctx context.Context, corporationID int64, itemIDs []int64) ([]esiapi.CharactersCharacterIdAssetsNamesPostInner, error)
}

// getCorporationAssetNames reads the custom names of a corporation's items.
func getCorporationAssetNames(ctx context.Context, client corporationAssetNamesESI, corporationID int64, itemIDs []int64) ([]AssetName, error) {
	names, err := client.CorporationAssetNames(ctx, corporationID, itemIDs)
	if err != nil {
		return nil, err
	}
	out := make([]AssetName, len(names))
	for i, n := range names {
		out[i] = AssetName{ItemID: n.ItemId, Name: n.Name}
	}
	return out, nil
}

// GetCorporationAssetNamesInput is get_corporation_asset_names's tool input.
type GetCorporationAssetNamesInput struct {
	CharacterID   int32   `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64   `json:"corporationId" jsonschema:"the corporation ID to look up"`
	ItemIDs       []int64 `json:"itemIds" jsonschema:"item IDs to resolve, from list_corporation_assets"`
}

// GetCorporationAssetNamesOutput is get_corporation_asset_names's tool result.
type GetCorporationAssetNamesOutput struct {
	CharacterID   int32       `json:"characterId"`
	CorporationID int64       `json:"corporationId"`
	Names         []AssetName `json:"names"`
}

// AddGetCorporationAssetNames registers get_corporation_asset_names. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddGetCorporationAssetNames[C corporationAssetNamesESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_corporation_asset_names",
		Description: "Resolve a set of a corporation's asset item IDs into their player-assigned custom names.",
	}, getCorporationAssetNamesHandler(open, store))
}

func getCorporationAssetNamesHandler[C corporationAssetNamesESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCorporationAssetNamesInput, GetCorporationAssetNamesOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCorporationAssetNamesInput) (*mcpsdk.CallToolResult, GetCorporationAssetNamesOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetCorporationAssetNamesOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetCorporationAssetNamesOutput{}, fmt.Errorf("get corporation asset names for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCorporationAssetNamesOutput{}, fmt.Errorf("get corporation asset names: %w", err)
		}
		names, err := getCorporationAssetNames(ctx, client, in.CorporationID, in.ItemIDs)
		if err != nil {
			return nil, GetCorporationAssetNamesOutput{}, fmt.Errorf("get corporation asset names: %w", err)
		}
		return nil, GetCorporationAssetNamesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Names: names}, nil
	}
}
