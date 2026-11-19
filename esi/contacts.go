package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterContacts returns the client's character's contacts. page=0 omits the page query
// parameter.
func (c *Client) CharacterContacts(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdContactsGetInner, int32, error) {
	req := c.api.ContactsAPI.GetCharactersCharacterIdContacts(ctx, int64(c.characterID))
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get character contacts: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CharacterContactLabels returns the client's character's contact labels.
func (c *Client) CharacterContactLabels(ctx context.Context) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
	result, resp, err := c.api.ContactsAPI.GetCharactersCharacterIdContactsLabels(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character contact labels: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationContacts returns corporationID's contacts. The authenticated character must hold
// sufficient corp roles — ESI itself returns 403 otherwise; no client-side check needed.
// page=0 omits the page query parameter.
func (c *Client) CorporationContacts(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContactsGetInner, int32, error) {
	req := c.api.ContactsAPI.GetCorporationsCorporationIdContacts(ctx, corporationID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get corporation contacts: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// CorporationContactLabels returns corporationID's contact labels.
func (c *Client) CorporationContactLabels(ctx context.Context, corporationID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
	result, resp, err := c.api.ContactsAPI.GetCorporationsCorporationIdContactsLabels(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation contact labels: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceContacts returns allianceID's contacts. page=0 omits the page query parameter.
func (c *Client) AllianceContacts(ctx context.Context, allianceID int64, page int32) ([]esiapi.AlliancesAllianceIdContactsGetInner, int32, error) {
	req := c.api.ContactsAPI.GetAlliancesAllianceIdContacts(ctx, allianceID)
	if page > 0 {
		req = req.Page(page)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("get alliance contacts: %w", describeError(err, resp))
	}
	return result, totalPagesFromHeader(resp), nil
}

// AllianceContactLabels returns allianceID's contact labels.
func (c *Client) AllianceContactLabels(ctx context.Context, allianceID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
	result, resp, err := c.api.ContactsAPI.GetAlliancesAllianceIdContactsLabels(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance contact labels: %w", describeError(err, resp))
	}
	return result, nil
}

// AddCharacterContacts adds contactIDs to the client's character's contact list with the given
// standing, optional label IDs, and watched flag. Returns the created contact IDs. This is a
// write: it changes state on FC's server.
func (c *Client) AddCharacterContacts(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error) {
	result, resp, err := c.api.ContactsAPI.PostCharactersCharacterIdContacts(ctx, int64(c.characterID)).
		Standing(standing).LabelIds(labelIDs).Watched(watched).RequestBody(contactIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("add character contacts: %w", describeError(err, resp))
	}
	return result, nil
}

// UpdateCharacterContacts updates the standing/labels/watched flag of existing contacts. This is
// a write: it changes state on FC's server. ESI returns no response body (204).
func (c *Client) UpdateCharacterContacts(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error {
	resp, err := c.api.ContactsAPI.PutCharactersCharacterIdContacts(ctx, int64(c.characterID)).
		Standing(standing).LabelIds(labelIDs).Watched(watched).RequestBody(contactIDs).Execute()
	if err != nil {
		return fmt.Errorf("update character contacts: %w", describeError(err, resp))
	}
	return nil
}

// RemoveCharacterContacts removes contactIDs from the client's character's contact list.
// contactIDs is sent as a query parameter, not a request body — verified against the real
// installed module. This is a write: it changes state on FC's server. ESI returns no response
// body (204).
func (c *Client) RemoveCharacterContacts(ctx context.Context, contactIDs []int64) error {
	resp, err := c.api.ContactsAPI.DeleteCharactersCharacterIdContacts(ctx, int64(c.characterID)).
		ContactIds(contactIDs).Execute()
	if err != nil {
		return fmt.Errorf("remove character contacts: %w", describeError(err, resp))
	}
	return nil
}
