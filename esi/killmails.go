// Package esi's killmails.go covers all 3 of "Killmails"'s endpoints: character/corp "recent
// killmails" listings (both page-paginated, both returning the SAME generated model -- verified
// from source, not a project typo) and a public killmail-detail lookup.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterKillmailsRecent returns the client's own character's recent killmail references
// (ID+hash pairs -- fetch full detail via KillmailDetail). page=0 omits the page query parameter.
func (c *Client) CharacterKillmailsRecent(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	req := c.api.KillmailsAPI.GetCharactersCharacterIdKillmailsRecent(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character killmails recent: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationKillmailsRecent returns corporationID's recent killmail references. The
// authenticated character must hold sufficient corp roles -- ESI itself returns 403 otherwise.
// Returns the exact same generated model as CharacterKillmailsRecent -- verified from the
// generated Execute() signature, not assumed. page=0 omits the page query parameter.
func (c *Client) CorporationKillmailsRecent(ctx context.Context, corporationID int64, page int32) ([]esiapi.CharactersCharacterIdKillmailsRecentGetInner, int32, error) {
	req := c.api.KillmailsAPI.GetCorporationsCorporationIdKillmailsRecent(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation killmails recent: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// KillmailDetail returns the full detail for one killmail. Public -- the hash itself functions as
// the access token ESI requires, no character auth needed.
//
// Quirk: the generated function's own argument order is (killmailHash, killmailId) -- reversed
// from the URL path (/killmails/{killmail_id}/{killmail_hash}/) and from this method's own
// (killmailID, killmailHash) signature. This facade normalizes to (killmailID, killmailHash) to
// match the URL and this project's tool input order, swapping internally -- same pattern as
// CorporationContractBids (esi/contracts.go).
func (c *Client) KillmailDetail(ctx context.Context, killmailID int64, killmailHash string) (*esiapi.KillmailsKillmailIdKillmailHashGet, error) {
	result, resp, err := c.api.KillmailsAPI.GetKillmailsKillmailIdKillmailHash(ctx, killmailHash, killmailID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get killmail detail: %w", describeError(err, resp))
	}
	return result, nil
}
