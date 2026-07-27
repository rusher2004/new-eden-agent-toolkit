// Package esi's mail.go covers all 9 of "Mail"'s endpoints. All 9 are private/"my own character"
// -- verified against esi/api/openapi.yaml's security blocks, every one of the 9
// /characters/{character_id}/mail... paths requires an esi-mail.{read_mail,send_mail,organize_mail}.v1
// character-scoped OAuth2 scope, never a caller-supplied character ID.
//
// CharacterMailHeaders' pagination is NOT the page/X-Pages style used elsewhere in this project
// (see totalPagesFromHeader in esi/contracts.go) -- verified from the generated request builder's
// own optional params (esi/api_mail.go's ApiGetCharactersCharacterIdMailRequest), ESI's mail
// listing endpoint offers only an optional `labels []int64` filter plus a `lastMailId *int64`
// cursor for paging backwards through older mail; there is no page number or total-pages header.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterMailHeaders returns up to the 50 most recent mail headers for the client's own
// character, optionally filtered to labelIDs and/or paged backwards from lastMailID (both
// verified from the generated request builder -- ESI's mail listing has no page-number/X-Pages
// pagination, only this label-filter-plus-cursor scheme). An empty labelIDs and lastMailID=0 omit
// both query parameters.
func (c *Client) CharacterMailHeaders(ctx context.Context, labelIDs []int64, lastMailID int64) ([]esiapi.CharactersCharacterIdMailGetInner, error) {
	req := c.api.MailAPI.GetCharactersCharacterIdMail(ctx, int64(c.characterID))
	if len(labelIDs) > 0 {
		req = req.Labels(labelIDs)
	}
	if lastMailID > 0 {
		req = req.LastMailId(lastMailID)
	}
	result, resp, err := req.Execute()
	if err != nil {
		return nil, fmt.Errorf("get character mail headers: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterMailLabels returns the client's own character's mail labels, their individual unread
// counts, and the total unread count across all mail.
func (c *Client) CharacterMailLabels(ctx context.Context) (*esiapi.CharactersCharacterIdMailLabelsGet, error) {
	result, resp, err := c.api.MailAPI.GetCharactersCharacterIdMailLabels(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character mail labels: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterMailLists returns every mailing list the client's own character is subscribed to.
func (c *Client) CharacterMailLists(ctx context.Context) ([]esiapi.CharactersCharacterIdMailListsGetInner, error) {
	result, resp, err := c.api.MailAPI.GetCharactersCharacterIdMailLists(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character mail lists: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterMail returns the full body and recipients of one of the client's own character's
// mails.
func (c *Client) CharacterMail(ctx context.Context, mailID int64) (*esiapi.CharactersCharacterIdMailMailIdGet, error) {
	result, resp, err := c.api.MailAPI.GetCharactersCharacterIdMailMailId(ctx, int64(c.characterID), mailID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character mail: %w", describeError(err, resp))
	}
	return result, nil
}

// SendCharacterMail sends a new mail as the client's own character and returns its assigned mail
// ID -- the only field ESI's response contains. approvedCost is nil unless the mail is being sent
// through a mailing list that requires CSPA charge pre-approval (see get_character_cspa_cost).
func (c *Client) SendCharacterMail(ctx context.Context, subject, body string, recipients []esiapi.PostCharactersCharacterIdMailRequestRecipientsInner, approvedCost *int64) (int64, error) {
	result, resp, err := c.api.MailAPI.PostCharactersCharacterIdMail(ctx, int64(c.characterID)).
		PostCharactersCharacterIdMailRequest(esiapi.PostCharactersCharacterIdMailRequest{
			ApprovedCost: approvedCost, Body: body, Recipients: recipients, Subject: subject,
		}).Execute()
	if err != nil {
		return 0, fmt.Errorf("send character mail: %w", describeError(err, resp))
	}
	return result, nil
}

// CreateCharacterMailLabel creates a new mail label for the client's own character and returns
// its assigned label ID -- the only field ESI's response contains. color is nil to accept ESI's
// default label color.
func (c *Client) CreateCharacterMailLabel(ctx context.Context, name string, color *string) (int64, error) {
	result, resp, err := c.api.MailAPI.PostCharactersCharacterIdMailLabels(ctx, int64(c.characterID)).
		PostCharactersCharacterIdMailLabelsRequest(esiapi.PostCharactersCharacterIdMailLabelsRequest{
			Color: color, Name: name,
		}).Execute()
	if err != nil {
		return 0, fmt.Errorf("create character mail label: %w", describeError(err, resp))
	}
	return result, nil
}

// UpdateCharacterMail updates the read status and/or label assignments of one of the client's own
// character's mails. ESI's response has no body -- same shape as DeleteCharacterFitting
// (esi/fittings.go). labels=nil leaves label assignments unchanged; read=nil leaves read status
// unchanged (both per ESI's own "omit to leave unchanged" semantics for this endpoint).
func (c *Client) UpdateCharacterMail(ctx context.Context, mailID int64, labels []int64, read *bool) error {
	resp, err := c.api.MailAPI.PutCharactersCharacterIdMailMailId(ctx, int64(c.characterID), mailID).
		PutCharactersCharacterIdMailMailIdRequest(esiapi.PutCharactersCharacterIdMailMailIdRequest{
			Labels: labels, Read: read,
		}).Execute()
	if err != nil {
		return fmt.Errorf("update character mail: %w", describeError(err, resp))
	}
	return nil
}

// DeleteCharacterMail deletes one of the client's own character's mails. ESI's response has no
// body -- same shape as DeleteCharacterFitting (esi/fittings.go).
func (c *Client) DeleteCharacterMail(ctx context.Context, mailID int64) error {
	resp, err := c.api.MailAPI.DeleteCharactersCharacterIdMailMailId(ctx, int64(c.characterID), mailID).Execute()
	if err != nil {
		return fmt.Errorf("delete character mail: %w", describeError(err, resp))
	}
	return nil
}

// DeleteCharacterMailLabel deletes one of the client's own character's mail labels. ESI's
// response has no body -- same shape as DeleteCharacterFitting (esi/fittings.go).
func (c *Client) DeleteCharacterMailLabel(ctx context.Context, labelID int64) error {
	resp, err := c.api.MailAPI.DeleteCharactersCharacterIdMailLabelsLabelId(ctx, int64(c.characterID), labelID).Execute()
	if err != nil {
		return fmt.Errorf("delete character mail label: %w", describeError(err, resp))
	}
	return nil
}
