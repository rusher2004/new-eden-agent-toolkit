package commands

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/analytics"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

func mcpCommand(version, clientID, aptabaseAppKey string) *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "run the neat MCP server (stdio transport)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store, err := config.NewStore()
			if err != nil {
				return fmt.Errorf("resolve config store: %w", err)
			}
			go sendStartPing(context.WithoutCancel(ctx), store, aptabaseAppKey, version)

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

// sendStartPing sends neat's one anonymous "app_started" event (see
// analytics/), unless the user turned analytics off or the build carries no
// app key. It runs in the background and never fails the server: every
// problem is logged at debug level and dropped.
func sendStartPing(ctx context.Context, store *config.Store, appKey, version string) {
	if appKey == "" {
		return
	}
	logger := slog.Default()
	cfg, err := store.Load()
	if err != nil {
		logger.DebugContext(ctx, "analytics: skipped, config unreadable", "error", err)
		return
	}
	if cfg.AnalyticsDisabled {
		return
	}
	client, err := analytics.New(appKey, version, sso.UserAgent(version))
	if err != nil {
		logger.DebugContext(ctx, "analytics: skipped", "error", err)
		return
	}
	idPath, err := analytics.DefaultIDPath()
	if err != nil {
		logger.DebugContext(ctx, "analytics: skipped", "error", err)
		return
	}
	now := time.Now()
	id, err := analytics.MonthlyID(idPath, now)
	if err != nil {
		logger.DebugContext(ctx, "analytics: skipped", "error", err)
		return
	}
	if err := client.TrackStart(ctx, id, now); err != nil {
		logger.DebugContext(ctx, "analytics: send failed", "error", err)
	}
}
