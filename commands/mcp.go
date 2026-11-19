package commands

import (
	"context"
	"fmt"
	"log/slog"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

func mcpCommand(version, clientID string) *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "run the neat MCP server (stdio transport)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store, err := config.NewStore()
			if err != nil {
				return fmt.Errorf("resolve config store: %w", err)
			}

			sdePath, err := sde.DefaultPath()
			if err != nil {
				return fmt.Errorf("resolve sde cache path: %w", err)
			}
			// A failed Open (e.g. an unwritable ~/.neat/sde directory) must
			// not take down the other ~190 non-sde/ tools with it --
			// sdeStore stays nil, and every sde.Store method is nil-safe
			// (see Store's doc comment), so every Universe tool's existing
			// cache-miss path already falls through to ESI. A sync-time
			// failure (e.g. the sqlite file locked by another neat process,
			// or a failed dump download) is handled separately, inside
			// ensureSynced -- it's cached and logged once, not surfaced here.
			sdeStore, err := sde.Open(sdePath)
			if err != nil {
				slog.Default().WarnContext(ctx, "sde: failed to open cache, Universe tools will read through to ESI directly", "error", err)
				sdeStore = nil
			}
			defer func() {
				if err := sdeStore.Close(); err != nil {
					slog.Default().WarnContext(ctx, "sde: failed to close cache", "error", err)
				}
			}()

			server := mcp.NewServer(version, clientID, store, sdeStore)
			return server.Run(ctx, &mcpsdk.StdioTransport{})
		},
	}
}
