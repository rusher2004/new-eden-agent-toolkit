package sde

import (
	"context"
	"testing"
)

func TestFactions_ReturnsAllWithAugmentedCounts(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "factions", 500001,
		`{"_key": 500001, "corporationID": 1000035, "description": {"en": "The Caldari State..."}, "militiaCorporationID": 1000180, "name": {"en": "Caldari State"}, "sizeFactor": 5.0, "solarSystemID": 30000145, "uniqueName": true, "stationCount": 120, "stationSystemCount": 45}`)

	got, err := store.Factions(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Factions = %+v, want one entry", got)
	}
	f := got[0]
	if f.ID != 500001 || !f.IsUnique || f.SizeFactor != 5.0 {
		t.Errorf("got = %+v, want ID=500001 IsUnique=true SizeFactor=5.0", f)
	}
	if f.Name["en"] != "Caldari State" {
		t.Errorf("Name[en] = %q, want Caldari State", f.Name["en"])
	}
	if f.CorporationID == nil || *f.CorporationID != 1000035 {
		t.Errorf("CorporationID = %v, want 1000035", f.CorporationID)
	}
	if f.MilitiaCorporationID == nil || *f.MilitiaCorporationID != 1000180 {
		t.Errorf("MilitiaCorporationID = %v, want 1000180", f.MilitiaCorporationID)
	}
	if f.SolarSystemID == nil || *f.SolarSystemID != 30000145 {
		t.Errorf("SolarSystemID = %v, want 30000145", f.SolarSystemID)
	}
	if f.StationCount != 120 || f.StationSystemCount != 45 {
		t.Errorf("StationCount/StationSystemCount = %d/%d, want 120/45", f.StationCount, f.StationSystemCount)
	}
}

func TestFactions_EmptyBucket_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	got, err := store.Factions(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Factions = %+v, want empty", got)
	}
}

func TestFaction_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "factions", 500001, `{"_key": 500001, "corporationID": 1000035, "description": {"en": "..."}, "militiaCorporationID": 1000180, "name": {"en": "Caldari State"}, "sizeFactor": 5.0, "solarSystemID": 30000145, "uniqueName": true, "stationCount": 120, "stationSystemCount": 45}`)

	got, ok, err := store.Faction(context.Background(), "test-agent", 500001)
	if err != nil {
		t.Fatalf("Faction returned error: %v", err)
	}
	if !ok {
		t.Fatal("Faction ok = false, want true")
	}
	if got.ID != 500001 {
		t.Errorf("got.ID = %d, want 500001", got.ID)
	}
	if got.Name["en"] != "Caldari State" {
		t.Errorf("Name[en] = %q, want Caldari State", got.Name["en"])
	}
	if got.IsUnique != true {
		t.Errorf("IsUnique = %v, want true", got.IsUnique)
	}
}

func TestFaction_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Faction(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Faction returned error: %v", err)
	}
	if ok {
		t.Error("Faction ok = true, want false for an unknown ID")
	}
}

func TestFaction_MalformedCachedRecord_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "factions", 500001, `not valid json`)

	_, _, err := store.Faction(context.Background(), "test-agent", 500001)
	if err == nil {
		t.Fatal("Faction returned nil error, want a parse error for a corrupted cache entry")
	}
}

func TestFaction_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "factions", 500001, `{"_key": 999, "corporationID": 1000035, "description": {"en": "..."}, "militiaCorporationID": 1000180, "name": {"en": "Caldari State"}, "sizeFactor": 5.0, "solarSystemID": 30000145, "uniqueName": true, "stationCount": 120, "stationSystemCount": 45}`)

	_, _, err := store.Faction(context.Background(), "test-agent", 500001)
	if err == nil {
		t.Fatal("Faction returned nil error, want an error for a mismatched _key")
	}
}
