package mcp

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Contact is one contact entry, shared by character/corporation/alliance listings.
type Contact struct {
	ContactID   int64   `json:"contactId"`
	ContactType string  `json:"contactType"`
	Standing    float64 `json:"standing"`
	IsBlocked   *bool   `json:"isBlocked,omitempty"`
	IsWatched   *bool   `json:"isWatched,omitempty"`
	LabelIDs    []int64 `json:"labelIds,omitempty"`
}

// ContactLabel is one contact label, shared by character/corporation/alliance listings.
type ContactLabel struct {
	LabelID   int64  `json:"labelId"`
	LabelName string `json:"labelName"`
}

// --- list_character_contacts ---

// characterContactsESI is the ESI call list_character_contacts makes; the
// planner family reads standings through it too.
type characterContactsESI interface {
	CharacterContacts(ctx context.Context, page int32) ([]esiapi.CharactersCharacterIdContactsGetInner, int32, error)
}

func listCharacterContacts(ctx context.Context, client characterContactsESI, page int32) ([]Contact, int32, error) {
	contacts, totalPages, err := client.CharacterContacts(ctx, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Contact, len(contacts))
	for i, c := range contacts {
		out[i] = Contact{ContactID: c.ContactId, ContactType: c.ContactType, Standing: c.Standing, IsBlocked: c.IsBlocked, IsWatched: c.IsWatched, LabelIDs: c.LabelIds}
	}
	return out, totalPages, nil
}

type ListCharacterContactsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCharacterContactsOutput struct {
	CharacterID int32     `json:"characterId"`
	Page        int32     `json:"page"`
	TotalPages  int32     `json:"totalPages"`
	Contacts    []Contact `json:"contacts"`
}

// AddListCharacterContacts registers list_character_contacts. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterContacts[C characterContactsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_contacts",
		Description: "List a logged-in EVE Online character's contacts.",
	}, listCharacterContactsHandler(open, store))
}

func listCharacterContactsHandler[C characterContactsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterContactsInput, ListCharacterContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContactsInput) (*mcpsdk.CallToolResult, ListCharacterContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContactsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContactsOutput{}, fmt.Errorf("list character contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContactsOutput{}, fmt.Errorf("list character contacts: %w", err)
		}
		contacts, totalPages, err := listCharacterContacts(ctx, client, in.Page)
		if err != nil {
			return nil, ListCharacterContactsOutput{}, fmt.Errorf("list character contacts: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCharacterContactsOutput{CharacterID: in.CharacterID, Page: page, TotalPages: totalPages, Contacts: contacts}, nil
	}
}

// --- list_character_contact_labels ---

// characterContactLabelsESI is the ESI call list_character_contact_labels makes.
type characterContactLabelsESI interface {
	CharacterContactLabels(ctx context.Context) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error)
}

func listCharacterContactLabels(ctx context.Context, client characterContactLabelsESI) ([]ContactLabel, error) {
	labels, err := client.CharacterContactLabels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ContactLabel, len(labels))
	for i, l := range labels {
		out[i] = ContactLabel{LabelID: l.LabelId, LabelName: l.LabelName}
	}
	return out, nil
}

type ListCharacterContactLabelsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterContactLabelsOutput struct {
	CharacterID int32          `json:"characterId"`
	Labels      []ContactLabel `json:"labels"`
}

// AddListCharacterContactLabels registers list_character_contact_labels. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCharacterContactLabels[C characterContactLabelsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_contact_labels",
		Description: "List a logged-in EVE Online character's contact labels.",
	}, listCharacterContactLabelsHandler(open, store))
}

