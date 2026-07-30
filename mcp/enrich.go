package mcp

import (
	"context"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// typeNameCache is the minimal read-only view of sde.Store.Type() that
// resolveTypeNames needs -- narrower than universe.go's typeCache, which
// also exposes PutType/TypeIDs for the Universe tag's own write-back and
// bulk-listing needs that this batch's tools don't use.
type typeNameCache interface {
	Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error)
}

// resolveTypeNames looks up names for a set of type IDs, deduplicating so a
// value repeated across many records (e.g. the same item type on hundreds of
// asset rows) costs one lookup, not one per occurrence. A lookup failure is
// a cache miss (the ID isn't in the local SDE), not a network error: unlike
// universe.go's read-through path, this resolver deliberately never falls
// back to a live ESI call, because the high-cardinality callers
// (list_market_prices, get_killmail, contract items) would turn one tool
// call into thousands of ESI requests. Either way, a miss is dropped from
// the result rather than surfacing an error: enrichment is additive, not
// required, so one bad ID must not fail an entire tool call over data ESI
// already returned successfully.
var resolveTypeNames = func(ctx context.Context, sdeStore typeNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		t, ok, err := sdeStore.Type(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = t.Name["en"]
	}
	return names
}
