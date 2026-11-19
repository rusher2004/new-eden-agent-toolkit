package mcp

import (
	"context"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// --- get_status (public) ---

var getStatus = func(ctx context.Context, userAgent string) (int64, string, time.Time, bool, error) {
	client := esi.NewPublicClient(userAgent)
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

func getStatusHandler(version string) mcpsdk.ToolHandlerFor[GetStatusInput, GetStatusOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in GetStatusInput) (*mcpsdk.CallToolResult, GetStatusOutput, error) {
		players, serverVersion, startTime, vip, err := getStatus(ctx, sso.UserAgent(version))
		if err != nil {
			return nil, GetStatusOutput{}, fmt.Errorf("get status: %w", err)
		}
		return nil, GetStatusOutput{Players: players, ServerVersion: serverVersion, StartTime: startTime, Vip: vip}, nil
	}
}
