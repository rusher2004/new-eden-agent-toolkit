package commands

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/urlscheme"
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

			launcherPath, err := ensureLauncher()
			if err != nil {
				return fmt.Errorf("ensure launcher binary: %w", err)
			}

			if err := urlscheme.Register(launcherPath); err != nil {
				return fmt.Errorf("register URL scheme: %w", err)
			}

			server := mcp.NewServer(version, clientID, store, sdeStore)
			return server.Run(ctx, &mcpsdk.StdioTransport{})
		},
	}
}

// currentExecutable resolves the path to the currently running binary.
// Overridable in tests so they don't depend on the actual go test binary.
var currentExecutable = os.Executable

// ensureLauncher makes sure ~/.neat/bin/neat exists and matches the
// currently running binary, copying it into place if missing or the
// running binary lives elsewhere. Returns the resolved launcher path.
func ensureLauncher() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	launcherName := "neat"
	if runtime.GOOS == "windows" {
		launcherName += ".exe"
	}
	target := filepath.Join(home, ".neat", "bin", launcherName)

	current, err := currentExecutable()
	if err != nil {
		return "", fmt.Errorf("resolve running executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(current); err == nil {
		current = resolved
	}

	if resolvedTarget, err := filepath.EvalSymlinks(target); err == nil && resolvedTarget == current {
		return target, nil
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return "", fmt.Errorf("create launcher directory: %w", err)
	}

	src, err := os.Open(current)
	if err != nil {
		return "", fmt.Errorf("open running executable: %w", err)
	}
	defer src.Close()

	tmp := target + ".tmp"
	dst, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return "", fmt.Errorf("create launcher temp file: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return "", fmt.Errorf("copy launcher binary: %w", err)
	}
	if err := dst.Close(); err != nil {
		return "", fmt.Errorf("close launcher temp file: %w", err)
	}

	if err := os.Rename(tmp, target); err != nil {
		return "", fmt.Errorf("rename launcher into place: %w", err)
	}

	return target, nil
}
