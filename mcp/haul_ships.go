package mcp

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/nav"
)

// cynoClass is the class list_haul_ships gives a ship that's in none of the
// hull classes but has a cyno generator fitted.
const cynoClass = "cyno ship"

// maxAssetNameIDs is how many item IDs one ESI asset names request takes.
const maxAssetNameIDs = 1000

// corpOfficeFlag is the location flag ESI gives a corporation's office in a
// station or structure; corporation hangar items sit inside it.
const corpOfficeFlag = "OfficeFolder"

// ListHaulShipsInput is list_haul_ships' input.
type ListHaulShipsInput struct {
	CharacterIDs           []int32  `json:"characterIds,omitempty" jsonschema:"whose ships to list; omit for every logged-in character"`
	Classes                []string `json:"classes,omitempty" jsonschema:"ship classes to list, such as Jump Freighter, Carrier or 'cyno ship' (a ship of any other class with a cyno generator fitted); omit for all of them"`
	CorporationID          int64    `json:"corporationId,omitempty" jsonschema:"also list this corporation's ships"`
	CorporationCharacterID int32    `json:"corporationCharacterId,omitempty" jsonschema:"the logged-in character whose corporation roles read corporationId's assets; required with corporationId"`
}

// HaulShip is one assembled ship and where it is.
type HaulShip struct {
	OwnerID         int32    `json:"ownerId,omitempty" jsonschema:"the character who owns it; pass it as ownerId, with itemId as shipItemId, to estimate_ship_hold or estimate_cyno, or as shipOwnerId to save_haul_profile. Absent for a corporation's ship"`
	Owner           string   `json:"owner"`
	ItemID          int64    `json:"itemId"`
	Type            string   `json:"type"`
	Class           string   `json:"class"`
	Name            string   `json:"name,omitempty"`
	LocationID      int64    `json:"locationId" jsonschema:"the station, structure or solar system it's in"`
	Location        string   `json:"location"`
	System          string   `json:"system,omitempty" jsonschema:"absent for a structure that isn't saved with save_structure"`
	SecurityDisplay string   `json:"securityDisplay,omitempty"`
	SecurityColor   string   `json:"securityColor,omitempty"`
	InsideItemID    int64    `json:"insideItemId,omitempty" jsonschema:"the ship or container it's carried in, such as a carrier's ship maintenance bay"`
	Inside          string   `json:"inside,omitempty"`
	CynoGenerator   string   `json:"cynoGenerator,omitempty" jsonschema:"the cyno generator fitted to it"`
	Modules         []string `json:"modules,omitempty" jsonschema:"a corporation ship's fitted modules, to pass with its type to estimate_ship_hold, since only a character's own ships can be read by itemId"`
}

// ListHaulShipsOutput is list_haul_ships' result.
type ListHaulShipsOutput struct {
	Ships    []HaulShip `json:"ships"`
	Warnings []string   `json:"warnings,omitempty"`
}

// haulShipOwner is one owner's assets, read for list_haul_ships.
type haulShipOwner struct {
	id     int32 // the character, or 0 for a corporation
	name   string
	assets []AssetItem
}

// rootOf follows an item up through the items holding it to the station,
// structure or solar system they're all in. It returns that location's ID
// and kind and the item directly holding this one, if any. A corporation
// office is transparent: a ship in a corporation hangar has no holder.
func rootOf(byID map[int64]AssetItem, a AssetItem) (loc int64, kind string, holder int64) {
	cur := a
	for seen := 0; seen <= len(byID); seen++ {
		parent, ok := byID[cur.LocationID]
		if !ok {
			return cur.LocationID, cur.LocationType, holder
		}
		if holder == 0 && parent.LocationFlag != corpOfficeFlag {
			holder = parent.ItemID
		}
		cur = parent
	}
	return cur.LocationID, cur.LocationType, holder
}

// AddListHaulShips registers list_haul_ships. openESI and openPublicESI build the ESI clients;
// any client with the methods the planner and this tool call will do.
func AddListHaulShips[C plannerClient, P plannerPublicClient](s *mcpsdk.Server, userAgent string, openESI func(ctx context.Context, characterID int32) (C, error), openPublicESI func() P, store plannerConfig, sdeStore plannerSDE) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_haul_ships",
		Description: "Find where the user's hauling, jump-capable and capital ships are: every assembled hauler, jump freighter, freighter, transport, blockade runner, industrial command ship, Rorqual, Black Ops, carrier, dreadnought, force auxiliary, supercarrier and titan the logged-in characters own (and a corporation's, through a character with the roles), plus any other ship with a cyno generator fitted. Each comes with its owner, item ID, class, name, the station, structure or system it's in with its displayed security, the ship it's carried in if any, and its fitted cyno generator. Use it instead of reading full asset lists to find haulers and cyno ships; pass its itemId as shipItemId and its ownerId as ownerId to estimate_ship_hold or estimate_cyno, or as shipOwnerId to save_haul_profile.",
	}, listHaulShipsHandler(userAgent, openESI, openPublicESI, store, sdeStore))
}

