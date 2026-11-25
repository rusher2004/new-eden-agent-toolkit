package main

//go:generate go run ../tools/genversion

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

// clientID is the ESI OAuth2 client ID, set at build time via
// -ldflags "-X main.clientID=<id>". Never committed to source — see
// app_config.json (git-ignored) and the Makefile's build target.
var clientID = "dev"

// aptabaseAppKey is the Aptabase app key for neat's anonymous usage ping,
// set at build time via -ldflags "-X main.aptabaseAppKey=<key>" from
// app_config.json's aptabaseAppKey. Empty (the default) sends nothing.
var aptabaseAppKey = ""

// analyticsDebug marks this build's analytics events as debug data, so
// Aptabase keeps them out of its release numbers. The Makefile sets it to
// "true" via -ldflags "-X main.analyticsDebug=true"; release builds leave
// it empty.
var analyticsDebug = ""

func main() {
	root := commands.Root(version, clientID, aptabaseAppKey, analyticsDebug == "true")
	if err := root.Run(context.Background(), os.Args); err != nil {
		os.Exit(report(err))
	}
}

// report prints err to stderr and returns the exit code. An error carrying an
// exit code (cli.ExitCoder) keeps it, and prints only when its message is
// non-empty: an operation command has already written its {"error": ...}
// line and returns an empty one. Anything else is exit code 1.
//
// A failed write to stderr is not reported: stderr is where it would go.
func report(err error) int {
	if exit, ok := errors.AsType[cli.ExitCoder](err); ok {
		if msg := err.Error(); msg != "" {
			fmt.Fprintf(os.Stderr, "neat: %s\n", msg)
		}
		return exit.ExitCode()
	}
	fmt.Fprintf(os.Stderr, "neat: %v\n", err)
	return 1
}
