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
			tracker := startAnalytics(context.WithoutCancel(ctx), store, aptabaseAppKey, version)

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
			if tracker != nil {
				server.AddReceivingMiddleware(analytics.ToolMiddleware(tracker))
				flushCtx, stopFlushing := context.WithCancel(context.WithoutCancel(ctx))
				go tracker.Run(flushCtx, time.Minute, logAnalyticsError(ctx))
				defer func() {
					stopFlushing()
					finalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
					defer cancel()
					if err := tracker.Flush(finalCtx); err != nil {
						logAnalyticsError(ctx)(err)
					}
				}()
			}
			return server.Run(ctx, &mcpsdk.StdioTransport{})
		},
	}
}

// startAnalytics sends neat's anonymous "app_started" event in the
// background and returns the ToolTracker that counts tool calls, unless the
// user turned analytics off or the build carries no app key, in which case
// it returns nil and nothing is ever sent (see analytics/). It never fails
// the server: every problem is logged at debug level and dropped.
func startAnalytics(ctx context.Context, store *config.Store, appKey, version string) *analytics.ToolTracker {
	if appKey == "" {
		return nil
	}
	cfg, err := store.Load()
	if err != nil {
		logAnalyticsError(ctx)(fmt.Errorf("config unreadable: %w", err))
		return nil
	}
	if cfg.AnalyticsDisabled {
		return nil
	}
	client, err := analytics.New(appKey, version, sso.UserAgent(version))
	if err != nil {
		logAnalyticsError(ctx)(err)
		return nil
	}
	tracker, err := client.NewToolTracker(time.Now)
	if err != nil {
		logAnalyticsError(ctx)(err)
		return nil
	}

	go func() {
		idPath, err := analytics.DefaultIDPath()
		if err != nil {
			logAnalyticsError(ctx)(err)
			return
		}
		now := time.Now()
		id, err := analytics.MonthlyID(idPath, now)
		if err != nil {
			logAnalyticsError(ctx)(err)
			return
		}
		if err := client.TrackStart(ctx, id, now); err != nil {
			logAnalyticsError(ctx)(err)
		}
	}()
	return tracker
}

func logAnalyticsError(ctx context.Context) func(error) {
	return func(err error) {
		slog.Default().DebugContext(ctx, "analytics: not sent", "error", err)
	}
}
