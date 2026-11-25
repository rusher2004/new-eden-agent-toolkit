package mcp

import (
	"context"

	"github.com/rusher2004/new-eden-agent-toolkit/sde"
)

// Every resolver in this file treats a nil cache interface as "nothing to
// look up" and returns no names. The composition root passes a nil *sde.Store
// when the cache can't open; that is a non-nil interface holding a nil
// pointer, so it is not caught by the check here but by sde.Store's own
// nil-safe methods, which fail the lookup, and a failed lookup is dropped. Either
// way a missing SDE degrades to unenriched output instead of failing the tool,
// as the additive-enrichment exception in CLAUDE.md allows.

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
	if len(ids) == 0 || sdeStore == nil {
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
		names[id] = t.Name
	}
	return names
}

// systemNameCache is the minimal read-only view of sde.Store.System() that
// resolveSystemNames needs.
type systemNameCache interface {
	System(ctx context.Context, userAgent string, id int64) (sde.System, bool, error)
}

// resolveSystemNames looks up solar system names for a set of IDs, following
// resolveTypeNames's exact contract: dedup, drop a miss or error rather than
// erroring, never fall back to a live ESI call.
var resolveSystemNames = func(ctx context.Context, sdeStore systemNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		s, ok, err := sdeStore.System(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = s.Name
	}
	return names
}

// regionNameCache is the minimal read-only view of sde.Store.Region() that
// resolveRegionNames needs.
type regionNameCache interface {
	Region(ctx context.Context, userAgent string, id int64) (sde.Region, bool, error)
}

var resolveRegionNames = func(ctx context.Context, sdeStore regionNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		r, ok, err := sdeStore.Region(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = r.Name
	}
	return names
}

// constellationNameCache is the minimal read-only view of
// sde.Store.Constellation() that resolveConstellationNames needs.
type constellationNameCache interface {
	Constellation(ctx context.Context, userAgent string, id int64) (sde.Constellation, bool, error)
}

var resolveConstellationNames = func(ctx context.Context, sdeStore constellationNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		c, ok, err := sdeStore.Constellation(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = c.Name
	}
	return names
}

// stationNameCache is the minimal read-only view of sde.Store.Station() that
// resolveStationNames needs.
type stationNameCache interface {
	Station(ctx context.Context, userAgent string, id int64) (sde.Station, bool, error)
}

// resolveStationNames follows the same contract as resolveSystemNames, but
// sde.Station.Name is a plain string (station names are generated
// server-side, never localized) -- not a map[string]string like System's.
var resolveStationNames = func(ctx context.Context, sdeStore stationNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		s, ok, err := sdeStore.Station(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = s.Name
	}
	return names
}

// planetNameCache is the minimal read-only view of sde.Store.Planet() that
// resolvePlanetNames needs.
type planetNameCache interface {
	Planet(ctx context.Context, userAgent string, id int64) (sde.Planet, bool, error)
}

// resolvePlanetNames follows resolveStationNames's plain-string shape --
// sde.Planet.Name is also a plain string, not a localized map.
var resolvePlanetNames = func(ctx context.Context, sdeStore planetNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		p, ok, err := sdeStore.Planet(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = p.Name
	}
	return names
}

// moonNameCache is the minimal read-only view of sde.Store.Moon() that
// resolveMoonNames needs.
type moonNameCache interface {
	Moon(ctx context.Context, userAgent string, id int64) (sde.Moon, bool, error)
}

// resolveMoonNames follows resolveStationNames's plain-string shape --
// sde.Moon.Name is also a plain string, not a localized map.
var resolveMoonNames = func(ctx context.Context, sdeStore moonNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		m, ok, err := sdeStore.Moon(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = m.Name
	}
	return names
}

// raceNameCache is the minimal read-only view of sde.Store.Race() that
// resolveRaceNames needs.
type raceNameCache interface {
	Race(ctx context.Context, userAgent string, id int64) (sde.Race, bool, error)
}

var resolveRaceNames = func(ctx context.Context, sdeStore raceNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		r, ok, err := sdeStore.Race(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = r.Name
	}
	return names
}

