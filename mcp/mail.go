package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// mailESI is the ESI calls the Mail tools make.
type mailESI interface {
	CharacterMailHeaders(ctx context.Context, labelIDs []int64, lastMailID int64) ([]esiapi.CharactersCharacterIdMailGetInner, error)
	CharacterMailLabels(ctx context.Context) (*esiapi.CharactersCharacterIdMailLabelsGet, error)
	CharacterMailLists(ctx context.Context) ([]esiapi.CharactersCharacterIdMailListsGetInner, error)
	CharacterMail(ctx context.Context, mailID int64) (*esiapi.CharactersCharacterIdMailMailIdGet, error)
	SendCharacterMail(ctx context.Context, subject, body string, recipients []esiapi.PostCharactersCharacterIdMailRequestRecipientsInner, approvedCost *int64) (int64, error)
	CreateCharacterMailLabel(ctx context.Context, name string, color *string) (int64, error)
	UpdateCharacterMail(ctx context.Context, mailID int64, labels []int64, read *bool) error
	DeleteCharacterMail(ctx context.Context, mailID int64) error
	DeleteCharacterMailLabel(ctx context.Context, labelID int64) error
}

// MailRecipient is shared by list_character_mail and get_character_mail's outputs since ESI
// returns the exact same generated recipient shape from both endpoints
// (PostCharactersCharacterIdMailRequestRecipientsInner -- yes, the "Post" name even though this
// usage is read-only; verified from the generated model files, not a project typo, same class of
// quirk as KillmailRef's naming). RecipientType is a plain string, not a generated enum type --
// ESI documents its values (character/corporation/alliance/mailing_list) but goesi-openapi
// doesn't generate a Go enum for it, so this facade doesn't invent one either. This is NOT the
// self-referential-type trap from Killmails: it's a flat struct with a discriminator string
// field, not a oneOf/polymorphic or nested-tree shape.
type MailRecipient struct {
	RecipientID   int64  `json:"recipientId"`
	RecipientType string `json:"recipientType"`
}

func toMailRecipients(recipients []esiapi.PostCharactersCharacterIdMailRequestRecipientsInner) []MailRecipient {
	out := make([]MailRecipient, len(recipients))
	for i, r := range recipients {
		out[i] = MailRecipient{RecipientID: r.RecipientId, RecipientType: r.RecipientType}
	}
	return out
}

// --- list_character_mail (private, read) ---

// MailHeader mirrors CharactersCharacterIdMailGetInner. Every field is a pointer because ESI
// itself marks every field of this response optional (verified from the generated model, not
// assumed) -- e.g. IsRead/Subject can be genuinely absent on some mail headers.
type MailHeader struct {
	MailID     *int64          `json:"mailId,omitempty"`
	From       *int64          `json:"from,omitempty"`
	Subject    *string         `json:"subject,omitempty"`
	Timestamp  *time.Time      `json:"timestamp,omitempty"`
	IsRead     *bool           `json:"isRead,omitempty"`
	Labels     []int64         `json:"labels,omitempty"`
	Recipients []MailRecipient `json:"recipients,omitempty"`
}

func toMailHeaders(headers []esiapi.CharactersCharacterIdMailGetInner) []MailHeader {
	out := make([]MailHeader, len(headers))
	for i, h := range headers {
		out[i] = MailHeader{
			MailID: h.MailId, From: h.From, Subject: h.Subject, Timestamp: h.Timestamp,
			IsRead: h.IsRead, Labels: h.Labels, Recipients: toMailRecipients(h.Recipients),
		}
	}
	return out
}

// minMailID returns the lowest non-nil MailID across headers, for ListCharacterMailOutput's
// NextLastMailID -- see that field's own doc comment for why.
func minMailID(headers []MailHeader) *int64 {
	var min *int64
	for _, h := range headers {
		if h.MailID == nil {
			continue
		}
		if min == nil || *h.MailID < *min {
			min = h.MailID
		}
	}
	return min
}

func listCharacterMail(ctx context.Context, client mailESI, labelIDs []int64, lastMailID int64) ([]MailHeader, error) {
	headers, err := client.CharacterMailHeaders(ctx, labelIDs, lastMailID)
	if err != nil {
		return nil, err
	}
	return toMailHeaders(headers), nil
}

type ListCharacterMailInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	LabelIDs    []int64 `json:"labelIds,omitempty" jsonschema:"only return mail carrying at least one of these label IDs; omit for no label filter"`
	LastMailID  int64   `json:"lastMailId,omitempty" jsonschema:"page backwards through older mail: omit for the most recent mail, or pass the previous call's nextLastMailId to fetch the next older batch"`
}

