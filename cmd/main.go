package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	root := commands.Root(version)
	if err := root.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "neat: %v\n", err)
		os.Exit(1)
	}
}
