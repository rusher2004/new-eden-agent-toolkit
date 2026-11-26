package mcp_test

import (
	"errors"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/character"
	"github.com/rusher2004/new-eden-agent-toolkit/mcp"
)

// ErrCharacterNotFound is an alias for character.ErrNotFound, so callers
// branching on either match errors from both the tools and the operations.
func TestErrCharacterNotFoundIsCharacterErrNotFound(t *testing.T) {
	if mcp.ErrCharacterNotFound != character.ErrNotFound {
		t.Errorf("mcp.ErrCharacterNotFound (%v) is not the same value as character.ErrNotFound (%v)", mcp.ErrCharacterNotFound, character.ErrNotFound)
	}
	if !errors.Is(mcp.ErrCharacterNotFound, character.ErrNotFound) {
		t.Error("errors.Is(mcp.ErrCharacterNotFound, character.ErrNotFound) = false")
	}
}
