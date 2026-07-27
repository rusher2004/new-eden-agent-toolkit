// Package esi's corporation.go covers all 22 of Corporation's endpoints: the first 14 (basics and
// membership/roles) plus the 8 assets/infrastructure-flavored endpoints (blueprints, container
// logs, facilities, medals, medals issued, starbases list/detail, structures) added in "Part 2" —
// see docs/superpowers/specs/2026-07-26-esi-corporation-part1-design.md and
// docs/superpowers/specs/2026-07-26-esi-corporation-part2-design.md.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CorporationDetail returns corporationID's basic public info.
func (c *Client) CorporationDetail(ctx context.Context, corporationID int64) (*esiapi.CorporationsDetail, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationId(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationAllianceHistory returns corporationID's alliance membership history.
func (c *Client) CorporationAllianceHistory(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdAlliancehistoryGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdAlliancehistory(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation alliance history: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationIcons returns corporationID's icon URLs.
func (c *Client) CorporationIcons(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdIconsGet, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdIcons(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation icons: %w", describeError(err, resp))
	}
	return result, nil
}

// NPCCorporations returns the IDs of every NPC corporation in the game. Takes no corporationID —
// this is a whole-game listing, not scoped to any one corporation.
func (c *Client) NPCCorporations(ctx context.Context) ([]int64, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsNpccorps(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get npc corporations: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationMembers returns corporationID's member character IDs. The authenticated character
// must hold sufficient corp roles -- ESI itself returns 403 otherwise.
func (c *Client) CorporationMembers(ctx context.Context, corporationID int64) ([]int64, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdMembers(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation members: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationMembersLimit returns corporationID's member limit.
func (c *Client) CorporationMembersLimit(ctx context.Context, corporationID int64) (int64, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdMembersLimit(ctx, corporationID).Execute()
	if err != nil {
		return 0, fmt.Errorf("get corporation members limit: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationMemberTitles returns each member's assigned title IDs.
func (c *Client) CorporationMemberTitles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdMembersTitlesGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdMembersTitles(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation member titles: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationMemberTracking returns member login/location/ship tracking data.
func (c *Client) CorporationMemberTracking(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdMembertrackingGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdMembertracking(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation member tracking: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationRoles returns each member's corp roles.
func (c *Client) CorporationRoles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdRolesGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdRoles(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation roles: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationRolesHistory returns corporationID's role-change history. page=0 omits the page
// query parameter entirely, letting ESI default to page 1.
func (c *Client) CorporationRolesHistory(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdRolesHistoryGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdRolesHistory(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation roles history: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationTitles returns corporationID's defined titles.
func (c *Client) CorporationTitles(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdTitlesGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdTitles(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation titles: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationDivisions returns corporationID's hangar and wallet division names.
func (c *Client) CorporationDivisions(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdDivisionsGet, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdDivisions(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation divisions: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationShareholders returns corporationID's shareholders. page=0 omits the page query
// parameter entirely, letting ESI default to page 1.
func (c *Client) CorporationShareholders(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdShareholdersGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdShareholders(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation shareholders: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationStandings returns corporationID's standings toward NPC entities. Returns the same
// generated model as Character's own standings (CharactersCharacterIdStandingsGetInner) -- verified
// from source, not assumed. page=0 omits the page query parameter entirely.
func (c *Client) CorporationStandings(ctx context.Context, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdStandingsGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdStandings(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation standings: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationBlueprints returns corporationID's blueprints. page=0 omits the page query parameter.
func (c *Client) CorporationBlueprints(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdBlueprintsGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdBlueprints(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation blueprints: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationContainerLogs returns corporationID's container access logs. page=0 omits the page
// query parameter.
func (c *Client) CorporationContainerLogs(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContainersLogsGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdContainersLogs(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation container logs: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationFacilities returns corporationID's facilities. Not paginated.
func (c *Client) CorporationFacilities(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdFacilitiesGetInner, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdFacilities(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation facilities: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationMedals returns corporationID's defined medals. page=0 omits the page query parameter.
func (c *Client) CorporationMedals(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdMedalsGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdMedals(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation medals: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationMedalsIssued returns medals corporationID has issued to members. page=0 omits the
// page query parameter.
func (c *Client) CorporationMedalsIssued(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdMedalsIssuedGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdMedalsIssued(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation medals issued: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationStarbases returns corporationID's starbases (POSes). page=0 omits the page query
// parameter.
func (c *Client) CorporationStarbases(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdStarbasesGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdStarbases(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation starbases: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationStarbaseDetail returns detail for one starbase. systemID is required by ESI to
// resolve the request (verified from the generated request builder's SystemId method) -- always
// sent, unlike page which is optionally omitted.
func (c *Client) CorporationStarbaseDetail(ctx context.Context, corporationID int64, starbaseID int64, systemID int64) (*esiapi.CorporationsCorporationIdStarbasesStarbaseIdGet, error) {
	result, resp, err := c.api.CorporationAPI.GetCorporationsCorporationIdStarbasesStarbaseId(ctx, corporationID, starbaseID).SystemId(systemID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation starbase detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationStructures returns corporationID's structures. page=0 omits the page query parameter.
func (c *Client) CorporationStructures(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdStructuresGetInner, int32, error) {
	req := c.api.CorporationAPI.GetCorporationsCorporationIdStructures(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation structures: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}
