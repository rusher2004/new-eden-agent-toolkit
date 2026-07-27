package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- set_autopilot_waypoint (private, write) ---
//
// This changes state on the querying character's live EVE client (sets/queues an autopilot
// destination) -- verb-first name, no confirmation parameter, same convention as every other
// write tool in this project (e.g. respond_calendar_event).

var setAutopilotWaypoint = func(ctx context.Context, clientID string, characterID int32, userAgent string, destinationID int64, addToBeginning, clearOtherWaypoints bool) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.SetAutopilotWaypoint(ctx, destinationID, addToBeginning, clearOtherWaypoints)
}

type SetAutopilotWaypointInput struct {
	CharacterID         int32 `json:"characterId" jsonschema:"the character ID to set the waypoint for"`
	DestinationID       int64 `json:"destinationId" jsonschema:"the item/station/structure/system ID to set as an autopilot waypoint"`
	AddToBeginning      bool  `json:"addToBeginning" jsonschema:"whether to add this waypoint to the beginning of the autopilot route"`
	ClearOtherWaypoints bool  `json:"clearOtherWaypoints" jsonschema:"whether to clear all other waypoints before adding this one"`
}
type SetAutopilotWaypointOutput struct {
	CharacterID         int32 `json:"characterId"`
	DestinationID       int64 `json:"destinationId"`
	AddToBeginning      bool  `json:"addToBeginning"`
	ClearOtherWaypoints bool  `json:"clearOtherWaypoints"`
}

func setAutopilotWaypointHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[SetAutopilotWaypointInput, SetAutopilotWaypointOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in SetAutopilotWaypointInput) (*mcpsdk.CallToolResult, SetAutopilotWaypointOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, SetAutopilotWaypointOutput{}, err
		}
		if !found {
			return nil, SetAutopilotWaypointOutput{}, fmt.Errorf("set autopilot waypoint for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := setAutopilotWaypoint(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.DestinationID, in.AddToBeginning, in.ClearOtherWaypoints); err != nil {
			return nil, SetAutopilotWaypointOutput{}, fmt.Errorf("set autopilot waypoint: %w", err)
		}
		return nil, SetAutopilotWaypointOutput{
			CharacterID: in.CharacterID, DestinationID: in.DestinationID,
			AddToBeginning: in.AddToBeginning, ClearOtherWaypoints: in.ClearOtherWaypoints,
		}, nil
	}
}

// --- open_contract_window (private, write) ---

var openContractWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, contractID int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.OpenContractWindow(ctx, contractID)
}

type OpenContractWindowInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID whose client should open the window"`
	ContractID  int64 `json:"contractId" jsonschema:"the contract ID to open"`
}
type OpenContractWindowOutput struct {
	CharacterID int32 `json:"characterId"`
	ContractID  int64 `json:"contractId"`
}

func openContractWindowHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[OpenContractWindowInput, OpenContractWindowOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in OpenContractWindowInput) (*mcpsdk.CallToolResult, OpenContractWindowOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, OpenContractWindowOutput{}, err
		}
		if !found {
			return nil, OpenContractWindowOutput{}, fmt.Errorf("open contract window for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := openContractWindow(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ContractID); err != nil {
			return nil, OpenContractWindowOutput{}, fmt.Errorf("open contract window: %w", err)
		}
		return nil, OpenContractWindowOutput{CharacterID: in.CharacterID, ContractID: in.ContractID}, nil
	}
}

// --- open_information_window (private, write) ---

var openInformationWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, targetID int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.OpenInformationWindow(ctx, targetID)
}

type OpenInformationWindowInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID whose client should open the window"`
	TargetID    int64 `json:"targetId" jsonschema:"the character/corporation/alliance/item/system ID to show an info window for"`
}
type OpenInformationWindowOutput struct {
	CharacterID int32 `json:"characterId"`
	TargetID    int64 `json:"targetId"`
}

func openInformationWindowHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[OpenInformationWindowInput, OpenInformationWindowOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in OpenInformationWindowInput) (*mcpsdk.CallToolResult, OpenInformationWindowOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, OpenInformationWindowOutput{}, err
		}
		if !found {
			return nil, OpenInformationWindowOutput{}, fmt.Errorf("open information window for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := openInformationWindow(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.TargetID); err != nil {
			return nil, OpenInformationWindowOutput{}, fmt.Errorf("open information window: %w", err)
		}
		return nil, OpenInformationWindowOutput{CharacterID: in.CharacterID, TargetID: in.TargetID}, nil
	}
}

