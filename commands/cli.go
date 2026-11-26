package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	neatcli "github.com/rusher2004/new-eden-agent-toolkit/cli"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/operation"
	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// cliCommands builds the operation command groups (neat loyalty ...). Their
// names, help and flags come from Catalog, which touches nothing; the real
// operations are built by cliSources only when a leaf command runs, so
// neat --help and a group's --help open no file, keychain or network.
func cliCommands(version, clientID string) ([]*cli.Command, error) {
	cmds, err := neatcli.Build(Catalog(), func(ctx context.Context, warn func(string)) ([]operation.Operation, func() error, error) {
		return cliSources(ctx, version, clientID, warn)
	})
	if err != nil {
		return nil, fmt.Errorf("build operation commands: %w", err)
	}
	return cmds, nil
}

// cliSources builds the operations over the config store and SDE cache, as
// neat mcp does, and returns a function that closes the SDE cache. warn is the
// CLI's stderr warning sink, which keeps stderr JSON only.
func cliSources(_ context.Context, version, clientID string, warn func(string)) ([]operation.Operation, func() error, error) {
	store, err := config.NewStore()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve config store: %w", err)
	}
	sdePath, err := sde.DefaultPath()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve sde cache path: %w", err)
	}
	// As in neat mcp: a failed Open leaves a nil store, whose every method is
	// nil-safe, so name enrichment falls through to ESI instead of failing
	// commands that don't need the SDE.
	sdeStore, err := sde.Open(sdePath)
	if err != nil {
		warn(fmt.Sprintf("sde: failed to open cache, names will read through to ESI directly: %v", err))
		sdeStore = nil
	}
	deps := toolDeps{userAgent: sso.UserAgent(version), store: store, sde: sdeStore, clientID: clientID}
	closeFn := func() error {
		if err := sdeStore.Close(); err != nil {
			return fmt.Errorf("close sde cache: %w", err)
		}
		return nil
	}
	return Operations(deps), closeFn, nil
}
