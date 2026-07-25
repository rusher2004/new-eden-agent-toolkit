// Package mcp builds neat's MCP server and registers its character tools.
package mcp

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// NewServer builds neat's MCP server with add_character, list_characters,
// and remove_character registered.
func NewServer(version, clientID string, store *config.Store) *mcpsdk.Server {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "neat", Version: version}, nil)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "add_character",
		Description: "Log in a new EVE Online character via ESI OAuth2, opening the user's browser to complete authentication.",
	}, addCharacterHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_characters",
		Description: "List EVE Online characters currently logged in to neat.",
	}, listCharactersHandler(store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "remove_character",
		Description: "Remove a logged-in EVE Online character, deleting its stored credentials.",
	}, removeCharacterHandler(store))

	return server
}