// ListCharacterMailOutput's pagination is ESI's own cursor scheme, not this project's usual
// page/totalPages shape (see esi/mail.go's package doc for why) -- there is no total count or
// page number to report, only "here's where to resume paging backwards from".
type ListCharacterMailOutput struct {
	CharacterID    int32        `json:"characterId"`
	Mail           []MailHeader `json:"mail"`
	NextLastMailID *int64       `json:"nextLastMailId,omitempty"`
}

// AddListCharacterMail registers list_character_mail. open builds the character's ESI
// client; any client with the mailESI methods will do.
func AddListCharacterMail[C mailESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_mail",
		Description: "List a logged-in EVE Online character's recent mail headers, optionally filtered by label and paged backwards via a mail ID cursor.",
	}, listCharacterMailHandler(open, store))
}

func listCharacterMailHandler[C mailESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterMailInput, ListCharacterMailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMailInput) (*mcpsdk.CallToolResult, ListCharacterMailOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMailOutput{}, err
		}
		if !found {
			return nil, ListCharacterMailOutput{}, fmt.Errorf("list character mail for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMailOutput{}, fmt.Errorf("list character mail: %w", err)
		}
		headers, err := listCharacterMail(ctx, client, in.LabelIDs, in.LastMailID)
		if err != nil {
			return nil, ListCharacterMailOutput{}, fmt.Errorf("list character mail: %w", err)
		}
		return nil, ListCharacterMailOutput{CharacterID: in.CharacterID, Mail: headers, NextLastMailID: minMailID(headers)}, nil
	}
}

// --- get_character_mail_labels (private, read) ---

type MailLabel struct {
	LabelID     *int64  `json:"labelId,omitempty"`
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	UnreadCount *int64  `json:"unreadCount,omitempty"`
}

func getCharacterMailLabels(ctx context.Context, client mailESI) ([]MailLabel, *int64, error) {
	labels, err := client.CharacterMailLabels(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]MailLabel, len(labels.Labels))
	for i, l := range labels.Labels {
		out[i] = MailLabel{LabelID: l.LabelId, Name: l.Name, Color: l.Color, UnreadCount: l.UnreadCount}
	}
	return out, labels.TotalUnreadCount, nil
}

type GetCharacterMailLabelsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type GetCharacterMailLabelsOutput struct {
	CharacterID      int32       `json:"characterId"`
	Labels           []MailLabel `json:"labels"`
	TotalUnreadCount *int64      `json:"totalUnreadCount,omitempty"`
}

// AddGetCharacterMailLabels registers get_character_mail_labels. open builds the character's ESI
// client; any client with the mailESI methods will do.
func AddGetCharacterMailLabels[C mailESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_mail_labels",
		Description: "Get a logged-in EVE Online character's mail labels and unread counts.",
	}, getCharacterMailLabelsHandler(open, store))
}

func getCharacterMailLabelsHandler[C mailESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCharacterMailLabelsInput, GetCharacterMailLabelsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterMailLabelsInput) (*mcpsdk.CallToolResult, GetCharacterMailLabelsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterMailLabelsOutput{}, err
		}
		if !found {
			return nil, GetCharacterMailLabelsOutput{}, fmt.Errorf("get character mail labels for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterMailLabelsOutput{}, fmt.Errorf("get character mail labels: %w", err)
		}
		labels, totalUnread, err := getCharacterMailLabels(ctx, client)
		if err != nil {
			return nil, GetCharacterMailLabelsOutput{}, fmt.Errorf("get character mail labels: %w", err)
		}
		return nil, GetCharacterMailLabelsOutput{CharacterID: in.CharacterID, Labels: labels, TotalUnreadCount: totalUnread}, nil
	}
}

// --- list_character_mail_lists (private, read) ---

// MailList mirrors CharactersCharacterIdMailListsGetInner, whose two fields are both required
// (non-pointer) in the generated model -- verified from source, unlike MailHeader's all-optional
// fields above.
type MailList struct {
	MailingListID int64  `json:"mailingListId"`
	Name          string `json:"name"`
}

func listCharacterMailLists(ctx context.Context, client mailESI) ([]MailList, error) {
	lists, err := client.CharacterMailLists(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MailList, len(lists))
	for i, l := range lists {
		out[i] = MailList{MailingListID: l.MailingListId, Name: l.Name}
	}
	return out, nil
}

type ListCharacterMailListsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterMailListsOutput struct {
	CharacterID int32      `json:"characterId"`
	MailLists   []MailList `json:"mailLists"`
}

// AddListCharacterMailLists registers list_character_mail_lists. open builds the character's ESI
// client; any client with the mailESI methods will do.
func AddListCharacterMailLists[C mailESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_mail_lists",
		Description: "List the mailing lists a logged-in EVE Online character is subscribed to.",
	}, listCharacterMailListsHandler(open, store))
}

func listCharacterMailListsHandler[C mailESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterMailListsInput, ListCharacterMailListsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterMailListsInput) (*mcpsdk.CallToolResult, ListCharacterMailListsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMailListsOutput{}, err
		}
		if !found {
			return nil, ListCharacterMailListsOutput{}, fmt.Errorf("list character mail lists for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterMailListsOutput{}, fmt.Errorf("list character mail lists: %w", err)
		}
		lists, err := listCharacterMailLists(ctx, client)
		if err != nil {
			return nil, ListCharacterMailListsOutput{}, fmt.Errorf("list character mail lists: %w", err)
		}
		return nil, ListCharacterMailListsOutput{CharacterID: in.CharacterID, MailLists: lists}, nil
	}
}

// --- get_character_mail (private, read) ---

func getCharacterMail(ctx context.Context, client mailESI, mailID int64) (GetCharacterMailOutput, error) {
	mail, err := client.CharacterMail(ctx, mailID)
	if err != nil {
		return GetCharacterMailOutput{}, err
	}
	return GetCharacterMailOutput{
		MailID: mailID, Body: mail.Body, From: mail.From, Subject: mail.Subject, Timestamp: mail.Timestamp,
		Read: mail.Read, Labels: mail.Labels, Recipients: toMailRecipients(mail.Recipients),
	}, nil
}

type GetCharacterMailInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	MailID      int64 `json:"mailId" jsonschema:"the mail ID, from a list_character_mail result"`
}
type GetCharacterMailOutput struct {
	CharacterID int32           `json:"characterId"`
	MailID      int64           `json:"mailId"`
	From        *int64          `json:"from,omitempty"`
	Subject     *string         `json:"subject,omitempty"`
	Body        *string         `json:"body,omitempty"`
	Timestamp   *time.Time      `json:"timestamp,omitempty"`
	Read        *bool           `json:"read,omitempty"`
	Labels      []int64         `json:"labels,omitempty"`
	Recipients  []MailRecipient `json:"recipients,omitempty"`
}

// AddGetCharacterMail registers get_character_mail. open builds the character's ESI
// client; any client with the mailESI methods will do.
func AddGetCharacterMail[C mailESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_character_mail",
		Description: "Get the full body and recipients of one of a logged-in EVE Online character's mails.",
	}, getCharacterMailHandler(open, store))
}

func getCharacterMailHandler[C mailESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetCharacterMailInput, GetCharacterMailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetCharacterMailInput) (*mcpsdk.CallToolResult, GetCharacterMailOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, GetCharacterMailOutput{}, err
		}
		if !found {
			return nil, GetCharacterMailOutput{}, fmt.Errorf("get character mail for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetCharacterMailOutput{}, fmt.Errorf("get character mail: %w", err)
		}
		out, err := getCharacterMail(ctx, client, in.MailID)
		if err != nil {
			return nil, GetCharacterMailOutput{}, fmt.Errorf("get character mail: %w", err)
		}
		out.CharacterID = in.CharacterID
		return nil, out, nil
	}
}

// --- send_character_mail (private, write) ---

// MailRecipientInput is send_character_mail's input counterpart to MailRecipient -- same split
// as FittingItem/FittingItemInput in mcp/fittings.go.
type MailRecipientInput struct {
	RecipientID   int64  `json:"recipientId" jsonschema:"the recipient's character, corporation, alliance, or mailing list ID"`
	RecipientType string `json:"recipientType" jsonschema:"one of: character, corporation, alliance, mailing_list"`
}

func sendCharacterMail(ctx context.Context, client mailESI, subject, body string, recipients []MailRecipientInput, approvedCost *int64) (int64, error) {
	esiRecipients := make([]esiapi.PostCharactersCharacterIdMailRequestRecipientsInner, len(recipients))
	for i, r := range recipients {
		esiRecipients[i] = esiapi.PostCharactersCharacterIdMailRequestRecipientsInner{RecipientId: r.RecipientID, RecipientType: r.RecipientType}
	}
	return client.SendCharacterMail(ctx, subject, body, esiRecipients, approvedCost)
}

