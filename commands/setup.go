package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

func setupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "prepare neat's local config directory and verify keychain access",
		Action: func(_ context.Context, cmd *cli.Command) error {
			store, err := config.NewStore()
			if err != nil {
				return fmt.Errorf("resolve config store: %w", err)
			}
			return runSetup(cmd.Writer, store)
		},
	}
}

func runSetup(w io.Writer, store *config.Store) error {
	if _, err := os.Stat(store.Path()); errors.Is(err, os.ErrNotExist) {
		if err := store.Save(config.Config{}); err != nil {
			return fmt.Errorf("initialize config file: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("check config file: %w", err)
	}
	if _, err := fmt.Fprintf(w, "Config directory ready at %s\n", store.Path()); err != nil {
		return fmt.Errorf("print config directory status: %w", err)
	}

	if err := creds.VerifyAccess(); err != nil {
		return fmt.Errorf("keychain check: %w", err)
	}
	if _, err := fmt.Fprintln(w, "Keychain access verified."); err != nil {
		return fmt.Errorf("print keychain status: %w", err)
	}

	return nil
}