func listCharacterContactLabelsHandler[C characterContactLabelsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCharacterContactLabelsInput, ListCharacterContactLabelsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContactLabelsInput) (*mcpsdk.CallToolResult, ListCharacterContactLabelsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContactLabelsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContactLabelsOutput{}, fmt.Errorf("list character contact labels for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContactLabelsOutput{}, fmt.Errorf("list character contact labels: %w", err)
		}
		labels, err := listCharacterContactLabels(ctx, client)
		if err != nil {
			return nil, ListCharacterContactLabelsOutput{}, fmt.Errorf("list character contact labels: %w", err)
		}
		return nil, ListCharacterContactLabelsOutput{CharacterID: in.CharacterID, Labels: labels}, nil
	}
}

// --- list_corporation_contacts ---

// corporationContactsESI is the ESI call list_corporation_contacts makes;
// the planner family reads standings through it too.
type corporationContactsESI interface {
	CorporationContacts(ctx context.Context, corporationID int64, page int32) ([]esiapi.CorporationsCorporationIdContactsGetInner, int32, error)
}

func listCorporationContacts(ctx context.Context, client corporationContactsESI, corporationID int64, page int32) ([]Contact, int32, error) {
	contacts, totalPages, err := client.CorporationContacts(ctx, corporationID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Contact, len(contacts))
	for i, c := range contacts {
		out[i] = Contact{ContactID: c.ContactId, ContactType: c.ContactType, Standing: c.Standing, IsWatched: c.IsWatched, LabelIDs: c.LabelIds}
	}
	return out, totalPages, nil
}

type ListCorporationContactsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
	Page          int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListCorporationContactsOutput struct {
	CharacterID   int32     `json:"characterId"`
	CorporationID int64     `json:"corporationId"`
	Page          int32     `json:"page"`
	TotalPages    int32     `json:"totalPages"`
	Contacts      []Contact `json:"contacts"`
}

// AddListCorporationContacts registers list_corporation_contacts. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCorporationContacts[C corporationContactsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_contacts",
		Description: "List a corporation's contacts, using a logged-in character's roles to authorize.",
	}, listCorporationContactsHandler(open, store))
}

func listCorporationContactsHandler[C corporationContactsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCorporationContactsInput, ListCorporationContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContactsInput) (*mcpsdk.CallToolResult, ListCorporationContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContactsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContactsOutput{}, fmt.Errorf("list corporation contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContactsOutput{}, fmt.Errorf("list corporation contacts: %w", err)
		}
		contacts, totalPages, err := listCorporationContacts(ctx, client, in.CorporationID, in.Page)
		if err != nil {
			return nil, ListCorporationContactsOutput{}, fmt.Errorf("list corporation contacts: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListCorporationContactsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Page: page, TotalPages: totalPages, Contacts: contacts}, nil
	}
}

// --- list_corporation_contact_labels ---

// corporationContactLabelsESI is the ESI call list_corporation_contact_labels makes.
type corporationContactLabelsESI interface {
	CorporationContactLabels(ctx context.Context, corporationID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error)
}

func listCorporationContactLabels(ctx context.Context, client corporationContactLabelsESI, corporationID int64) ([]ContactLabel, error) {
	labels, err := client.CorporationContactLabels(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]ContactLabel, len(labels))
	for i, l := range labels {
		out[i] = ContactLabel{LabelID: l.LabelId, LabelName: l.LabelName}
	}
	return out, nil
}

type ListCorporationContactLabelsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationContactLabelsOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	Labels        []ContactLabel `json:"labels"`
}

// AddListCorporationContactLabels registers list_corporation_contact_labels. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListCorporationContactLabels[C corporationContactLabelsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_corporation_contact_labels",
		Description: "List a corporation's contact labels, using a logged-in character's roles to authorize.",
	}, listCorporationContactLabelsHandler(open, store))
}