// bloodlineNameCache is the minimal read-only view of sde.Store.Bloodline()
// that resolveBloodlineNames needs.
type bloodlineNameCache interface {
	Bloodline(ctx context.Context, userAgent string, id int64) (sde.Bloodline, bool, error)
}

var resolveBloodlineNames = func(ctx context.Context, sdeStore bloodlineNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		b, ok, err := sdeStore.Bloodline(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = b.Name
	}
	return names
}

// factionNameCache is the minimal read-only view of sde.Store.Faction() that
// resolveFactionNames needs.
type factionNameCache interface {
	Faction(ctx context.Context, userAgent string, id int64) (sde.Faction, bool, error)
}

var resolveFactionNames = func(ctx context.Context, sdeStore factionNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		f, ok, err := sdeStore.Faction(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = f.Name
	}
	return names
}

// groupNameCache is the minimal read-only view of sde.Store.Group() that
// resolveGroupNames needs -- narrower than universe.go's groupCache, which
// also exposes PutGroup/GroupIDs for the Universe tag's own write-back and
// bulk-listing needs that this batch's tools don't use.
type groupNameCache interface {
	Group(ctx context.Context, userAgent string, id int64) (sde.Group, bool, error)
}

var resolveGroupNames = func(ctx context.Context, sdeStore groupNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		g, ok, err := sdeStore.Group(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = g.Name
	}
	return names
}

// skinrComponentNameCache is the minimal read-only view of sde.Store.SkinrComponent() that
// resolveSkinrComponentNames needs.
type skinrComponentNameCache interface {
	SkinrComponent(ctx context.Context, userAgent string, id int64) (sde.SkinrComponent, bool, error)
}

// resolveSkinrComponentNames looks up SKINR component (nanocoating/pattern) names for a set of
// IDs, following resolveTypeNames's exact contract: dedup, drop a miss or error rather than
// erroring. Unlike the other resolvers, a miss has no ESI resource to fall back to even in
// principle -- SKINR components exist only in the SDE.
var resolveSkinrComponentNames = func(ctx context.Context, sdeStore skinrComponentNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		c, ok, err := sdeStore.SkinrComponent(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = c.Name
	}
	return names
}

// skinrSlotNameCache is the minimal read-only view of sde.Store.SkinrSlot() that
// resolveSkinrSlotNames needs.
type skinrSlotNameCache interface {
	SkinrSlot(ctx context.Context, userAgent string, id int64) (sde.SkinrSlot, bool, error)
}

// resolveSkinrSlotNames looks up SKINR cosmetic slot names for a set of slot IDs, following
// resolveTypeNames's exact contract.
var resolveSkinrSlotNames = func(ctx context.Context, sdeStore skinrSlotNameCache, userAgent string, ids []int64) map[int64]string {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	names := make(map[int64]string, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		s, ok, err := sdeStore.SkinrSlot(ctx, userAgent, id)
		if err != nil || !ok {
			continue
		}
		names[id] = s.Name
	}
	return names
}

// typeBasicsCache is the minimal read-only view of sde.Store.TypeBasics()
// that resolveTypeBasics needs.
type typeBasicsCache interface {
	TypeBasics(ctx context.Context, userAgent string, ids []int64) (map[int64]sde.TypeBasic, error)
}

// resolveTypeBasics looks up group, category and volumes for a set of type
// IDs in one batch, deduplicated. Like resolveTypeNames it's additive: a
// failed lookup returns nothing rather than failing the tool call.
var resolveTypeBasics = func(ctx context.Context, sdeStore typeBasicsCache, userAgent string, ids []int64) map[int64]sde.TypeBasic {
	if len(ids) == 0 || sdeStore == nil {
		return nil
	}
	seen := make(map[int64]bool, len(ids))
	uniq := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			uniq = append(uniq, id)
		}
	}
	basics, err := sdeStore.TypeBasics(ctx, userAgent, uniq)
	if err != nil {
		return nil
	}
	return basics
}