func listHaulShipsHandler[C plannerClient, P plannerPublicClient](userAgent string, openESI func(ctx context.Context, characterID int32) (C, error), openPublicESI func() P, store plannerConfig, sdeStore plannerSDE) mcpsdk.ToolHandlerFor[ListHaulShipsInput, ListHaulShipsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListHaulShipsInput) (*mcpsdk.CallToolResult, ListHaulShipsOutput, error) {
		if (in.CorporationID == 0) != (in.CorporationCharacterID == 0) {
			return nil, ListHaulShipsOutput{}, fmt.Errorf("list haul ships: corporationId and corporationCharacterId go together")
		}
		cfg, err := store.Load()
		if err != nil {
			return nil, ListHaulShipsOutput{}, fmt.Errorf("load config: %w", err)
		}
		chars, err := targetCharacters(cfg, in.CharacterIDs)
		if err != nil {
			return nil, ListHaulShipsOutput{}, err
		}
		if in.CorporationCharacterID != 0 {
			if _, ok := cfg.Character(in.CorporationCharacterID); !ok {
				return nil, ListHaulShipsOutput{}, fmt.Errorf("list haul ships for character %d: %w", in.CorporationCharacterID, ErrCharacterNotFound)
			}
		}
		p := newPlanner(userAgent, plannerOpener(openESI), plannerPublicOpener(openPublicESI), store, sdeStore)

		classTypes, err := sdeStore.PublishedTypesInGroups(ctx, p.ua(), hullClassGroups)
		if err != nil {
			return nil, ListHaulShipsOutput{}, fmt.Errorf("hull classes: %w", err)
		}
		className := map[int64]string{}
		for _, t := range classTypes {
			className[t.GroupID] = t.GroupName
		}
		want := map[string]bool{}
		for _, c := range in.Classes {
			c = strings.ToLower(strings.TrimSpace(c))
			known := c == cynoClass
			for _, n := range className {
				known = known || strings.ToLower(n) == c
			}
			if !known {
				var names []string
				for _, g := range hullClassGroups {
					if n := className[g]; n != "" {
						names = append(names, n)
					}
				}
				return nil, ListHaulShipsOutput{}, fmt.Errorf("list haul ships: unknown class %q; use one of %s or %q", c, strings.Join(names, ", "), cynoClass)
			}
			want[c] = true
		}
		generators := map[int64]bool{}
		for _, g := range cynoGenerators {
			generators[g] = true
		}

		var owners []haulShipOwner
		var warnings []string
		for _, c := range chars {
			client, err := p.esi(ctx, c.ID)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("couldn't read %s's assets: %v", c.Name, err))
				continue
			}
			assets, truncated, err := allAssetPages(func(page int32) ([]AssetItem, int32, error) { return listAssets(ctx, client, page) })
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("couldn't read %s's assets: %v", c.Name, err))
				continue
			}
			if truncated {
				warnings = append(warnings, truncatedWarning(c.Name))
			}
			owners = append(owners, haulShipOwner{c.ID, c.Name, assets})
		}
		if in.CorporationID != 0 {
			var assets []AssetItem
			var truncated bool
			client, err := p.esi(ctx, in.CorporationCharacterID)
			if err == nil {
				assets, truncated, err = allAssetPages(func(page int32) ([]AssetItem, int32, error) {
					return listCorporationAssets(ctx, client, in.CorporationID, page)
				})
			}
			corp := fmt.Sprintf("corporation %d", in.CorporationID)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("couldn't read %s's assets: %v", corp, err))
			} else {
				if truncated {
					warnings = append(warnings, truncatedWarning(corp))
				}
				owners = append(owners, haulShipOwner{0, corp, assets})
			}
		}
		if len(owners) == 0 {
			return nil, ListHaulShipsOutput{}, fmt.Errorf("list haul ships: no assets could be read: %v", warnings)
		}

		out := ListHaulShipsOutput{Ships: []HaulShip{}, Warnings: warnings}
		for _, o := range owners {
			ships, warning, err := p.haulShips(ctx, cfg, o, className, generators, want)
			if err != nil {
				return nil, ListHaulShipsOutput{}, err
			}
			if warning != "" {
				out.Warnings = append(out.Warnings, warning)
			}
			out.Ships = append(out.Ships, ships...)
		}
		return nil, out, nil
	}
}

