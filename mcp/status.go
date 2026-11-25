package mcp

import (
	"context"
	"fmt"
	"time"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// statusESI is the ESI call get_status makes; it is public.
type statusESI interface {
	Status(ctx context.Context) (*esiapi.Status, error)
}

// --- get_status (public) ---

func getStatus(ctx context.Context, client statusESI) (int64, string, time.Time, bool, error) {
	status, err := client.Status(ctx)
	if err != nil {
		return 0, "", time.Time{}, false, err
	}
	return status.Players, status.ServerVersion, status.StartTime, status.Vip, nil
}

type GetStatusInput struct{}
type GetStatusOutput struct {
	Players       int64     `json:"players"`
	ServerVersion string    `json:"serverVersion"`
	StartTime     time.Time `json:"startTime"`
	Vip           bool      `json:"vip"`
}

// AddGetStatus registers get_status. open builds the ESI client; any client with the
// statusESI methods will do.
func AddGetStatus[C statusESI](s *mcpsdk.Server, open func() C) {
	mcpsdk.AddTool(s, &mcpsdk.Tool{
		Name:        "get_status",
		Description: "Get the current EVE Online server status: player count, version, start time, and VIP mode (public data, no character required).",
	}, getStatusHandler(open))
}

func getStatusHandler[C statusESI](open func() C) mcpsdk.ToolHandlerFor[GetStatusInput, GetStatusOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetStatusInput) (*mcpsdk.CallToolResult, GetStatusOutput, error) {
		players, serverVersion, startTime, vip, err := getStatus(ctx, open())
		if err != nil {
			return nil, GetStatusOutput{}, fmt.Errorf("get status: %w", err)
		}
		return nil, GetStatusOutput{Players: players, ServerVersion: serverVersion, StartTime: startTime, Vip: vip}, nil
	}
}