func listCorporationContactLabelsHandler[C corporationContactLabelsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListCorporationContactLabelsInput, ListCorporationContactLabelsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContactLabelsInput) (*mcpsdk.CallToolResult, ListCorporationContactLabelsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContactLabelsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContactLabelsOutput{}, fmt.Errorf("list corporation contact labels for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContactLabelsOutput{}, fmt.Errorf("list corporation contact labels: %w", err)
		}
		labels, err := listCorporationContactLabels(ctx, client, in.CorporationID)
		if err != nil {
			return nil, ListCorporationContactLabelsOutput{}, fmt.Errorf("list corporation contact labels: %w", err)
		}
		return nil, ListCorporationContactLabelsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Labels: labels}, nil
	}
}

// --- list_alliance_contacts ---

// allianceContactsESI is the ESI call list_alliance_contacts makes; the
// planner family reads standings through it too.
type allianceContactsESI interface {
	AllianceContacts(ctx context.Context, allianceID int64, page int32) ([]esiapi.AlliancesAllianceIdContactsGetInner, int32, error)
}

func listAllianceContacts(ctx context.Context, client allianceContactsESI, allianceID int64, page int32) ([]Contact, int32, error) {
	contacts, totalPages, err := client.AllianceContacts(ctx, allianceID, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Contact, len(contacts))
	for i, c := range contacts {
		out[i] = Contact{ContactID: c.ContactId, ContactType: c.ContactType, Standing: c.Standing, LabelIDs: c.LabelIds}
	}
	return out, totalPages, nil
}

type ListAllianceContactsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	AllianceID  int64 `json:"allianceId" jsonschema:"the alliance ID to look up"`
	Page        int32 `json:"page,omitempty" jsonschema:"page number, 1-indexed; omit for page 1"`
}
type ListAllianceContactsOutput struct {
	CharacterID int32     `json:"characterId"`
	AllianceID  int64     `json:"allianceId"`
	Page        int32     `json:"page"`
	TotalPages  int32     `json:"totalPages"`
	Contacts    []Contact `json:"contacts"`
}

// AddListAllianceContacts registers list_alliance_contacts. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListAllianceContacts[C allianceContactsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_alliance_contacts",
		Description: "List an alliance's contacts, using a logged-in character's roles to authorize.",
	}, listAllianceContactsHandler(open, store))
}

func listAllianceContactsHandler[C allianceContactsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListAllianceContactsInput, ListAllianceContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAllianceContactsInput) (*mcpsdk.CallToolResult, ListAllianceContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListAllianceContactsOutput{}, err
		}
		if !found {
			return nil, ListAllianceContactsOutput{}, fmt.Errorf("list alliance contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListAllianceContactsOutput{}, fmt.Errorf("list alliance contacts: %w", err)
		}
		contacts, totalPages, err := listAllianceContacts(ctx, client, in.AllianceID, in.Page)
		if err != nil {
			return nil, ListAllianceContactsOutput{}, fmt.Errorf("list alliance contacts: %w", err)
		}
		page := in.Page
		if page <= 0 {
			page = 1
		}
		return nil, ListAllianceContactsOutput{CharacterID: in.CharacterID, AllianceID: in.AllianceID, Page: page, TotalPages: totalPages, Contacts: contacts}, nil
	}
}

// --- list_alliance_contact_labels ---

// allianceContactLabelsESI is the ESI call list_alliance_contact_labels makes.
type allianceContactLabelsESI interface {
	AllianceContactLabels(ctx context.Context, allianceID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error)
}

func listAllianceContactLabels(ctx context.Context, client allianceContactLabelsESI, allianceID int64) ([]ContactLabel, error) {
	labels, err := client.AllianceContactLabels(ctx, allianceID)
	if err != nil {
		return nil, err
	}
	out := make([]ContactLabel, len(labels))
	for i, l := range labels {
		out[i] = ContactLabel{LabelID: l.LabelId, LabelName: l.LabelName}
	}
	return out, nil
}

type ListAllianceContactLabelsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	AllianceID  int64 `json:"allianceId" jsonschema:"the alliance ID to look up"`
}
type ListAllianceContactLabelsOutput struct {
	CharacterID int32          `json:"characterId"`
	AllianceID  int64          `json:"allianceId"`
	Labels      []ContactLabel `json:"labels"`
}

