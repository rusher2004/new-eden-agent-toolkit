package mcp

import (
	"context"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

type fakeTypeNameCache struct {
	types map[int64]sde.Type
	errs  map[int64]error
	calls map[int64]int
}

func (f *fakeTypeNameCache) Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	if err, ok := f.errs[id]; ok {
		return sde.Type{}, false, err
	}
	t, ok := f.types[id]
	return t, ok, nil
}

func TestResolveTypeNames_Dedup(t *testing.T) {
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{
		587: {ID: 587, Name: map[string]string{"en": "Rifter"}},
		588: {ID: 588, Name: map[string]string{"en": "Slasher"}},
	}}

	got := resolveTypeNames(context.Background(), cache, "test-agent", []int64{587, 587, 588})

	if got[587] != "Rifter" {
		t.Errorf("got[587] = %q, want %q", got[587], "Rifter")
	}
	if got[588] != "Slasher" {
		t.Errorf("got[588] = %q, want %q", got[588], "Slasher")
	}
	if cache.calls[587] != 1 {
		t.Errorf("calls[587] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[587])
	}
}

func TestResolveTypeNames_MissOmitted(t *testing.T) {
	cache := &fakeTypeNameCache{types: map[int64]sde.Type{}}

	got := resolveTypeNames(context.Background(), cache, "test-agent", []int64{999})

	if _, ok := got[999]; ok {
		t.Errorf("got[999] present = %v, want omitted on a cache miss", got[999])
	}
}

func TestResolveTypeNames_ErrorOmitted(t *testing.T) {
	cache := &fakeTypeNameCache{errs: map[int64]error{123: context.DeadlineExceeded}}

	got := resolveTypeNames(context.Background(), cache, "test-agent", []int64{123})

	if _, ok := got[123]; ok {
		t.Errorf("got[123] present = %v, want omitted when the lookup errors", got[123])
	}
}

func TestResolveTypeNames_EmptyIDs(t *testing.T) {
	cache := &fakeTypeNameCache{}

	got := resolveTypeNames(context.Background(), cache, "test-agent", nil)

	if got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}
