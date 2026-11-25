// Package character answers questions about the characters neat has logged
// in. It imports only the standard library and config, so domain packages and
// the MCP tools share one implementation of the logged-in check.
package character

import (
	"errors"
	"fmt"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

// ErrNotFound is returned (wrapped) when a character ID is not among the
// logged-in characters.
var ErrNotFound = errors.New("character not found")

// loader is the one config call Known makes. *config.Store satisfies it.
type loader interface {
	Load() (config.Config, error)
}

// Known reports whether characterID is a logged-in character in store. It is
// generic over the unexported loader so callers outside the package can pass a
// value without naming the interface.
func Known[L loader](store L, characterID int32) (bool, error) {
	cfg, err := store.Load()
	if err != nil {
		return false, fmt.Errorf("load config: %w", err)
	}
	for _, c := range cfg.Characters {
		if c.ID == characterID {
			return true, nil
		}
	}
	return false, nil
}
