package universe_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
	"github.com/rusher2004/new-eden-agent-toolkit/universe"
)

type fakeCache struct {
	types map[int64]sde.Type
	errs  map[int64]error
	calls int
}

func (f *fakeCache) Type(_ context.Context, _ string, id int64) (sde.Type, bool, error) {
	f.calls++
	if err := f.errs[id]; err != nil {
		return sde.Type{}, false, err
	}
	t, ok := f.types[id]
	return t, ok, nil
}

func TestTypeNames_DedupesLookups(t *testing.T) {
	c := &fakeCache{types: map[int64]sde.Type{34: {Name: "Tritanium"}, 35: {Name: "Pyerite"}}}
	got := universe.NewTypeNamer(c, "ua").TypeNames(context.Background(), []int64{34, 34, 35})
	if c.calls != 2 {
		t.Fatalf("lookups = %d, want 2", c.calls)
	}
	if got[34] != "Tritanium" || got[35] != "Pyerite" || len(got) != 2 {
		t.Fatalf("names = %v", got)
	}
}

func TestTypeNames_DropsMissesAndErrors(t *testing.T) {
	c := &fakeCache{
		types: map[int64]sde.Type{34: {Name: "Tritanium"}},
		errs:  map[int64]error{36: errors.New("boom")},
	}
	got := universe.NewTypeNamer(c, "ua").TypeNames(context.Background(), []int64{34, 35, 36})
	if len(got) != 1 || got[34] != "Tritanium" {
		t.Fatalf("names = %v, want only 34", got)
	}
}

func TestTypeNames_NoIDsReturnsNil(t *testing.T) {
	c := &fakeCache{}
	if got := universe.NewTypeNamer(c, "ua").TypeNames(context.Background(), nil); got != nil {
		t.Fatalf("names = %v, want nil", got)
	}
	if c.calls != 0 {
		t.Fatalf("lookups = %d, want 0", c.calls)
	}
}

type typeLooker interface {
	Type(ctx context.Context, userAgent string, id int64) (sde.Type, bool, error)
}

func TestTypeNames_NilCacheReturnsNil(t *testing.T) {
	var c typeLooker
	if got := universe.NewTypeNamer(c, "ua").TypeNames(context.Background(), []int64{34}); got != nil {
		t.Fatalf("names = %v, want nil", got)
	}
}
