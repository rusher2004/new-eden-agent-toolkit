package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
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

// enrichAssets fills in each asset's type name and volume from the SDE.
func enrichAssets(ctx context.Context, sdeStore *sde.Store, userAgent string, assets []AssetItem) map[int64]sde.TypeBasic {
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

// listAssets is overridable in tests so no real network/auth runs.
var listAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, page int32) ([]AssetItem, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
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

func listAssetsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListAssetsInput, ListAssetsOutput] {
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
		assets, totalPages, err := listAssets(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Page)
		if err != nil {
			return nil, ListAssetsOutput{}, fmt.Errorf("list assets: %w", err)
		}
		enrichAssets(ctx, sdeStore, sso.UserAgent(version), assets)
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

// getAssetLocations is overridable in tests.
var getAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetLocation, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func getAssetLocationsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetAssetLocationsInput, GetAssetLocationsOutput] {
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
		locs, err := getAssetLocations(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ItemIDs)
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

// getAssetNames is overridable in tests.
var getAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, itemIDs []int64) ([]AssetName, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func getAssetNamesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetAssetNamesInput, GetAssetNamesOutput] {
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
		names, err := getAssetNames(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ItemIDs)
		if err != nil {
			return nil, GetAssetNamesOutput{}, fmt.Errorf("get asset names: %w", err)
		}
		return nil, GetAssetNamesOutput{CharacterID: in.CharacterID, Names: names}, nil
	}
}

// listCorporationAssets is overridable in tests.
var listCorporationAssets = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, page int32) ([]AssetItem, int32, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, 0, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCorporationAssetsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCorporationAssetsInput, ListCorporationAssetsOutput] {
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
		assets, totalPages, err := listCorporationAssets(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationAssetsOutput{}, fmt.Errorf("list corporation assets: %w", err)
		}
		enrichAssets(ctx, sdeStore, sso.UserAgent(version), assets)
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationAssetsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Assets: assets}, nil
	}
}

// getCorporationAssetLocations is overridable in tests.
var getCorporationAssetLocations = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetLocation, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func getCorporationAssetLocationsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationAssetLocationsInput, GetCorporationAssetLocationsOutput] {
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
		locs, err := getCorporationAssetLocations(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ItemIDs)
		if err != nil {
			return nil, GetCorporationAssetLocationsOutput{}, fmt.Errorf("get corporation asset locations: %w", err)
		}
		return nil, GetCorporationAssetLocationsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Locations: locs}, nil
	}
}

// getCorporationAssetNames is overridable in tests.
var getCorporationAssetNames = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64, itemIDs []int64) ([]AssetName, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func getCorporationAssetNamesHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetCorporationAssetNamesInput, GetCorporationAssetNamesOutput] {
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
		names, err := getCorporationAssetNames(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID, in.ItemIDs)
		if err != nil {
			return nil, GetCorporationAssetNamesOutput{}, fmt.Errorf("get corporation asset names: %w", err)
		}
		return nil, GetCorporationAssetNamesOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Names: names}, nil
	}
}
