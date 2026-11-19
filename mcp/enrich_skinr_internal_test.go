package mcp

import (
	"context"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// fakeSkinrNameCache implements both skinrComponentNameCache and skinrSlotNameCache.
type fakeSkinrNameCache struct {
	components map[int64]sde.SkinrComponent
	slots      map[int64]sde.SkinrSlot
	errs       map[int64]error
	calls      map[int64]int
}

func (f *fakeSkinrNameCache) record(id int64) error {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	return f.errs[id]
}

func (f *fakeSkinrNameCache) SkinrComponent(ctx context.Context, userAgent string, id int64) (sde.SkinrComponent, bool, error) {
	if err := f.record(id); err != nil {
		return sde.SkinrComponent{}, false, err
	}
	c, ok := f.components[id]
	return c, ok, nil
}

func (f *fakeSkinrNameCache) SkinrSlot(ctx context.Context, userAgent string, id int64) (sde.SkinrSlot, bool, error) {
	if err := f.record(id); err != nil {
		return sde.SkinrSlot{}, false, err
	}
	s, ok := f.slots[id]
	return s, ok, nil
}

func TestResolveSkinrComponentNames_DedupMissAndError(t *testing.T) {
	cache := &fakeSkinrNameCache{
		components: map[int64]sde.SkinrComponent{1254: {ID: 1254, Name: "Charcoal Covenant Satin"}},
		errs:       map[int64]error{7: context.DeadlineExceeded},
	}

	got := resolveSkinrComponentNames(context.Background(), cache, "test-agent", []int64{1254, 1254, 999, 7})

	if got[1254] != "Charcoal Covenant Satin" {
		t.Errorf("got[1254] = %q, want %q", got[1254], "Charcoal Covenant Satin")
	}
	if cache.calls[1254] != 1 {
		t.Errorf("calls[1254] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[1254])
	}
	if _, ok := got[999]; ok {
		t.Error("got[999] present, want omitted on a cache miss")
	}
	if _, ok := got[7]; ok {
		t.Error("got[7] present, want omitted when the lookup errors")
	}
}

func TestResolveSkinrComponentNames_EmptyIDs(t *testing.T) {
	cache := &fakeSkinrNameCache{}
	if got := resolveSkinrComponentNames(context.Background(), cache, "test-agent", nil); got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}

func TestResolveSkinrSlotNames_DedupMissAndError(t *testing.T) {
	cache := &fakeSkinrNameCache{
		slots: map[int64]sde.SkinrSlot{5: {ID: 5, Name: "Primary Pattern Slot"}},
		errs:  map[int64]error{7: context.DeadlineExceeded},
	}

	got := resolveSkinrSlotNames(context.Background(), cache, "test-agent", []int64{5, 5, 99, 7})

	if got[5] != "Primary Pattern Slot" {
		t.Errorf("got[5] = %q, want %q", got[5], "Primary Pattern Slot")
	}
	if cache.calls[5] != 1 {
		t.Errorf("calls[5] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[5])
	}
	if _, ok := got[99]; ok {
		t.Error("got[99] present, want omitted on a cache miss")
	}
	if _, ok := got[7]; ok {
		t.Error("got[7] present, want omitted when the lookup errors")
	}
}

func TestResolveSkinrSlotNames_EmptyIDs(t *testing.T) {
	cache := &fakeSkinrNameCache{}
	if got := resolveSkinrSlotNames(context.Background(), cache, "test-agent", nil); got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
}
