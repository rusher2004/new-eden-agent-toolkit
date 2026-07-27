// Package esi's planetary_interaction.go covers all 4 of "Planetary Interaction"'s endpoints:
// list a character's colonized planets, get one colony's full layout (pins/links/routes), list a
// corporation's customs offices (POCOs), and look up a public PI schematic.
//
// The colony-layout endpoint (CharacterPlanetDetail) returns the most structurally complex model
// in this file -- but it is NOT recursive. Verified field-by-field against goesi-openapi
// v0.0.37's generated models: esiapi.CharactersCharacterIdPlanetsPlanetIdGet has exactly 3 flat
// sibling slice fields -- Links, Pins, Routes. Pins may carry one level of nested detail
// (ExtractorDetails, itself containing a Heads slice; FactoryDetails; Contents), but nothing in
// that nesting is a Pin, a Link, or a Route. Links and Routes reference pins purely by int64 ID
// (SourcePinId/DestinationPinId/Waypoints []int64), never by embedding another Link/Route/Pin
// value. There is no self-referential or mutually-recursive Go type here, unlike the shape
// Killmails had to split apart (see mcp/killmails.go's KillmailItem/KillmailInnerItem note) --
// this file needs no equivalent split at the esi/ layer since goesi-openapi already generated
// flat, non-recursive types.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterPlanets returns the client's own character's colonized planets. Private
// ("my own character" -- OAuth2 scope esi-planets.manage_planets.v1, verified against
// esi/api/openapi.yaml). Not paginated -- the generated request builder has no Page() setter.
func (c *Client) CharacterPlanets(ctx context.Context) ([]esiapi.CharactersCharacterIdPlanetsGetInner, error) {
	result, resp, err := c.api.PlanetaryInteractionAPI.GetCharactersCharacterIdPlanets(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character planets: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterPlanetDetail returns the full colony layout (links, pins, routes) of one of the
// client's own character's planets. Private ("my own character" -- OAuth2 scope
// esi-planets.manage_planets.v1, verified against esi/api/openapi.yaml). Not paginated -- the
// generated request builder has no Page() setter.
func (c *Client) CharacterPlanetDetail(ctx context.Context, planetID int64) (*esiapi.CharactersCharacterIdPlanetsPlanetIdGet, error) {
	result, resp, err := c.api.PlanetaryInteractionAPI.GetCharactersCharacterIdPlanetsPlanetId(ctx, int64(c.characterID), planetID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character planet detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationCustomsOffices returns corporationID's customs offices (POCOs), including their tax
// rates and reinforcement schedule. Private, corp-scoped like esi/industry.go's pattern: the
// authenticated character must hold sufficient corp roles (Director, per openapi.yaml's
// x-required-roles) -- ESI itself returns 403 otherwise; this facade adds no additional gate.
// OAuth2 scope esi-planets.read_customs_offices.v1, verified against esi/api/openapi.yaml.
// page=0 omits the page query parameter -- the generated request builder DOES have a Page()
// setter here (verified individually; the other 3 methods in this file do not).
func (c *Client) CorporationCustomsOffices(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdCustomsOfficesGetInner, int32, error) {
	req := c.api.PlanetaryInteractionAPI.GetCorporationsCorporationIdCustomsOffices(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation customs offices: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// PlanetarySchematic returns one PI schematic's detail (cycle time and name). Public -- verified
// against esi/api/openapi.yaml: this operation has no `security:` block at all, unlike the other
// 3 methods in this file. Not paginated.
func (c *Client) PlanetarySchematic(ctx context.Context, schematicID int64) (*esiapi.UniverseSchematicsSchematicIdGet, error) {
	result, resp, err := c.api.PlanetaryInteractionAPI.GetUniverseSchematicsSchematicId(ctx, schematicID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get planetary schematic: %w", describeError(err, resp))
	}
	return result, nil
}