// --- open_market_details_window (private, write) ---

var openMarketDetailsWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, typeID int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.OpenMarketDetailsWindow(ctx, typeID)
}

type OpenMarketDetailsWindowInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID whose client should open the window"`
	TypeID      int64 `json:"typeId" jsonschema:"the item type ID to show market details for"`
}
type OpenMarketDetailsWindowOutput struct {
	CharacterID int32 `json:"characterId"`
	TypeID      int64 `json:"typeId"`
}

func openMarketDetailsWindowHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[OpenMarketDetailsWindowInput, OpenMarketDetailsWindowOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in OpenMarketDetailsWindowInput) (*mcpsdk.CallToolResult, OpenMarketDetailsWindowOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, OpenMarketDetailsWindowOutput{}, err
		}
		if !found {
			return nil, OpenMarketDetailsWindowOutput{}, fmt.Errorf("open market details window for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := openMarketDetailsWindow(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.TypeID); err != nil {
			return nil, OpenMarketDetailsWindowOutput{}, fmt.Errorf("open market details window: %w", err)
		}
		return nil, OpenMarketDetailsWindowOutput{CharacterID: in.CharacterID, TypeID: in.TypeID}, nil
	}
}

// --- open_new_mail_window (private, write) ---

var openNewMailWindow = func(ctx context.Context, clientID string, characterID int32, userAgent string, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.OpenNewMailWindow(ctx, recipients, subject, body, toCorpOrAllianceID, toMailingListID)
}

type OpenNewMailWindowInput struct {
	CharacterID        int32   `json:"characterId" jsonschema:"the character ID whose client should open the window"`
	Recipients         []int64 `json:"recipients" jsonschema:"character/corporation/alliance/mailing-list IDs to pre-fill as recipients"`
	Subject            string  `json:"subject" jsonschema:"the pre-filled mail subject"`
	Body               string  `json:"body" jsonschema:"the pre-filled mail body"`
	ToCorpOrAllianceID *int64  `json:"toCorpOrAllianceId,omitempty" jsonschema:"an extra corporation/alliance mailing-group recipient -- mutually exclusive with toMailingListId"`
	ToMailingListID    *int64  `json:"toMailingListId,omitempty" jsonschema:"an extra mailing-list recipient -- mutually exclusive with toCorpOrAllianceId"`
}
type OpenNewMailWindowOutput struct {
	CharacterID        int32   `json:"characterId"`
	Recipients         []int64 `json:"recipients"`
	Subject            string  `json:"subject"`
	Body               string  `json:"body"`
	ToCorpOrAllianceID *int64  `json:"toCorpOrAllianceId,omitempty"`
	ToMailingListID    *int64  `json:"toMailingListId,omitempty"`
}

func openNewMailWindowHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[OpenNewMailWindowInput, OpenNewMailWindowOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in OpenNewMailWindowInput) (*mcpsdk.CallToolResult, OpenNewMailWindowOutput, error) {
		if in.ToCorpOrAllianceID != nil && in.ToMailingListID != nil {
			return nil, OpenNewMailWindowOutput{}, fmt.Errorf("open new mail window: toCorpOrAllianceId and toMailingListId are mutually exclusive, set at most one")
		}
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, OpenNewMailWindowOutput{}, err
		}
		if !found {
			return nil, OpenNewMailWindowOutput{}, fmt.Errorf("open new mail window for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := openNewMailWindow(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.Recipients, in.Subject, in.Body, in.ToCorpOrAllianceID, in.ToMailingListID); err != nil {
			return nil, OpenNewMailWindowOutput{}, fmt.Errorf("open new mail window: %w", err)
		}
		return nil, OpenNewMailWindowOutput{
			CharacterID: in.CharacterID, Recipients: in.Recipients, Subject: in.Subject, Body: in.Body,
			ToCorpOrAllianceID: in.ToCorpOrAllianceID, ToMailingListID: in.ToMailingListID,
		}, nil
	}
}
