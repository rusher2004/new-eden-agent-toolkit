// Package universe holds lookups over New Eden's static universe data.
package universe

import (
	"context"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// typeCache is the read-only view of the SDE that TypeNamer needs.
type typeCache interface {
	Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error)
}

// TypeNamer resolves type IDs to names from the local SDE.
type TypeNamer struct {
	cache     typeCache
	userAgent string
}

// NewTypeNamer returns a TypeNamer reading from cache. A nil cache yields a
// namer that resolves nothing. A typed-nil pointer (such as a nil *sde.Store
// when the cache can't open) is a non-nil interface; sde.Store's own
// nil-safe methods fail those lookups, and a failed lookup is dropped.
func NewTypeNamer[C typeCache](cache C, userAgent string) *TypeNamer {
	n := &TypeNamer{userAgent: userAgent}
	if any(cache) != nil {
		n.cache = cache
	}
	return n
}

// TypeNames looks up names for a set of type IDs, deduplicating so a value
// repeated across many records (e.g. the same item type on hundreds of asset
// rows) costs one lookup, not one per occurrence. It never falls back to a
// live ESI call: the high-cardinality callers (market prices, killmails,
// contract items) would turn one tool call into thousands of ESI requests.
//
// Best-effort exception (CLAUDE.md, additive enrichment): a miss or a lookup
// error is dropped from the result rather than returned, because enrichment
// is additive, so one bad ID must not fail a call over data ESI already
// returned successfully. It returns nil for no IDs or no cache.
func (n *TypeNamer) TypeNames(ctx context.Context, ids []int64) map[int64]string {
	if len(ids) == 0 || n == nil || n.cache == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		t, ok, err := n.cache.Type(ctx, n.userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = t.Name
	}
	return names
}
