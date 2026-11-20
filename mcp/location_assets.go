package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/cargo"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// maxAssetPages caps how many pages of assets one owner's listing walks, so
// a very large corporation can't turn one call into hundreds of requests.
const maxAssetPages = 50

// ListLocationAssetsInput is list_location_assets' input.
type ListLocationAssetsInput struct {
	LocationID             int64   `json:"locationId" jsonschema:"the NPC station or Upwell structure ID"`
	CharacterIDs           []int32 `json:"characterIds,omitempty" jsonschema:"whose assets to include; omit for every logged-in character"`
	CorporationID          int64   `json:"corporationId,omitempty" jsonschema:"also include this corporation's assets there"`
	CorporationCharacterID int32   `json:"corporationCharacterId,omitempty" jsonschema:"the logged-in character whose corporation roles read corporationId's assets; required with corporationId"`
}

// AssetSource is one owner whose assets were read.
type AssetSource struct {
	Owner string `json:"owner"`
	Items int    `json:"items" jsonschema:"how many of the owner's items are at the location"`
}

// ListLocationAssetsOutput is list_location_assets' result.
type ListLocationAssetsOutput struct {
	LocationID   int64                 `json:"locationId"`
	LocationName string                `json:"locationName,omitempty"`
	Items        []cargo.InventoryLine `json:"items"`
	Totals       cargo.InventoryTotals `json:"totals"`
	Sources      []AssetSource         `json:"sources"`
	Warnings     []string              `json:"warnings,omitempty"`
}

// allAssetPages walks every page of an owner's assets.
func allAssetPages(fetch func(page int32) ([]AssetItem, int32, error)) ([]AssetItem, error) {
	var all []AssetItem
	for page := int32(1); ; page++ {
		items, total, err := fetch(page)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if page >= total || page >= maxAssetPages {
			return all, nil
		}
	}
}

func listLocationAssetsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListLocationAssetsInput, ListLocationAssetsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListLocationAssetsInput) (*mcpsdk.CallToolResult, ListLocationAssetsOutput, error) {
		if in.LocationID == 0 {
			return nil, ListLocationAssetsOutput{}, fmt.Errorf("list location assets: locationId is required")
		}
		if (in.CorporationID == 0) != (in.CorporationCharacterID == 0) {
			return nil, ListLocationAssetsOutput{}, fmt.Errorf("list location assets: corporationId and corporationCharacterId go together")
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, ListLocationAssetsOutput{}, fmt.Errorf("load config: %w", err)
		}
		chars := in.CharacterIDs
		if len(chars) == 0 {
			for _, c := range cfg.Characters {
				chars = append(chars, c.ID)
			}
		}
		for _, id := range append(append([]int32(nil), chars...), in.CorporationCharacterID) {
			if _, ok := cfg.Character(id); id != 0 && !ok {
				return nil, ListLocationAssetsOutput{}, fmt.Errorf("list location assets for character %d: %w", id, ErrCharacterNotFound)
			}
		}
		ua := sso.UserAgent(version)

		type owned struct {
			owner  string
			assets []AssetItem
		}
		var owners []owned
		var warnings []string
		for _, id := range chars {
			c, _ := cfg.Character(id)
			assets, err := allAssetPages(func(page int32) ([]AssetItem, int32, error) {
				return listAssets(ctx, clientID, id, ua, page)
			})
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("couldn't read %s's assets: %v", c.Name, err))
				continue
			}
			owners = append(owners, owned{c.Name, assets})
		}
		if in.CorporationID != 0 {
			assets, err := allAssetPages(func(page int32) ([]AssetItem, int32, error) {
				return listCorporationAssets(ctx, clientID, in.CorporationCharacterID, ua, in.CorporationID, page)
			})
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("couldn't read corporation %d's assets: %v", in.CorporationID, err))
			} else {
				owners = append(owners, owned{fmt.Sprintf("corporation %d", in.CorporationID), assets})
			}
		}
		if len(owners) == 0 {
			return nil, ListLocationAssetsOutput{}, fmt.Errorf("list location assets: no assets could be read: %v", warnings)
		}

		var items []cargo.Item
		out := ListLocationAssetsOutput{LocationID: in.LocationID, Warnings: warnings}
		for _, o := range owners {
			basics := enrichAssets(ctx, sdeStore, ua, o.assets)
			before := len(items)
			for _, a := range o.assets {
				items = append(items, cargoItem(a, basics[a.TypeID], o.owner))
			}
			lines, _ := cargo.Inventory(items[before:], in.LocationID)
			out.Sources = append(out.Sources, AssetSource{Owner: o.owner, Items: len(lines)})
		}
		out.Items, out.Totals = cargo.Inventory(items, in.LocationID)
		if out.Items == nil {
			out.Items = []cargo.InventoryLine{}
		}
		if names := resolveStationNames(ctx, sdeStore, ua, []int64{in.LocationID}); names != nil {
			out.LocationName = names[in.LocationID]
		}
		return nil, out, nil
	}
}
