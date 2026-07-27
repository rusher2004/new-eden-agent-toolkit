package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// listAccessLists is overridable in tests so no real network/auth runs.
var listAccessLists = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
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

func listAccessListsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListAccessListsInput, ListAccessListsOutput] {
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

		ids, err := listAccessLists(ctx, clientID, in.CharacterID, sso.UserAgent(version))
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

// getAccessListDetail is overridable in tests so no real network/auth runs.
var getAccessListDetail = func(ctx context.Context, clientID string, characterID int32, userAgent string, accessListID int64) (AccessListDetailResult, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return AccessListDetailResult{}, fmt.Errorf("build ESI client: %w", err)
	}
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

func getAccessListHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[GetAccessListInput, GetAccessListOutput] {
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

		result, err := getAccessListDetail(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.AccessListID)
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
