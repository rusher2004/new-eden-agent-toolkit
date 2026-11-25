package commands

import (
	"github.com/rusher2004/new-eden-agent-toolkit/loyalty"
	"github.com/rusher2004/new-eden-agent-toolkit/operation"
	"github.com/rusher2004/new-eden-agent-toolkit/universe"
)

// Operations builds every domain service over d's sources and returns their
// operations, which the MCP server registers with mcp.Register and the CLI
// builds commands from.
//
// It is safe with a zero toolDeps: constructors only store d's sources and
// closures over them, so nothing opens a file, the keychain or the network
// until an operation runs. A zero toolDeps gives operations whose metadata
// (name, description, kind, types, schemas) is complete but which must not be
// run.
func Operations(d toolDeps) []operation.Operation {
	names := universe.NewTypeNamer(d.sde, d.userAgent)
	return loyalty.New(d.openESI, d.openPublicESI, d.store, names).Operations()
}

// Catalog returns the operations' metadata without any sources, for listing
// names, descriptions and schemas (documentation, CLI help). Its operations
// must not be run: they have no ESI client, config store or SDE behind them.
func Catalog() []operation.Operation { return Operations(toolDeps{}) }
