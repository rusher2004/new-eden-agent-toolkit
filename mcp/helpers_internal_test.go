package mcp

import (
	"context"
	"errors"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// errBoom is the error the internal tests' fakes fail with.
var errBoom = errors.New("boom")

// noTypesSDE is a typeNameCache that knows no types: every lookup misses, so
// name enrichment adds nothing.
type noTypesSDE struct{}

func (noTypesSDE) Type(context.Context, string, int64) (sde.Type, bool, error) {
	return sde.Type{}, false, nil
}
