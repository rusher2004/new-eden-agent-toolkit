package commands_test

import (
	"path/filepath"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// A nil *sde.Store is what neat mcp passes when the SDE cache can't open; its
// methods are nil-safe, so the server must still build.
func TestNewMCPServer_BuildsWithNilSDE(t *testing.T) {
	store := config.NewStoreAt(filepath.Join(t.TempDir(), "config.json"))
	if commands.NewMCPServer("test", "client-id", store, nil) == nil {
		t.Fatal("NewMCPServer with a nil SDE store returned no server")
	}
}
