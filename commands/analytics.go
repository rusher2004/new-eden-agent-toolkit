package commands

import (
	"context"
	"fmt"
	"io"

	"github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/analytics"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func analyticsCommand() *cli.Command {
	return &cli.Command{
		Name:  "analytics",
		Usage: "show or change whether neat sends its anonymous usage statistics (see PRIVACY.md)",
		Commands: []*cli.Command{
			{
				Name:  "status",
				Usage: "show whether the anonymous usage statistics are on",
				Action: func(_ context.Context, cmd *cli.Command) error {
					store, err := config.NewStore()
					if err != nil {
						return fmt.Errorf("resolve config store: %w", err)
					}
					return runAnalyticsStatus(cmd.Writer, store)
				},
			},
			{
				Name:  "on",
				Usage: "turn the anonymous usage statistics on",
				Action: func(_ context.Context, cmd *cli.Command) error {
					store, idPath, err := analyticsPaths()
					if err != nil {
						return err
					}
					return runAnalyticsSet(cmd.Writer, store, idPath, true)
				},
			},
			{
				Name:  "off",
				Usage: "turn the anonymous usage statistics off and delete its saved monthly ID",
				Action: func(_ context.Context, cmd *cli.Command) error {
					store, idPath, err := analyticsPaths()
					if err != nil {
						return err
					}
					return runAnalyticsSet(cmd.Writer, store, idPath, false)
				},
			},
		},
	}
}

func analyticsPaths() (*config.Store, string, error) {
	store, err := config.NewStore()
	if err != nil {
		return nil, "", fmt.Errorf("resolve config store: %w", err)
	}
	idPath, err := analytics.DefaultIDPath()
	if err != nil {
		return nil, "", fmt.Errorf("resolve analytics id path: %w", err)
	}
	return store, idPath, nil
}

func runAnalyticsStatus(w io.Writer, store *config.Store) error {
	cfg, err := store.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	msg := "Anonymous usage statistics are on. Turn them off with: neat analytics off"
	if cfg.AnalyticsDisabled {
		msg = "Anonymous usage statistics are off. Turn them on with: neat analytics on"
	}
	if _, err := fmt.Fprintln(w, msg); err != nil {
		return fmt.Errorf("print analytics status: %w", err)
	}
	return nil
}

func runAnalyticsSet(w io.Writer, store *config.Store, idPath string, on bool) error {
	if _, err := store.Update(func(cfg *config.Config) error { cfg.AnalyticsDisabled = !on; return nil }); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	msg := "Anonymous usage statistics turned on."
	if !on {
		if err := analytics.RemoveID(idPath); err != nil {
			return fmt.Errorf("delete monthly id: %w", err)
		}
		msg = "Anonymous usage statistics turned off. neat will send nothing to Aptabase, and its saved monthly ID is deleted."
	}
	if _, err := fmt.Fprintln(w, msg); err != nil {
		return fmt.Errorf("print analytics status: %w", err)
	}
	return nil
}
