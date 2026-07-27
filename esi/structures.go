// Package esi's structures.go covers all 6 of "Structures"'s endpoints -- a character's own
// mercenary dens, and a corporation's skyhooks and sovereignty hubs. Mercenary dens are
// private/"my own character" (esi-structures.read_character.v1); skyhooks and sovereignty hubs
// are corp-scoped (esi-structures.read_corporation.v1), same convention as
// esi/planetary_interaction.go's CorporationCustomsOffices -- ESI's own 403 is the sole
// authorization gate, no additional role check in this facade. None of the 6 endpoints is
// paginated (verified individually: no .Page() setter on any of the 6 request builders).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterMercenaryDens returns the client's own character's mercenary dens (bare ID + planet ID
// per den; call CharacterMercenaryDenDetail for full detail on one).
func (c *Client) CharacterMercenaryDens(ctx context.Context) ([]esiapi.CharactersStructuresMercenaryDensListingMercenaryden, error) {
	result, resp, err := c.api.StructuresAPI.GetCharactersStructuresMercenaryDensListing(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character mercenary dens: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get character mercenary dens: empty response")
	}
	return result.MercenaryDens, nil
}

// CharacterMercenaryDenDetail returns one of the client's own character's mercenary dens in full
// detail: evolution stats, infomorph count, reinforcement timer (if any), and the skyhook it's
// attached to.
func (c *Client) CharacterMercenaryDenDetail(ctx context.Context, mercenaryDenID int64) (*esiapi.CharactersStructuresMercenaryDensDetail, error) {
	result, resp, err := c.api.StructuresAPI.GetCharactersStructuresMercenaryDensDetail(ctx, mercenaryDenID, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character mercenary den detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationSkyhooks returns corporationID's skyhooks (bare ID + planet ID per skyhook; call
// CorporationSkyhookDetail for full detail on one). Corp-scoped: the authenticated character must
// hold sufficient corp roles per ESI's own x-required-roles -- this facade adds no additional
// gate, ESI's 403 is authoritative.
func (c *Client) CorporationSkyhooks(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSkyhooksListingSkyhook, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSkyhooksListing(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation skyhooks: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get corporation skyhooks: empty response")
	}
	return result.Skyhooks, nil
}

// CorporationSkyhookDetail returns one of corporationID's skyhooks in full detail: workforce,
// reagent stock, active/reinforcement state, and theft vulnerability window (if any).
func (c *Client) CorporationSkyhookDetail(ctx context.Context, corporationID, skyhookID int64) (*esiapi.CorporationsStructuresSkyhooksDetail, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSkyhooksDetail(ctx, skyhookID, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation skyhook detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationSovereigntyHubs returns corporationID's sovereignty hubs (bare ID + solar system ID
// per hub; call CorporationSovereigntyHubDetail for full detail on one).
func (c *Client) CorporationSovereigntyHubs(ctx context.Context, corporationID int64) ([]esiapi.CorporationsStructuresSovereigntyHubsListingSovereigntyhub, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSovereigntyHubsListing(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation sovereignty hubs: %w", describeError(err, resp))
	}
	if result == nil {
		return nil, fmt.Errorf("get corporation sovereignty hubs: empty response")
	}
	return result.SovereigntyHubs, nil
}

// CorporationSovereigntyHubDetail returns one of corporationID's sovereignty hubs in full detail:
// reagent bay, power/workforce resources, installed upgrades, vulnerability window (if any), and
// workforce transport configuration/state.
func (c *Client) CorporationSovereigntyHubDetail(ctx context.Context, corporationID, sovereigntyHubID int64) (*esiapi.CorporationsStructuresSovereigntyHubsDetail, error) {
	result, resp, err := c.api.StructuresAPI.GetCorporationsStructuresSovereigntyHubsDetail(ctx, sovereigntyHubID, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation sovereignty hub detail: %w", describeError(err, resp))
	}
	return result, nil
}
