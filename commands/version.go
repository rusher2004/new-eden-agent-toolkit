package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func versionCommand(version string) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "print the neat version",
		Action: func(_ context.Context, cmd *cli.Command) error {
			if _, err := fmt.Fprintln(cmd.Writer, version); err != nil {
				return fmt.Errorf("print version: %w", err)
			}
			return nil
		},
	}
}
