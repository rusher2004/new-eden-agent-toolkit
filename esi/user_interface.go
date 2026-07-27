// Package esi's user_interface.go covers all 5 of "User Interface"'s endpoints -- these are all
// writes that control the querying character's own live EVE client (move the autopilot, open a
// UI window). All 5 return only an error -- ESI gives a 204 with no response body for every one
// of them (verified individually from api_user_interface.go), same write-return-shape as
// esi/calendar.go's RespondToCalendarEvent. All 5 are "my own character" -- no separate target-
// character parameter, since ESI only ever acts on the authenticated character's own client.
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

func (c *Client) SetAutopilotWaypoint(ctx context.Context, destinationID int64, addToBeginning, clearOtherWaypoints bool) error {
	resp, err := c.api.UserInterfaceAPI.PostUiAutopilotWaypoint(ctx).
		DestinationId(destinationID).AddToBeginning(addToBeginning).ClearOtherWaypoints(clearOtherWaypoints).
		Execute()
	if err != nil {
		return fmt.Errorf("set autopilot waypoint: %w", describeError(err, resp))
	}
	return nil
}

func (c *Client) OpenContractWindow(ctx context.Context, contractID int64) error {
	resp, err := c.api.UserInterfaceAPI.PostUiOpenwindowContract(ctx).ContractId(contractID).Execute()
	if err != nil {
		return fmt.Errorf("open contract window: %w", describeError(err, resp))
	}
	return nil
}

func (c *Client) OpenInformationWindow(ctx context.Context, targetID int64) error {
	resp, err := c.api.UserInterfaceAPI.PostUiOpenwindowInformation(ctx).TargetId(targetID).Execute()
	if err != nil {
		return fmt.Errorf("open information window: %w", describeError(err, resp))
	}
	return nil
}

func (c *Client) OpenMarketDetailsWindow(ctx context.Context, typeID int64) error {
	resp, err := c.api.UserInterfaceAPI.PostUiOpenwindowMarketdetails(ctx).TypeId(typeID).Execute()
	if err != nil {
		return fmt.Errorf("open market details window: %w", describeError(err, resp))
	}
	return nil
}

func (c *Client) OpenNewMailWindow(ctx context.Context, recipients []int64, subject, body string, toCorpOrAllianceID, toMailingListID *int64) error {
	req := esiapi.PostUiOpenwindowNewmailRequest{Body: body, Recipients: recipients, Subject: subject}
	if toCorpOrAllianceID != nil {
		req.ToCorpOrAllianceId = toCorpOrAllianceID
	}
	if toMailingListID != nil {
		req.ToMailingListId = toMailingListID
	}
	resp, err := c.api.UserInterfaceAPI.PostUiOpenwindowNewmail(ctx).PostUiOpenwindowNewmailRequest(req).Execute()
	if err != nil {
		return fmt.Errorf("open new mail window: %w", describeError(err, resp))
	}
	return nil
}
