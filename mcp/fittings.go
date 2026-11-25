package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// fittingsESI is the ESI calls the Fittings tools make.
type fittingsESI interface {
	CharacterFittings(ctx context.Context) ([]esiapi.CharactersCharacterIdFittingsGetInner, error)
	CreateCharacterFitting(ctx context.Context, description string, items []esiapi.PostCharactersCharacterIdFittingsRequestItemsInner, name string, shipTypeID int64) (int64, error)
	DeleteCharacterFitting(ctx context.Context, fittingID int64) error
}

// --- list_character_fittings (private) ---

type FittingItem struct {
	Flag     string `json:"flag"`
	Quantity int64  `json:"quantity"`
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"typeName,omitempty"`
}

type Fitting struct {
	FittingID    int64         `json:"fittingId"`
	ShipTypeID   int64         `json:"shipTypeId"`
	ShipTypeName string        `json:"shipTypeName,omitempty"`
	Description  string        `json:"description"`
	Name         string        `json:"name"`
	Items        []FittingItem `json:"items"`
}

func listCharacterFittings(ctx context.Context, client fittingsESI) ([]Fitting, error) {
	fittings, err := client.CharacterFittings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Fitting, len(fittings))
	for i, f := range fittings {
		items := make([]FittingItem, len(f.Items))
		for j, it := range f.Items {
			items[j] = FittingItem{Flag: it.Flag, Quantity: it.Quantity, TypeID: it.TypeId}
		}
		out[i] = Fitting{FittingID: f.FittingId, ShipTypeID: f.ShipTypeId, Description: f.Description, Name: f.Name, Items: items}
	}
	return out, nil
}

type ListCharacterFittingsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterFittingsOutput struct {
	CharacterID int32     `json:"characterId"`
	Fittings    []Fitting `json:"fittings"`
}

// AddListCharacterFittings registers list_character_fittings. open builds the character's ESI
// client; any client with the fittingsESI methods will do.
func AddListCharacterFittings[C fittingsESI](s *mcpsdk.Server, userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_character_fittings",
		Description: "List a logged-in EVE Online character's saved ship fittings.",
	}, listCharacterFittingsHandler(userAgent, open, store, sdeStore))
}

func listCharacterFittingsHandler[C fittingsESI](userAgent string, open func(ctx context.Context, characterID int32) (C, error), store configLoader, sdeStore typeNameCache) mcpsdk.ToolHandlerFor[ListCharacterFittingsInput, ListCharacterFittingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterFittingsInput) (*mcpsdk.CallToolResult, ListCharacterFittingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterFittingsOutput{}, err
		}
		if !found {
			return nil, ListCharacterFittingsOutput{}, fmt.Errorf("list character fittings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListCharacterFittingsOutput{}, fmt.Errorf("list character fittings: %w", err)
		}
		fittings, err := listCharacterFittings(ctx, client)
		if err != nil {
			return nil, ListCharacterFittingsOutput{}, fmt.Errorf("list character fittings: %w", err)
		}
		// Collect type IDs from both levels for enrichment
		ids := make([]int64, 0, len(fittings))
		for _, f := range fittings {
			ids = append(ids, f.ShipTypeID)
			for _, it := range f.Items {
				ids = append(ids, it.TypeID)
			}
		}
		names := resolveTypeNames(ctx, sdeStore, userAgent, ids)
		// Enrich both outer ship type and nested item types
		for i := range fittings {
			fittings[i].ShipTypeName = names[fittings[i].ShipTypeID]
			for j := range fittings[i].Items {
				fittings[i].Items[j].TypeName = names[fittings[i].Items[j].TypeID]
			}
		}
		return nil, ListCharacterFittingsOutput{CharacterID: in.CharacterID, Fittings: fittings}, nil
	}
}

// --- create_character_fitting (private, write) ---

type FittingItemInput struct {
	Flag     string `json:"flag" jsonschema:"fitting location for the item, e.g. HiSlot0, LoSlot0, Cargo"`
	Quantity int64  `json:"quantity" jsonschema:"how many of this item to fit"`
	TypeID   int64  `json:"typeId" jsonschema:"the item's type ID"`
}

func createCharacterFitting(ctx context.Context, client fittingsESI, description, name string, shipTypeID int64, items []FittingItemInput) (int64, error) {
	esiItems := make([]esiapi.PostCharactersCharacterIdFittingsRequestItemsInner, len(items))
	for i, it := range items {
		esiItems[i] = esiapi.PostCharactersCharacterIdFittingsRequestItemsInner{Flag: it.Flag, Quantity: it.Quantity, TypeId: it.TypeID}
	}
	return client.CreateCharacterFitting(ctx, description, esiItems, name, shipTypeID)
}

type CreateCharacterFittingInput struct {
	CharacterID int32              `json:"characterId" jsonschema:"the character ID to check"`
	Description string             `json:"description" jsonschema:"the fitting's description"`
	Name        string             `json:"name" jsonschema:"the fitting's name"`
	ShipTypeID  int64              `json:"shipTypeId" jsonschema:"the ship type ID this fitting is for"`
	Items       []FittingItemInput `json:"items" jsonschema:"the modules/items in this fitting"`
}
type CreateCharacterFittingOutput struct {
	CharacterID int32 `json:"characterId"`
	FittingID   int64 `json:"fittingId"`
}

// AddCreateCharacterFitting registers create_character_fitting. open builds the character's ESI
// client; any client with the fittingsESI methods will do.
func AddCreateCharacterFitting[C fittingsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "create_character_fitting",
		Description: "Save a new ship fitting for a logged-in character — this creates a new fitting on EVE's server.",
	}, createCharacterFittingHandler(open, store))
}

func createCharacterFittingHandler[C fittingsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[CreateCharacterFittingInput, CreateCharacterFittingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateCharacterFittingInput) (*mcpsdk.CallToolResult, CreateCharacterFittingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, CreateCharacterFittingOutput{}, err
		}
		if !found {
			return nil, CreateCharacterFittingOutput{}, fmt.Errorf("create character fitting for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, CreateCharacterFittingOutput{}, fmt.Errorf("create character fitting: %w", err)
		}
		fittingID, err := createCharacterFitting(ctx, client, in.Description, in.Name, in.ShipTypeID, in.Items)
		if err != nil {
			return nil, CreateCharacterFittingOutput{}, fmt.Errorf("create character fitting: %w", err)
		}
		return nil, CreateCharacterFittingOutput{CharacterID: in.CharacterID, FittingID: fittingID}, nil
	}
}

// --- delete_character_fitting (private, write) ---

func deleteCharacterFitting(ctx context.Context, client fittingsESI, fittingID int64) error {
	return client.DeleteCharacterFitting(ctx, fittingID)
}

type DeleteCharacterFittingInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	FittingID   int64 `json:"fittingId" jsonschema:"the fitting ID to delete"`
}
type DeleteCharacterFittingOutput struct {
	CharacterID int32 `json:"characterId"`
	FittingID   int64 `json:"fittingId"`
}

// AddDeleteCharacterFitting registers delete_character_fitting. open builds the character's ESI
// client; any client with the fittingsESI methods will do.
func AddDeleteCharacterFitting[C fittingsESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "delete_character_fitting",
		Description: "Delete one of a logged-in character's saved ship fittings — this deletes the fitting on EVE's server.",
	}, deleteCharacterFittingHandler(open, store))
}

func deleteCharacterFittingHandler[C fittingsESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[DeleteCharacterFittingInput, DeleteCharacterFittingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteCharacterFittingInput) (*mcpsdk.CallToolResult, DeleteCharacterFittingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, DeleteCharacterFittingOutput{}, err
		}
		if !found {
			return nil, DeleteCharacterFittingOutput{}, fmt.Errorf("delete character fitting for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, DeleteCharacterFittingOutput{}, fmt.Errorf("delete character fitting: %w", err)
		}
		if err := deleteCharacterFitting(ctx, client, in.FittingID); err != nil {
			return nil, DeleteCharacterFittingOutput{}, fmt.Errorf("delete character fitting: %w", err)
		}
		return nil, DeleteCharacterFittingOutput{CharacterID: in.CharacterID, FittingID: in.FittingID}, nil
	}
}