// AddListAllianceContactLabels registers list_alliance_contact_labels. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddListAllianceContactLabels[C allianceContactLabelsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_alliance_contact_labels",
		Description: "List an alliance's contact labels, using a logged-in character's roles to authorize.",
	}, listAllianceContactLabelsHandler(open, store))
}

func listAllianceContactLabelsHandler[C allianceContactLabelsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListAllianceContactLabelsInput, ListAllianceContactLabelsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAllianceContactLabelsInput) (*mcpsdk.CallToolResult, ListAllianceContactLabelsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListAllianceContactLabelsOutput{}, err
		}
		if !found {
			return nil, ListAllianceContactLabelsOutput{}, fmt.Errorf("list alliance contact labels for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListAllianceContactLabelsOutput{}, fmt.Errorf("list alliance contact labels: %w", err)
		}
		labels, err := listAllianceContactLabels(ctx, client, in.AllianceID)
		if err != nil {
			return nil, ListAllianceContactLabelsOutput{}, fmt.Errorf("list alliance contact labels: %w", err)
		}
		return nil, ListAllianceContactLabelsOutput{CharacterID: in.CharacterID, AllianceID: in.AllianceID, Labels: labels}, nil
	}
}

// --- add_character_contacts (write) ---

// addCharacterContactsESI is the ESI call add_character_contacts (a write tool: it changes state on EVE's server) makes.
type addCharacterContactsESI interface {
	AddCharacterContacts(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error)
}

func addCharacterContacts(ctx context.Context, client addCharacterContactsESI, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error) {
	return client.AddCharacterContacts(ctx, contactIDs, standing, labelIDs, watched)
}

type AddCharacterContactsInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	ContactIDs  []int64 `json:"contactIds" jsonschema:"character/corporation/alliance IDs to add as contacts"`
	Standing    float64 `json:"standing" jsonschema:"standing to assign, from -10.0 to 10.0"`
	LabelIDs    []int64 `json:"labelIds,omitempty" jsonschema:"label IDs to apply, from list_character_contact_labels"`
	Watched     bool    `json:"watched,omitempty" jsonschema:"whether to watch these contacts"`
}
type AddCharacterContactsOutput struct {
	CharacterID     int32   `json:"characterId"`
	AddedContactIDs []int64 `json:"addedContactIds"`
}

// AddAddCharacterContacts registers add_character_contacts. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddAddCharacterContacts[C addCharacterContactsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "add_character_contacts",
		Description: "Add contacts to a character's contact list with a given standing — this creates new contacts on EVE's server.",
	}, addCharacterContactsHandler(open, store))
}

func addCharacterContactsHandler[C addCharacterContactsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[AddCharacterContactsInput, AddCharacterContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in AddCharacterContactsInput) (*mcpsdk.CallToolResult, AddCharacterContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, AddCharacterContactsOutput{}, err
		}
		if !found {
			return nil, AddCharacterContactsOutput{}, fmt.Errorf("add character contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, AddCharacterContactsOutput{}, fmt.Errorf("add character contacts: %w", err)
		}
		added, err := addCharacterContacts(ctx, client, in.ContactIDs, in.Standing, in.LabelIDs, in.Watched)
		if err != nil {
			return nil, AddCharacterContactsOutput{}, fmt.Errorf("add character contacts: %w", err)
		}
		return nil, AddCharacterContactsOutput{CharacterID: in.CharacterID, AddedContactIDs: added}, nil
	}
}

// --- update_character_contacts (write) ---