// haulShips picks one owner's hauling, jump-capable, capital and cyno ships
// out of their assets and says where each is. warning is set when the names
// the user gave their ships couldn't be read.
func (p *planner) haulShips(ctx context.Context, cfg config.Config, o haulShipOwner, className map[int64]string, generators map[int64]bool, want map[string]bool) (ships []HaulShip, warning string, err error) {
	byID := make(map[int64]AssetItem, len(o.assets))
	ids := make([]int64, 0, len(o.assets))
	for _, a := range o.assets {
		byID[a.ItemID] = a
		ids = append(ids, a.TypeID)
	}
	basics, err := p.typeBasics(ctx, ids)
	if err != nil {
		return nil, "", err
	}
	fitted := map[int64][]int64{} // ship item ID -> fitted module types
	for _, a := range o.assets {
		if isFittedFlag(a.LocationFlag) && basics[a.TypeID].CategoryID != chargeCategoryID {
			fitted[a.LocationID] = append(fitted[a.LocationID], a.TypeID)
		}
	}

	for _, a := range o.assets {
		b := basics[a.TypeID]
		if !a.IsSingleton || b.CategoryID != shipCategoryID {
			continue
		}
		s := HaulShip{OwnerID: o.id, Owner: o.name, ItemID: a.ItemID, Type: b.Name, Class: className[b.GroupID]}
		for _, m := range fitted[a.ItemID] {
			if generators[m] {
				s.CynoGenerator = p.typeName(ctx, m)
			}
		}
		if s.Class == "" && s.CynoGenerator != "" {
			s.Class = cynoClass
		}
		if s.Class == "" || (len(want) > 0 && !want[strings.ToLower(s.Class)]) {
			continue
		}
		if o.id == 0 {
			for _, m := range fitted[a.ItemID] {
				s.Modules = append(s.Modules, p.typeName(ctx, m))
			}
		}
		loc, kind, holder := rootOf(byID, a)
		s.LocationID, s.InsideItemID = loc, holder
		if holder != 0 {
			s.Inside = basics[byID[holder].TypeID].Name
		}
		var system int64
		if kind == "solar_system" {
			system = loc
		} else {
			s.Location = p.placeName(ctx, cfg, loc)
			if st, ok := cfg.Structure(loc); ok && st.System != nil {
				system = st.System.ID
			} else if station, ok, err := p.sde.Station(ctx, p.ua(), loc); err == nil && ok {
				system = station.SystemID
			}
		}
		if system != 0 {
			if sys, err := p.system(ctx, system); err == nil {
				s.System = sys.Name
				s.SecurityDisplay, s.SecurityColor = nav.SecurityLabel(sys.Security), nav.SecurityColor(sys.Security)
				if kind == "solar_system" {
					s.Location = sys.Name + " (in space)"
				}
			}
		}
		if s.Location == "" {
			s.Location = fmt.Sprintf("solar system %d (in space)", loc)
		}
		ships = append(ships, s)
	}

	// Ships the user has named, read for characters only: a corporation's
	// asset names need roles this tool doesn't ask for.
	if o.id != 0 && len(ships) > 0 {
		names := map[int64]string{}
		client, err := p.esi(ctx, o.id)
		if err != nil {
			warning = fmt.Sprintf("couldn't read the names of %s's ships, so they're left out: %v", o.name, err)
		}
		for chunk := range slices.Chunk(ships, maxAssetNameIDs) {
			if err != nil {
				break // no client to read names with
			}
			var shipIDs []int64
			for _, s := range chunk {
				shipIDs = append(shipIDs, s.ItemID)
			}
			got, err := getAssetNames(ctx, client, shipIDs)
			if err != nil {
				warning = fmt.Sprintf("couldn't read the names of %s's ships, so they're left out: %v", o.name, err)
				continue
			}
			for _, n := range got {
				names[n.ItemID] = n.Name
			}
		}
		for i := range ships {
			ships[i].Name = names[ships[i].ItemID]
		}
	}

	order := map[string]int{}
	for i, g := range hullClassGroups {
		order[className[g]] = i
	}
	order[cynoClass] = len(hullClassGroups)
	sort.SliceStable(ships, func(i, j int) bool {
		if order[ships[i].Class] != order[ships[j].Class] {
			return order[ships[i].Class] < order[ships[j].Class]
		}
		if ships[i].Type != ships[j].Type {
			return ships[i].Type < ships[j].Type
		}
		return ships[i].ItemID < ships[j].ItemID
	})
	return ships, warning, nil
}
