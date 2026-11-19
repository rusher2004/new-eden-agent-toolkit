package esi

import (
	"context"
	"fmt"
)

// maxPages guards against a runaway X-Pages header.
const maxPages = 500

// AllPages fetches page 1, reads the total page count it reports, then
// fetches the rest in order and concatenates them. fetch is one of this
// package's paginated methods (e.g. a closure over CorporationAssets), so
// errors already carry ESI's detail via describeError.
func AllPages[T any](ctx context.Context, fetch func(ctx context.Context, page int32) ([]T, int32, error)) ([]T, error) {
	first, total, err := fetch(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("page 1: %w", err)
	}
	if total > maxPages {
		return nil, fmt.Errorf("ESI reports %d pages, more than the %d this helper will fetch", total, maxPages)
	}
	out := first
	for p := int32(2); p <= total; p++ {
		items, _, err := fetch(ctx, p)
		if err != nil {
			return nil, fmt.Errorf("page %d of %d: %w", p, total, err)
		}
		out = append(out, items...)
	}
	return out, nil
}
