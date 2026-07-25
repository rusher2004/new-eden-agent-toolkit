// Package commands defines neat's CLI command tree.
//
// Named commands, not cli, to avoid shadowing the github.com/urfave/cli/v3 import.
package commands

import "github.com/urfave/cli/v3"

// Root builds the top-level neat command, wiring in every subcommand.
func Root(version, clientID string) *cli.Command {
	return &cli.Command{
		Name:  "neat",
		Usage: "New Eden Agent Toolkit",
		Commands: []*cli.Command{
			versionCommand(version),
			setupCommand(),
			callbackHandlerCommand(),
			mcpCommand(version, clientID),
		},
	}
}
