package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

// clientID is the ESI OAuth2 client ID, set at build time via
// -ldflags "-X main.clientID=<id>". Never committed to source — see
// app_config.json (git-ignored) and the Makefile's build target.
var clientID = "dev"

func main() {
	root := commands.Root(version, clientID)
	if err := root.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "neat: %v\n", err)
		os.Exit(1)
	}
}