type SendCharacterMailInput struct {
	CharacterID  int32                `json:"characterId" jsonschema:"the character ID to check"`
	Subject      string               `json:"subject" jsonschema:"the mail's subject line"`
	Body         string               `json:"body" jsonschema:"the mail's body"`
	Recipients   []MailRecipientInput `json:"recipients" jsonschema:"who to send the mail to"`
	ApprovedCost *int64               `json:"approvedCost,omitempty" jsonschema:"pre-approved CSPA charge cost; omit unless sending through a mailing list that requires it (see get_character_cspa_cost)"`
}
type SendCharacterMailOutput struct {
	CharacterID int32 `json:"characterId"`
	MailID      int64 `json:"mailId"`
}

// AddSendCharacterMail registers send_character_mail. open builds the character's ESI
// client; any client with the mailESI methods will do.
func AddSendCharacterMail[C mailESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "send_character_mail",
		Description: "Send a new mail as a logged-in EVE Online character. This changes state on EVE's server.",
	}, sendCharacterMailHandler(open, store))
}

func sendCharacterMailHandler[C mailESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[SendCharacterMailInput, SendCharacterMailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SendCharacterMailInput) (*mcpsdk.CallToolResult, SendCharacterMailOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, SendCharacterMailOutput{}, err
		}
		if !found {
			return nil, SendCharacterMailOutput{}, fmt.Errorf("send character mail for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, SendCharacterMailOutput{}, fmt.Errorf("send character mail: %w", err)
		}
		mailID, err := sendCharacterMail(ctx, client, in.Subject, in.Body, in.Recipients, in.ApprovedCost)
		if err != nil {
			return nil, SendCharacterMailOutput{}, fmt.Errorf("send character mail: %w", err)
		}
		return nil, SendCharacterMailOutput{CharacterID: in.CharacterID, MailID: mailID}, nil
	}
}

// --- create_character_mail_label (private, write) ---

func createCharacterMailLabel(ctx context.Context, client mailESI, name string, color *string) (int64, error) {
	return client.CreateCharacterMailLabel(ctx, name, color)
}

type CreateCharacterMailLabelInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	Name        string  `json:"name" jsonschema:"the label's name"`
	Color       *string `json:"color,omitempty" jsonschema:"hex color for the label, e.g. #0000fe; omit to accept ESI's default"`
}
type CreateCharacterMailLabelOutput struct {
	CharacterID int32 `json:"characterId"`
	LabelID     int64 `json:"labelId"`
}

// AddCreateCharacterMailLabel registers create_character_mail_label. open builds the character's ESI
// client; any client with the mailESI methods will do.
func AddCreateCharacterMailLabel[C mailESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_character_mail_label",
		Description: "Create a new mail label for a logged-in EVE Online character. This changes state on EVE's server.",
	}, createCharacterMailLabelHandler(open, store))
}

func createCharacterMailLabelHandler[C mailESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[CreateCharacterMailLabelInput, CreateCharacterMailLabelOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateCharacterMailLabelInput) (*mcpsdk.CallToolResult, CreateCharacterMailLabelOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, CreateCharacterMailLabelOutput{}, err
		}
		if !found {
			return nil, CreateCharacterMailLabelOutput{}, fmt.Errorf("create character mail label for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, CreateCharacterMailLabelOutput{}, fmt.Errorf("create character mail label: %w", err)
		}
		labelID, err := createCharacterMailLabel(ctx, client, in.Name, in.Color)
		if err != nil {
			return nil, CreateCharacterMailLabelOutput{}, fmt.Errorf("create character mail label: %w", err)
		}
		return nil, CreateCharacterMailLabelOutput{CharacterID: in.CharacterID, LabelID: labelID}, nil
	}
}

// --- update_character_mail (private, write) ---

func updateCharacterMail(ctx context.Context, client mailESI, mailID int64, labels []int64, read *bool) error {
	return client.UpdateCharacterMail(ctx, mailID, labels, read)
}

type UpdateCharacterMailInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	MailID      int64   `json:"mailId" jsonschema:"the mail ID to update"`
	Labels      []int64 `json:"labels,omitempty" jsonschema:"replace the mail's label IDs with this set; omit to leave labels unchanged"`
	Read        *bool   `json:"read,omitempty" jsonschema:"mark the mail read (true) or unread (false); omit to leave read status unchanged"`
}
type UpdateCharacterMailOutput struct {
	CharacterID int32 `json:"characterId"`
	MailID      int64 `json:"mailId"`
}