// updateCharacterContactsESI is the ESI call update_character_contacts (a write tool: it changes state on EVE's server) makes.
type updateCharacterContactsESI interface {
	UpdateCharacterContacts(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error
}

func updateCharacterContacts(ctx context.Context, client updateCharacterContactsESI, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error {
	return client.UpdateCharacterContacts(ctx, contactIDs, standing, labelIDs, watched)
}

type UpdateCharacterContactsInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	ContactIDs  []int64 `json:"contactIds" jsonschema:"contact IDs to update"`
	Standing    float64 `json:"standing" jsonschema:"new standing to assign, from -10.0 to 10.0"`
	LabelIDs    []int64 `json:"labelIds,omitempty" jsonschema:"label IDs to apply"`
	Watched     bool    `json:"watched,omitempty" jsonschema:"whether to watch these contacts"`
}
type UpdateCharacterContactsOutput struct {
	CharacterID int32   `json:"characterId"`
	ContactIDs  []int64 `json:"contactIds"`
}

// AddUpdateCharacterContacts registers update_character_contacts. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddUpdateCharacterContacts[C updateCharacterContactsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "update_character_contacts",
		Description: "Update the standing/label/watched status of existing contacts — this changes state on EVE's server.",
	}, updateCharacterContactsHandler(open, store))
}

func updateCharacterContactsHandler[C updateCharacterContactsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[UpdateCharacterContactsInput, UpdateCharacterContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in UpdateCharacterContactsInput) (*mcpsdk.CallToolResult, UpdateCharacterContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, UpdateCharacterContactsOutput{}, err
		}
		if !found {
			return nil, UpdateCharacterContactsOutput{}, fmt.Errorf("update character contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, UpdateCharacterContactsOutput{}, fmt.Errorf("update character contacts: %w", err)
		}
		if err := updateCharacterContacts(ctx, client, in.ContactIDs, in.Standing, in.LabelIDs, in.Watched); err != nil {
			return nil, UpdateCharacterContactsOutput{}, fmt.Errorf("update character contacts: %w", err)
		}
		return nil, UpdateCharacterContactsOutput{CharacterID: in.CharacterID, ContactIDs: in.ContactIDs}, nil
	}
}

// --- remove_character_contacts (write) ---

// removeCharacterContactsESI is the ESI call remove_character_contacts (a write tool: it changes state on EVE's server) makes.
type removeCharacterContactsESI interface {
	RemoveCharacterContacts(ctx context.Context, contactIDs []int64) error
}

func removeCharacterContacts(ctx context.Context, client removeCharacterContactsESI, contactIDs []int64) error {
	return client.RemoveCharacterContacts(ctx, contactIDs)
}

type RemoveCharacterContactsInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	ContactIDs  []int64 `json:"contactIds" jsonschema:"contact IDs to remove"`
}
type RemoveCharacterContactsOutput struct {
	CharacterID       int32   `json:"characterId"`
	RemovedContactIDs []int64 `json:"removedContactIds"`
}

// AddRemoveCharacterContacts registers remove_character_contacts. open builds the ESI client; any client with
// the methods this tool calls will do.
func AddRemoveCharacterContacts[C removeCharacterContactsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "remove_character_contacts",
		Description: "Remove contacts from a character's contact list — this deletes contacts on EVE's server.",
	}, removeCharacterContactsHandler(open, store))
}

func removeCharacterContactsHandler[C removeCharacterContactsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[RemoveCharacterContactsInput, RemoveCharacterContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RemoveCharacterContactsInput) (*mcpsdk.CallToolResult, RemoveCharacterContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, RemoveCharacterContactsOutput{}, err
		}
		if !found {
			return nil, RemoveCharacterContactsOutput{}, fmt.Errorf("remove character contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, RemoveCharacterContactsOutput{}, fmt.Errorf("remove character contacts: %w", err)
		}
		if err := removeCharacterContacts(ctx, client, in.ContactIDs); err != nil {
			return nil, RemoveCharacterContactsOutput{}, fmt.Errorf("remove character contacts: %w", err)
		}
		return nil, RemoveCharacterContactsOutput{CharacterID: in.CharacterID, RemovedContactIDs: in.ContactIDs}, nil
	}
}
