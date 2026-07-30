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

type fakeSystemNameCache struct {
	systems map[int64]sde.System
	errs    map[int64]error
	calls   map[int64]int
}

func (f *fakeSystemNameCache) System(ctx context.Context, userAgent string, id int64) (sde.System, bool, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	if err, ok := f.errs[id]; ok {
		return sde.System{}, false, err
	}
	s, ok := f.systems[id]
	return s, ok, nil
}

func TestResolveSystemNames_Dedup(t *testing.T) {
	cache := &fakeSystemNameCache{systems: map[int64]sde.System{
		30000142: {ID: 30000142, Name: map[string]string{"en": "Jita"}},
		30002187: {ID: 30002187, Name: map[string]string{"en": "Amarr"}},
	}}

	got := resolveSystemNames(context.Background(), cache, "test-agent", []int64{30000142, 30000142, 30002187})

	if got[30000142] != "Jita" {
		t.Errorf("got[30000142] = %q, want %q", got[30000142], "Jita")
	}
	if got[30002187] != "Amarr" {
		t.Errorf("got[30002187] = %q, want %q", got[30002187], "Amarr")
	}
	if cache.calls[30000142] != 1 {
		t.Errorf("calls[30000142] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[30000142])
	}
}

func TestResolveSystemNames_MissOmitted(t *testing.T) {
	cache := &fakeSystemNameCache{systems: map[int64]sde.System{}}

	got := resolveSystemNames(context.Background(), cache, "test-agent", []int64{99999999})

	if _, ok := got[99999999]; ok {
		t.Errorf("got[99999999] present = %v, want omitted on a cache miss", got[99999999])
	}
}

func TestResolveSystemNames_ErrorOmitted(t *testing.T) {
	cache := &fakeSystemNameCache{errs: map[int64]error{30000142: context.DeadlineExceeded}}

	got := resolveSystemNames(context.Background(), cache, "test-agent", []int64{30000142})

	if _, ok := got[30000142]; ok {
		t.Errorf("got[30000142] present = %v, want omitted when the lookup errors", got[30000142])
	}
}

func TestResolveSystemNames_EmptyIDs(t *testing.T) {
	cache := &fakeSystemNameCache{}

	got := resolveSystemNames(context.Background(), cache, "test-agent", nil)

	if got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}

type fakeRegionNameCache struct {
	regions map[int64]sde.Region
	errs    map[int64]error
	calls   map[int64]int
}

func (f *fakeRegionNameCache) Region(ctx context.Context, userAgent string, id int64) (sde.Region, bool, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	if err, ok := f.errs[id]; ok {
		return sde.Region{}, false, err
	}
	r, ok := f.regions[id]
	return r, ok, nil
}

func TestResolveRegionNames_Dedup(t *testing.T) {
	cache := &fakeRegionNameCache{regions: map[int64]sde.Region{
		10000002: {ID: 10000002, Name: map[string]string{"en": "The Forge"}},
		10000003: {ID: 10000003, Name: map[string]string{"en": "Vale of the Silent"}},
	}}

	got := resolveRegionNames(context.Background(), cache, "test-agent", []int64{10000002, 10000002, 10000003})

	if got[10000002] != "The Forge" {
		t.Errorf("got[10000002] = %q, want %q", got[10000002], "The Forge")
	}
	if got[10000003] != "Vale of the Silent" {
		t.Errorf("got[10000003] = %q, want %q", got[10000003], "Vale of the Silent")
	}
	if cache.calls[10000002] != 1 {
		t.Errorf("calls[10000002] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[10000002])
	}
}

func TestResolveRegionNames_MissOmitted(t *testing.T) {
	cache := &fakeRegionNameCache{regions: map[int64]sde.Region{}}

	got := resolveRegionNames(context.Background(), cache, "test-agent", []int64{99999999})

	if _, ok := got[99999999]; ok {
		t.Errorf("got[99999999] present = %v, want omitted on a cache miss", got[99999999])
	}
}

func TestResolveRegionNames_ErrorOmitted(t *testing.T) {
	cache := &fakeRegionNameCache{errs: map[int64]error{10000002: context.DeadlineExceeded}}

	got := resolveRegionNames(context.Background(), cache, "test-agent", []int64{10000002})

	if _, ok := got[10000002]; ok {
		t.Errorf("got[10000002] present = %v, want omitted when the lookup errors", got[10000002])
	}
}

func TestResolveRegionNames_EmptyIDs(t *testing.T) {
	cache := &fakeRegionNameCache{}

	got := resolveRegionNames(context.Background(), cache, "test-agent", nil)

	if got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}

type fakeConstellationNameCache struct {
	constellations map[int64]sde.Constellation
	errs           map[int64]error
	calls          map[int64]int
}

func (f *fakeConstellationNameCache) Constellation(ctx context.Context, userAgent string, id int64) (sde.Constellation, bool, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	if err, ok := f.errs[id]; ok {
		return sde.Constellation{}, false, err
	}
	c, ok := f.constellations[id]
	return c, ok, nil
}

func TestResolveConstellationNames_Dedup(t *testing.T) {
	cache := &fakeConstellationNameCache{constellations: map[int64]sde.Constellation{
		20000020: {ID: 20000020, Name: map[string]string{"en": "Kimotoro"}},
		20000021: {ID: 20000021, Name: map[string]string{"en": "Sinq Laison"}},
	}}

	got := resolveConstellationNames(context.Background(), cache, "test-agent", []int64{20000020, 20000020, 20000021})

	if got[20000020] != "Kimotoro" {
		t.Errorf("got[20000020] = %q, want %q", got[20000020], "Kimotoro")
	}
	if got[20000021] != "Sinq Laison" {
		t.Errorf("got[20000021] = %q, want %q", got[20000021], "Sinq Laison")
	}
	if cache.calls[20000020] != 1 {
		t.Errorf("calls[20000020] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[20000020])
	}
}

func TestResolveConstellationNames_MissOmitted(t *testing.T) {
	cache := &fakeConstellationNameCache{constellations: map[int64]sde.Constellation{}}

	got := resolveConstellationNames(context.Background(), cache, "test-agent", []int64{99999999})

	if _, ok := got[99999999]; ok {
		t.Errorf("got[99999999] present = %v, want omitted on a cache miss", got[99999999])
	}
}

func TestResolveConstellationNames_ErrorOmitted(t *testing.T) {
	cache := &fakeConstellationNameCache{errs: map[int64]error{20000020: context.DeadlineExceeded}}

	got := resolveConstellationNames(context.Background(), cache, "test-agent", []int64{20000020})

	if _, ok := got[20000020]; ok {
		t.Errorf("got[20000020] present = %v, want omitted when the lookup errors", got[20000020])
	}
}

func TestResolveConstellationNames_EmptyIDs(t *testing.T) {
	cache := &fakeConstellationNameCache{}

	got := resolveConstellationNames(context.Background(), cache, "test-agent", nil)

	if got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}

type fakeStationNameCache struct {
	stations map[int64]sde.Station
	errs     map[int64]error
	calls    map[int64]int
}

func (f *fakeStationNameCache) Station(ctx context.Context, userAgent string, id int64) (sde.Station, bool, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	if err, ok := f.errs[id]; ok {
		return sde.Station{}, false, err
	}
	s, ok := f.stations[id]
	return s, ok, nil
}

func TestResolveStationNames_Dedup(t *testing.T) {
	cache := &fakeStationNameCache{stations: map[int64]sde.Station{
		60003760: {ID: 60003760, Name: "Jita IV - Moon 4 - Caldari Navy Assembly Plant"},
		60004961: {ID: 60004961, Name: "Amarr VIII (Oris) - Emperor Family Academy"},
	}}

	got := resolveStationNames(context.Background(), cache, "test-agent", []int64{60003760, 60003760, 60004961})

	if got[60003760] != "Jita IV - Moon 4 - Caldari Navy Assembly Plant" {
		t.Errorf("got[60003760] = %q, want %q", got[60003760], "Jita IV - Moon 4 - Caldari Navy Assembly Plant")
	}
	if got[60004961] != "Amarr VIII (Oris) - Emperor Family Academy" {
		t.Errorf("got[60004961] = %q, want %q", got[60004961], "Amarr VIII (Oris) - Emperor Family Academy")
	}
	if cache.calls[60003760] != 1 {
		t.Errorf("calls[60003760] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[60003760])
	}
}

func TestResolveStationNames_MissOmitted(t *testing.T) {
	cache := &fakeStationNameCache{stations: map[int64]sde.Station{}}

	got := resolveStationNames(context.Background(), cache, "test-agent", []int64{99999999})

	if _, ok := got[99999999]; ok {
		t.Errorf("got[99999999] present = %v, want omitted on a cache miss", got[99999999])
	}
}

func TestResolveStationNames_ErrorOmitted(t *testing.T) {
	cache := &fakeStationNameCache{errs: map[int64]error{60003760: context.DeadlineExceeded}}

	got := resolveStationNames(context.Background(), cache, "test-agent", []int64{60003760})

	if _, ok := got[60003760]; ok {
		t.Errorf("got[60003760] present = %v, want omitted when the lookup errors", got[60003760])
	}
}

func TestResolveStationNames_EmptyIDs(t *testing.T) {
	cache := &fakeStationNameCache{}

	got := resolveStationNames(context.Background(), cache, "test-agent", nil)

	if got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}

type fakePlanetNameCache struct {
	planets map[int64]sde.Planet
	errs    map[int64]error
	calls   map[int64]int
}

func (f *fakePlanetNameCache) Planet(ctx context.Context, userAgent string, id int64) (sde.Planet, bool, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	if err, ok := f.errs[id]; ok {
		return sde.Planet{}, false, err
	}
	p, ok := f.planets[id]
	return p, ok, nil
}

func TestResolvePlanetNames_Dedup(t *testing.T) {
	cache := &fakePlanetNameCache{planets: map[int64]sde.Planet{
		40009086: {ID: 40009086, Name: "Jita IV"},
		40009087: {ID: 40009087, Name: "Amarr VIII"},
	}}

	got := resolvePlanetNames(context.Background(), cache, "test-agent", []int64{40009086, 40009086, 40009087})

	if got[40009086] != "Jita IV" {
		t.Errorf("got[40009086] = %q, want %q", got[40009086], "Jita IV")
	}
	if got[40009087] != "Amarr VIII" {
		t.Errorf("got[40009087] = %q, want %q", got[40009087], "Amarr VIII")
	}
	if cache.calls[40009086] != 1 {
		t.Errorf("calls[40009086] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[40009086])
	}
}

func TestResolvePlanetNames_MissOmitted(t *testing.T) {
	cache := &fakePlanetNameCache{planets: map[int64]sde.Planet{}}

	got := resolvePlanetNames(context.Background(), cache, "test-agent", []int64{99999999})

	if _, ok := got[99999999]; ok {
		t.Errorf("got[99999999] present = %v, want omitted on a cache miss", got[99999999])
	}
}

func TestResolvePlanetNames_ErrorOmitted(t *testing.T) {
	cache := &fakePlanetNameCache{errs: map[int64]error{40009086: context.DeadlineExceeded}}

	got := resolvePlanetNames(context.Background(), cache, "test-agent", []int64{40009086})

	if _, ok := got[40009086]; ok {
		t.Errorf("got[40009086] present = %v, want omitted when the lookup errors", got[40009086])
	}
}

func TestResolvePlanetNames_EmptyIDs(t *testing.T) {
	cache := &fakePlanetNameCache{}

	got := resolvePlanetNames(context.Background(), cache, "test-agent", nil)

	if got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}

type fakeMoonNameCache struct {
	moons map[int64]sde.Moon
	errs  map[int64]error
	calls map[int64]int
}

func (f *fakeMoonNameCache) Moon(ctx context.Context, userAgent string, id int64) (sde.Moon, bool, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[id]++
	if err, ok := f.errs[id]; ok {
		return sde.Moon{}, false, err
	}
	m, ok := f.moons[id]
	return m, ok, nil
}

func TestResolveMoonNames_Dedup(t *testing.T) {
	cache := &fakeMoonNameCache{moons: map[int64]sde.Moon{
		40009087: {ID: 40009087, Name: "Jita IV - Moon 4"},
		40009088: {ID: 40009088, Name: "Amarr VIII - Moon 1"},
	}}

	got := resolveMoonNames(context.Background(), cache, "test-agent", []int64{40009087, 40009087, 40009088})

	if got[40009087] != "Jita IV - Moon 4" {
		t.Errorf("got[40009087] = %q, want %q", got[40009087], "Jita IV - Moon 4")
	}
	if got[40009088] != "Amarr VIII - Moon 1" {
		t.Errorf("got[40009088] = %q, want %q", got[40009088], "Amarr VIII - Moon 1")
	}
	if cache.calls[40009087] != 1 {
		t.Errorf("calls[40009087] = %d, want 1 (dedup should collapse the repeated ID)", cache.calls[40009087])
	}
}

func TestResolveMoonNames_MissOmitted(t *testing.T) {
	cache := &fakeMoonNameCache{moons: map[int64]sde.Moon{}}

	got := resolveMoonNames(context.Background(), cache, "test-agent", []int64{99999999})

	if _, ok := got[99999999]; ok {
		t.Errorf("got[99999999] present = %v, want omitted on a cache miss", got[99999999])
	}
}

func TestResolveMoonNames_ErrorOmitted(t *testing.T) {
	cache := &fakeMoonNameCache{errs: map[int64]error{40009087: context.DeadlineExceeded}}

	got := resolveMoonNames(context.Background(), cache, "test-agent", []int64{40009087})

	if _, ok := got[40009087]; ok {
		t.Errorf("got[40009087] present = %v, want omitted when the lookup errors", got[40009087])
	}
}

func TestResolveMoonNames_EmptyIDs(t *testing.T) {
	cache := &fakeMoonNameCache{}

	got := resolveMoonNames(context.Background(), cache, "test-agent", nil)

	if got != nil {
		t.Errorf("got = %v, want nil for an empty ID list", got)
	}
	if len(cache.calls) != 0 {
		t.Errorf("calls = %v, want no lookups for an empty ID list", cache.calls)
	}
}
