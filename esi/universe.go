// Package esi's universe.go covers all 30 of "Universe"'s endpoints -- static reference data
// (ancestries, bloodlines, races, factions, categories/groups/types), geography (regions,
// constellations, systems), celestial detail (asteroid belts, moons, planets, stargates, stars,
// stations), graphics, player structures, system activity stats, and bulk ID/name resolution.
// Every endpoint is public EXCEPT UniverseStructureDetail (esi-universe.read_structures.v1) --
// verified individually per endpoint from esi/api/openapi.yaml, not assumed from tag similarity.
// Only GetUniverseGroups and GetUniverseTypes are paginated (verified individually: no .Page()
// setter on any other of the 30 request builders).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

func (c *Client) UniverseAncestries(ctx context.Context) ([]esiapi.UniverseAncestriesGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseAncestries(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe ancestries: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseBloodlines(ctx context.Context) ([]esiapi.UniverseBloodlinesGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseBloodlines(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe bloodlines: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseRaces(ctx context.Context) ([]esiapi.UniverseRacesGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseRaces(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe races: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseFactions(ctx context.Context) ([]esiapi.UniverseFactionsGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseFactions(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe factions: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseCategories(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseCategories(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe categories: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseCategoryDetail(ctx context.Context, categoryID int64) (*esiapi.UniverseCategoriesCategoryIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseCategoriesCategoryId(ctx, categoryID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe category detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseGroups(ctx context.Context, page int32) ([]int64, int32, error) {
	req := c.api.UniverseAPI.GetUniverseGroups(ctx)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get universe groups: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

func (c *Client) UniverseGroupDetail(ctx context.Context, groupID int64) (*esiapi.UniverseGroupsGroupIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseGroupsGroupId(ctx, groupID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe group detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseTypes(ctx context.Context, page int32) ([]int64, int32, error) {
	req := c.api.UniverseAPI.GetUniverseTypes(ctx)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get universe types: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

func (c *Client) UniverseTypeDetail(ctx context.Context, typeID int64) (*esiapi.UniverseTypesTypeIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseTypesTypeId(ctx, typeID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe type detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseRegions(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseRegions(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe regions: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseRegionDetail(ctx context.Context, regionID int64) (*esiapi.UniverseRegionsRegionIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseRegionsRegionId(ctx, regionID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe region detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseConstellations(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseConstellations(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe constellations: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseConstellationDetail(ctx context.Context, constellationID int64) (*esiapi.UniverseConstellationsConstellationIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseConstellationsConstellationId(ctx, constellationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe constellation detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseSystems(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseSystems(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe systems: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseSystemDetail(ctx context.Context, systemID int64) (*esiapi.UniverseSystemsSystemIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseSystemsSystemId(ctx, systemID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe system detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseAsteroidBeltDetail(ctx context.Context, asteroidBeltID int64) (*esiapi.UniverseAsteroidBeltsAsteroidBeltIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseAsteroidBeltsAsteroidBeltId(ctx, asteroidBeltID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe asteroid belt detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseMoonDetail(ctx context.Context, moonID int64) (*esiapi.UniverseMoonsMoonIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseMoonsMoonId(ctx, moonID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe moon detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniversePlanetDetail(ctx context.Context, planetID int64) (*esiapi.UniversePlanetsPlanetIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniversePlanetsPlanetId(ctx, planetID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe planet detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseStargateDetail(ctx context.Context, stargateID int64) (*esiapi.UniverseStargatesStargateIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseStargatesStargateId(ctx, stargateID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe stargate detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseStarDetail(ctx context.Context, starID int64) (*esiapi.UniverseStarsStarIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseStarsStarId(ctx, starID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe star detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseStationDetail(ctx context.Context, stationID int64) (*esiapi.UniverseStationsStationIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseStationsStationId(ctx, stationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe station detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseGraphics(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseGraphics(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe graphics: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseGraphicDetail(ctx context.Context, graphicID int64) (*esiapi.UniverseGraphicsGraphicIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseGraphicsGraphicId(ctx, graphicID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe graphic detail: %w", describeError(err, resp))
	}
	return result, nil
}

// UniverseStructures returns public structure IDs, optionally narrowed to those with a given
// service online. filter must be "" (no filter), "market", or "manufacturing_basic" -- ESI's own
// accepted values (verified in esi/api/openapi.yaml); the MCP layer validates this before
// calling.
func (c *Client) UniverseStructures(ctx context.Context, filter string) ([]int64, error) {
	req := c.api.UniverseAPI.GetUniverseStructures(ctx)
	if filter != "" {
		req = req.Filter(filter)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe structures: %w", describeError(err, resp))
	}
	return result, nil
}

// UniverseStructureDetail is the only private method in this file: it must be called on a
// *Client built via esi.NewClient (per-character authenticated), not esi.NewPublicClient. The
// authenticated character just needs docking access to structureID -- ESI's own 403 is the sole
// authorization gate, same as esi/market.go's StructureMarketOrders.
func (c *Client) UniverseStructureDetail(ctx context.Context, structureID int64) (*esiapi.UniverseStructuresStructureIdGet, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseStructuresStructureId(ctx, structureID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe structure detail: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseSystemJumps(ctx context.Context) ([]esiapi.UniverseSystemJumpsGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseSystemJumps(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe system jumps: %w", describeError(err, resp))
	}
	return result, nil
}

func (c *Client) UniverseSystemKills(ctx context.Context) ([]esiapi.UniverseSystemKillsGetInner, error) {
	result, resp, err := c.api.UniverseAPI.GetUniverseSystemKills(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get universe system kills: %w", describeError(err, resp))
	}
	return result, nil
}

// UniverseIDsByName resolves names to IDs, grouped by category. Public, POST-based per the
// esi/assets.go precedent -- ESI uses POST here because the request carries a body, not because
// it mutates state.
func (c *Client) UniverseIDsByName(ctx context.Context, names []string) (*esiapi.UniverseIdsPost, error) {
	result, resp, err := c.api.UniverseAPI.PostUniverseIds(ctx).RequestBody(names).Execute()
	if err != nil {
		return nil, fmt.Errorf("resolve universe ids: %w", describeError(err, resp))
	}
	return result, nil
}

// UniverseNamesByID resolves IDs to names and their categories. Public, POST-based per the
// esi/assets.go precedent.
func (c *Client) UniverseNamesByID(ctx context.Context, ids []int64) ([]esiapi.UniverseNamesPostInner, error) {
	result, resp, err := c.api.UniverseAPI.PostUniverseNames(ctx).RequestBody(ids).Execute()
	if err != nil {
		return nil, fmt.Errorf("resolve universe names: %w", describeError(err, resp))
	}
	return result, nil
}
