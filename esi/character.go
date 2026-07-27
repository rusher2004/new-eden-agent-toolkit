package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterDetail returns public info about an arbitrary character.
func (c *Client) CharacterDetail(ctx context.Context, characterID int64) (*esiapi.CharactersDetail, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersDetail(ctx, characterID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character detail: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterAffiliations resolves a set of character IDs to their corporation/alliance/faction.
func (c *Client) CharacterAffiliations(ctx context.Context, characterIDs []int64) ([]esiapi.CharactersAffiliationPostInner, error) {
	result, resp, err := c.api.CharacterAPI.PostCharactersAffiliation(ctx).RequestBody(characterIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character affiliations: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterCorporationHistory returns an arbitrary character's corporation history.
func (c *Client) CharacterCorporationHistory(ctx context.Context, characterID int64) ([]esiapi.CharactersCharacterIdCorporationhistoryGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdCorporationhistory(ctx, characterID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character corporation history: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterPortrait returns an arbitrary character's portrait URLs.
func (c *Client) CharacterPortrait(ctx context.Context, characterID int64) (*esiapi.CharactersCharacterIdPortraitGet, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdPortrait(ctx, characterID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character portrait: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterAgentsResearch returns the client's character's agent research progress.
func (c *Client) CharacterAgentsResearch(ctx context.Context) ([]esiapi.CharactersCharacterIdAgentsResearchGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdAgentsResearch(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character agents research: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterBlueprints returns the client's character's blueprints. page=0 omits the page query
// parameter.
func (c *Client) CharacterBlueprints(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdBlueprintsGetInner, int32, error) {
	req := c.api.CharacterAPI.GetCharactersCharacterIdBlueprints(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character blueprints: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CharacterFatigue returns the client's character's jump fatigue state.
func (c *Client) CharacterFatigue(ctx context.Context) (*esiapi.CharactersCharacterIdFatigueGet, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdFatigue(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character fatigue: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterMedals returns the client's character's medals.
func (c *Client) CharacterMedals(ctx context.Context) ([]esiapi.CharactersCharacterIdMedalsGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdMedals(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character medals: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterNotifications returns the client's character's notifications.
func (c *Client) CharacterNotifications(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdNotifications(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character notifications: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterNotificationContacts returns the client's character's contact-standing notifications.
func (c *Client) CharacterNotificationContacts(ctx context.Context) ([]esiapi.CharactersCharacterIdNotificationsContactsGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdNotificationsContacts(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character notification contacts: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterRoles returns the client's character's corporation roles.
func (c *Client) CharacterRoles(ctx context.Context) (*esiapi.CharactersCharacterIdRolesGet, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdRoles(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character roles: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterStandings returns the client's character's standings.
func (c *Client) CharacterStandings(ctx context.Context) ([]esiapi.CharactersCharacterIdStandingsGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdStandings(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character standings: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterTitles returns the client's character's corporation titles.
func (c *Client) CharacterTitles(ctx context.Context) ([]esiapi.CharactersCharacterIdTitlesGetInner, error) {
	result, resp, err := c.api.CharacterAPI.GetCharactersCharacterIdTitles(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character titles: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterCSPACost calculates the CSPA charge cost, from the client's character, to contact
// targetCharacterIDs.
func (c *Client) CharacterCSPACost(ctx context.Context, targetCharacterIDs []int64) (float64, error) {
	result, resp, err := c.api.CharacterAPI.PostCharactersCharacterIdCspa(ctx, int64(c.characterID)).RequestBody(targetCharacterIDs).Execute()
	if err != nil {
		return 0, fmt.Errorf("get character cspa cost: %w", describeError(err, resp))
	}
	return result, nil
}
