// Package esi's faction_warfare.go covers all 8 of "Faction Warfare"'s endpoints. 2 are private
// (CharacterFwStats -- "my own" character, corp-scoped CorporationFwStats), 6 are public
// (leaderboards x3, per-faction stats, systems, wars).
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterFwStats returns the client's own character's faction warfare stats. Uses the Client's
// own baked-in characterID -- unlike Corporation Projects' Contribution endpoint, this ESI
// endpoint is genuinely "my own stats," not "look up an arbitrary character."
func (c *Client) CharacterFwStats(ctx context.Context) (*esiapi.CharactersCharacterIdFwStatsGet, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetCharactersCharacterIdFwStats(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character fw stats: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationFwStats returns corporationID's faction warfare stats. The authenticated character
// must hold sufficient corp roles -- ESI itself returns 403 otherwise.
func (c *Client) CorporationFwStats(ctx context.Context, corporationID int64) (*esiapi.CorporationsCorporationIdFwStatsGet, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetCorporationsCorporationIdFwStats(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation fw stats: %w", describeError(err, resp))
	}
	return result, nil
}

// FwLeaderboards returns the top-4 faction rankings by kills and victory points.
func (c *Client) FwLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsGet, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwLeaderboards(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw leaderboards: %w", describeError(err, resp))
	}
	return result, nil
}

// FwCharacterLeaderboards returns the top-100 pilot rankings by kills and victory points.
func (c *Client) FwCharacterLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsCharactersGet, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwLeaderboardsCharacters(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw character leaderboards: %w", describeError(err, resp))
	}
	return result, nil
}

// FwCorporationLeaderboards returns the top-10 corporation rankings by kills and victory points.
func (c *Client) FwCorporationLeaderboards(ctx context.Context) (*esiapi.FwLeaderboardsCorporationsGet, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwLeaderboardsCorporations(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw corporation leaderboards: %w", describeError(err, resp))
	}
	return result, nil
}

// FwStats returns per-faction faction warfare stats for every faction.
func (c *Client) FwStats(ctx context.Context) ([]esiapi.FwStatsGetInner, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwStats(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw stats: %w", describeError(err, resp))
	}
	return result, nil
}

// FwSystems returns every solar system's faction warfare occupation/contest state.
func (c *Client) FwSystems(ctx context.Context) ([]esiapi.FwSystemsGetInner, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwSystems(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw systems: %w", describeError(err, resp))
	}
	return result, nil
}

// FwWars returns every faction warfare declaration (which factions are at war with each other).
func (c *Client) FwWars(ctx context.Context) ([]esiapi.FwWarsGetInner, error) {
	result, resp, err := c.api.FactionWarfareAPI.GetFwWars(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("get fw wars: %w", describeError(err, resp))
	}
	return result, nil
}