// AddUpdateCharacterMail registers update_character_mail. open builds the character's ESI
// client; any client with the mailESI methods will do.
func AddUpdateCharacterMail[C mailESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "update_character_mail",
		Description: "Update the read status and/or labels of one of a logged-in EVE Online character's mails. This changes state on EVE's server.",
	}, updateCharacterMailHandler(open, store))
}

func updateCharacterMailHandler[C mailESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[UpdateCharacterMailInput, UpdateCharacterMailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in UpdateCharacterMailInput) (*mcpsdk.CallToolResult, UpdateCharacterMailOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, UpdateCharacterMailOutput{}, err
		}
		if !found {
			return nil, UpdateCharacterMailOutput{}, fmt.Errorf("update character mail for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, UpdateCharacterMailOutput{}, fmt.Errorf("update character mail: %w", err)
		}
		if err := updateCharacterMail(ctx, client, in.MailID, in.Labels, in.Read); err != nil {
			return nil, UpdateCharacterMailOutput{}, fmt.Errorf("update character mail: %w", err)
		}
		return nil, UpdateCharacterMailOutput{CharacterID: in.CharacterID, MailID: in.MailID}, nil
	}
}

// --- delete_character_mail (private, write) ---

func deleteCharacterMail(ctx context.Context, client mailESI, mailID int64) error {
	return client.DeleteCharacterMail(ctx, mailID)
}

type DeleteCharacterMailInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	MailID      int64 `json:"mailId" jsonschema:"the mail ID to delete"`
}
type DeleteCharacterMailOutput struct {
	CharacterID int32 `json:"characterId"`
	MailID      int64 `json:"mailId"`
}

// AddDeleteCharacterMail registers delete_character_mail. open builds the character's ESI
// client; any client with the mailESI methods will do.
func AddDeleteCharacterMail[C mailESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "delete_character_mail",
		Description: "Delete one of a logged-in EVE Online character's mails. This changes state on EVE's server.",
	}, deleteCharacterMailHandler(open, store))
}

func deleteCharacterMailHandler[C mailESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[DeleteCharacterMailInput, DeleteCharacterMailOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteCharacterMailInput) (*mcpsdk.CallToolResult, DeleteCharacterMailOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, DeleteCharacterMailOutput{}, err
		}
		if !found {
			return nil, DeleteCharacterMailOutput{}, fmt.Errorf("delete character mail for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, DeleteCharacterMailOutput{}, fmt.Errorf("delete character mail: %w", err)
		}
		if err := deleteCharacterMail(ctx, client, in.MailID); err != nil {
			return nil, DeleteCharacterMailOutput{}, fmt.Errorf("delete character mail: %w", err)
		}
		return nil, DeleteCharacterMailOutput{CharacterID: in.CharacterID, MailID: in.MailID}, nil
	}
}

// --- delete_character_mail_label (private, write) ---

func deleteCharacterMailLabel(ctx context.Context, client mailESI, labelID int64) error {
	return client.DeleteCharacterMailLabel(ctx, labelID)
}

type DeleteCharacterMailLabelInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	LabelID     int64 `json:"labelId" jsonschema:"the label ID to delete"`
}
type DeleteCharacterMailLabelOutput struct {
	CharacterID int32 `json:"characterId"`
	LabelID     int64 `json:"labelId"`
}

// AddDeleteCharacterMailLabel registers delete_character_mail_label. open builds the character's ESI
// client; any client with the mailESI methods will do.
func AddDeleteCharacterMailLabel[C mailESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "delete_character_mail_label",
		Description: "Delete one of a logged-in EVE Online character's mail labels. This changes state on EVE's server.",
	}, deleteCharacterMailLabelHandler(open, store))
}

func deleteCharacterMailLabelHandler[C mailESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[DeleteCharacterMailLabelInput, DeleteCharacterMailLabelOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteCharacterMailLabelInput) (*mcpsdk.CallToolResult, DeleteCharacterMailLabelOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, DeleteCharacterMailLabelOutput{}, err
		}
		if !found {
			return nil, DeleteCharacterMailLabelOutput{}, fmt.Errorf("delete character mail label for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, DeleteCharacterMailLabelOutput{}, fmt.Errorf("delete character mail label: %w", err)
		}
		if err := deleteCharacterMailLabel(ctx, client, in.LabelID); err != nil {
			return nil, DeleteCharacterMailLabelOutput{}, fmt.Errorf("delete character mail label: %w", err)
		}
		return nil, DeleteCharacterMailLabelOutput{CharacterID: in.CharacterID, LabelID: in.LabelID}, nil
	}
}
