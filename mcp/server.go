//go:generate go run ../tools/gentooldocs -out ../docs/reference/tools.md

// Package mcp holds neat's MCP tools. Each tool file declares the small
// interfaces it needs, a handler constructor that takes them, and an exported
// Add function that registers the tool; the composition root (commands/)
// builds the concrete config store, SDE cache, ESI clients and login flow and
// passes them to each Add function, so nothing here constructs one. See
// loyalty.go.
package mcp

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer returns an MCP server that carries neat's instructions and no
// tools. The composition root (commands.NewMCPServer) registers every tool on
// it, one explicit Add call each.
func NewServer(version string) *mcpsdk.Server {
	return mcpsdk.NewServer(&mcpsdk.Implementation{Name: "neat", Version: version}, &mcpsdk.ServerOptions{Instructions: Instructions})
}
