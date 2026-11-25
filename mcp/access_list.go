package mcp

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// accessListESI is the ESI calls the access list tools make.
type accessListESI interface {
	AccessLists(ctx context.Context) ([]esiapi.CharactersAccessListsListingAccesslist, error)
	AccessListDetail(ctx context.Context, accessListID int64) (*esiapi.CharactersAccessListsDetail, error)
}

func listAccessLists(ctx context.Context, client accessListESI) ([]int64, error) {
	lists, err := client.AccessLists(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(lists))
	for i, l := range lists {
		ids[i] = l.Id
	}
	return ids, nil
}

// ListAccessListsInput is list_access_lists's tool input.
type ListAccessListsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}

// ListAccessListsOutput is list_access_lists's tool result.
type ListAccessListsOutput struct {
	CharacterID   int32   `json:"characterId"`
	AccessListIDs []int64 `json:"accessListIds"`
}

// AddListAccessLists registers list_access_lists. open builds the ESI client; any client with the
// accessListESI methods will do.
func AddListAccessLists[C accessListESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "list_access_lists",
		Description: "List a logged-in EVE Online character's structure/fleet access list IDs.",
	}, listAccessListsHandler(open, store))
}

func listAccessListsHandler[C accessListESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[ListAccessListsInput, ListAccessListsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAccessListsInput) (*mcpsdk.CallToolResult, ListAccessListsOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, ListAccessListsOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, ListAccessListsOutput{}, fmt.Errorf("list access lists for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, ListAccessListsOutput{}, fmt.Errorf("list access lists: %w", err)
		}
		ids, err := listAccessLists(ctx, client)
		if err != nil {
			return nil, ListAccessListsOutput{}, fmt.Errorf("list access lists: %w", err)
		}
		return nil, ListAccessListsOutput{CharacterID: in.CharacterID, AccessListIDs: ids}, nil
	}
}

// AccessListEntry is one access-grant entry (a character, corporation, or alliance) within an
// access list's membership, mapped from ESI's three separate but identically-shaped entry types.
type AccessListEntry struct {
	EntityID int64  `json:"entityId"`
	Access   string `json:"access"`
}

// AccessListDetailResult is the raw mapped detail returned by getAccessListDetail, before the
// handler adds the characterID/accessListID it already knows from the request.
type AccessListDetailResult struct {
	Name          string
	Description   string
	AllowEveryone bool
	Characters    []AccessListEntry
	Corporations  []AccessListEntry
	Alliances     []AccessListEntry
}

func getAccessListDetail(ctx context.Context, client accessListESI, accessListID int64) (AccessListDetailResult, error) {
	detail, err := client.AccessListDetail(ctx, accessListID)
	if err != nil {
		return AccessListDetailResult{}, err
	}

	characters := make([]AccessListEntry, len(detail.Membership.Characters))
	for i, e := range detail.Membership.Characters {
		characters[i] = AccessListEntry{EntityID: e.CharacterId, Access: e.Access}
	}
	corporations := make([]AccessListEntry, len(detail.Membership.Corporations))
	for i, e := range detail.Membership.Corporations {
		corporations[i] = AccessListEntry{EntityID: e.CorporationId, Access: e.Access}
	}
	alliances := make([]AccessListEntry, len(detail.Membership.Alliances))
	for i, e := range detail.Membership.Alliances {
		alliances[i] = AccessListEntry{EntityID: e.AllianceId, Access: e.Access}
	}

	return AccessListDetailResult{
		Name:          detail.Name,
		Description:   detail.Description,
		AllowEveryone: detail.Membership.AllowEveryone,
		Characters:    characters,
		Corporations:  corporations,
		Alliances:     alliances,
	}, nil
}

// GetAccessListInput is get_access_list's tool input.
type GetAccessListInput struct {
	CharacterID  int32 `json:"characterId" jsonschema:"the character ID to check"`
	AccessListID int64 `json:"accessListId" jsonschema:"the access list ID, from list_access_lists"`
}

// GetAccessListOutput is get_access_list's tool result.
type GetAccessListOutput struct {
	CharacterID   int32             `json:"characterId"`
	AccessListID  int64             `json:"accessListId"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	AllowEveryone bool              `json:"allowEveryone"`
	Characters    []AccessListEntry `json:"characters"`
	Corporations  []AccessListEntry `json:"corporations"`
	Alliances     []AccessListEntry `json:"alliances"`
}

// AddGetAccessList registers get_access_list. open builds the ESI client; any client with the
// accessListESI methods will do.
func AddGetAccessList[C accessListESI](s *mcpsdk.Server, open func(ctx context.Context, characterID int32) (C, error), store configLoader) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_access_list",
		Description: "Get the name, description, and membership detail of one of a character's access lists.",
	}, getAccessListHandler(open, store))
}

func getAccessListHandler[C accessListESI](open func(ctx context.Context, characterID int32) (C, error), store configLoader) mcpsdk.ToolHandlerFor[GetAccessListInput, GetAccessListOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetAccessListInput) (*mcpsdk.CallToolResult, GetAccessListOutput, error) {
		cfg, err := store.Load()
		if err != nil {
			return nil, GetAccessListOutput{}, fmt.Errorf("load config: %w", err)
		}
		found := false
		for _, c := range cfg.Characters {
			if c.ID == in.CharacterID {
				found = true
				break
			}
		}
		if !found {
			return nil, GetAccessListOutput{}, fmt.Errorf("get access list for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}

		client, err := open(ctx, in.CharacterID)
		if err != nil {
			return nil, GetAccessListOutput{}, fmt.Errorf("get access list: %w", err)
		}
		result, err := getAccessListDetail(ctx, client, in.AccessListID)
		if err != nil {
			return nil, GetAccessListOutput{}, fmt.Errorf("get access list detail: %w", err)
		}
		return nil, GetAccessListOutput{
			CharacterID:   in.CharacterID,
			AccessListID:  in.AccessListID,
			Name:          result.Name,
			Description:   result.Description,
			AllowEveryone: result.AllowEveryone,
			Characters:    result.Characters,
			Corporations:  result.Corporations,
			Alliances:     result.Alliances,
		}, nil
	}
}
