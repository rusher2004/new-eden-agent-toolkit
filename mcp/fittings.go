package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

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

var listCharacterFittings = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Fitting, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func listCharacterFittingsHandler(version, clientID string, store *config.Store, sdeStore *sde.Store) mcpsdk.ToolHandlerFor[ListCharacterFittingsInput, ListCharacterFittingsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterFittingsInput) (*mcpsdk.CallToolResult, ListCharacterFittingsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterFittingsOutput{}, err
		}
		if !found {
			return nil, ListCharacterFittingsOutput{}, fmt.Errorf("list character fittings for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		fittings, err := listCharacterFittings(ctx, clientID, in.CharacterID, sso.UserAgent(version))
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
		names := resolveTypeNames(ctx, sdeStore, sso.UserAgent(version), ids)
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

var createCharacterFitting = func(ctx context.Context, clientID string, characterID int32, userAgent string, description, name string, shipTypeID int64, items []FittingItemInput) (int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return 0, fmt.Errorf("build ESI client: %w", err)
	}
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

func createCharacterFittingHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[CreateCharacterFittingInput, CreateCharacterFittingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in CreateCharacterFittingInput) (*mcpsdk.CallToolResult, CreateCharacterFittingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, CreateCharacterFittingOutput{}, err
		}
		if !found {
			return nil, CreateCharacterFittingOutput{}, fmt.Errorf("create character fitting for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		fittingID, err := createCharacterFitting(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Description, in.Name, in.ShipTypeID, in.Items)
		if err != nil {
			return nil, CreateCharacterFittingOutput{}, fmt.Errorf("create character fitting: %w", err)
		}
		return nil, CreateCharacterFittingOutput{CharacterID: in.CharacterID, FittingID: fittingID}, nil
	}
}

// --- delete_character_fitting (private, write) ---

var deleteCharacterFitting = func(ctx context.Context, clientID string, characterID int32, userAgent string, fittingID int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
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

func deleteCharacterFittingHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[DeleteCharacterFittingInput, DeleteCharacterFittingOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in DeleteCharacterFittingInput) (*mcpsdk.CallToolResult, DeleteCharacterFittingOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, DeleteCharacterFittingOutput{}, err
		}
		if !found {
			return nil, DeleteCharacterFittingOutput{}, fmt.Errorf("delete character fitting for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := deleteCharacterFitting(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.FittingID); err != nil {
			return nil, DeleteCharacterFittingOutput{}, fmt.Errorf("delete character fitting: %w", err)
		}
		return nil, DeleteCharacterFittingOutput{CharacterID: in.CharacterID, FittingID: in.FittingID}, nil
	}
}
