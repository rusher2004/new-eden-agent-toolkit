// Package commands defines neat's CLI command tree.
//
// Named commands, not cli, to avoid shadowing the github.com/urfave/cli/v3 import.
package commands

import (
	"context"

	"github.com/urfave/cli/v3"
)

// Root builds the top-level neat command, wiring in every subcommand: the
// fixed ones (version, setup, mcp, analytics) and one group per area of
// operations (cliCommands).
//
// Root does not exit the process: an error carrying an exit code
// (cli.ExitCoder) is returned to the caller, which prints its message, if
// any, and exits with its code (cmd/main.go). A leaf command prints its own
// JSON error and returns an exit error with an empty message.
//
// Root panics if the operation commands can't be built (an operation with no
// cli.Paths entry, two sharing a path): that is a programming error in fixed
// tables, caught by TestCLIPathsCoverOperations, like operation.New's panic
// on an underivable schema.
func Root(version, clientID, aptabaseAppKey string, analyticsDebug bool) *cli.Command {
	opCommands, err := cliCommands(version, clientID)
	if err != nil {
		panic(err)
	}
	return &cli.Command{
		Name:  "neat",
		Usage: "New Eden Agent Toolkit",
		Commands: append([]*cli.Command{
			versionCommand(version),
			setupCommand(),
			mcpCommand(version, clientID, aptabaseAppKey, analyticsDebug),
			analyticsCommand(),
		}, opCommands...),
		// The caller maps exit codes; urfave's default would call os.Exit.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
	}
}
