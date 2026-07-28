package sde

import (
	"context"
	"testing"
)

func TestRegion_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "regions", 10000001,
		`{"_key": 10000001, "constellationIDs": [20000001, 20000002], "description": {"en": "A dangerous frontier region.", "de": "Eine gefaehrliche Grenzregion."}, "name": {"en": "Derelik", "de": "Derelik"}}`)

	got, ok, err := store.Region(context.Background(), "test-agent", 10000001)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if !ok {
		t.Fatal("Region ok = false, want true")
	}
	if got.ID != 10000001 {
		t.Errorf("got.ID = %d, want 10000001", got.ID)
	}
	if got.Name["en"] != "Derelik" || got.Name["de"] != "Derelik" {
		t.Errorf("Name = %+v, want en=Derelik de=Derelik", got.Name)
	}
	if got.Description["en"] != "A dangerous frontier region." || got.Description["de"] != "Eine gefaehrliche Grenzregion." {
		t.Errorf("Description = %+v, want en/de set", got.Description)
	}
	if len(got.ConstellationIDs) != 2 || got.ConstellationIDs[0] != 20000001 || got.ConstellationIDs[1] != 20000002 {
		t.Errorf("ConstellationIDs = %v, want [20000001 20000002]", got.ConstellationIDs)
	}
}

func TestRegion_NoDescription_ReturnsNilDescription(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "regions", 10000002,
		`{"_key": 10000002, "constellationIDs": [], "name": {"en": "Region With No Description"}}`)

	got, ok, err := store.Region(context.Background(), "test-agent", 10000002)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if !ok {
		t.Fatal("Region ok = false, want true")
	}
	if got.Description != nil {
		t.Errorf("Description = %v, want nil", got.Description)
	}
}

func TestRegion_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Region(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if ok {
		t.Error("Region ok = true, want false for an unknown ID")
	}
}

func TestRegion_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "regions", 10000001,
		`{"_key": 999, "constellationIDs": [], "name": {"en": "Derelik"}}`)

	_, _, err := store.Region(context.Background(), "test-agent", 10000001)
	if err == nil {
		t.Fatal("Region returned nil error, want an error for a mismatched _key")
	}
}

func TestRegionIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "regions", 10000002, `{"_key": 10000002, "constellationIDs": [], "name": {"en": "B"}}`)
	putRecord(t, store, "regions", 10000001, `{"_key": 10000001, "constellationIDs": [], "name": {"en": "A"}}`)

	ids, err := store.RegionIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("RegionIDs returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10000001 || ids[1] != 10000002 {
		t.Errorf("RegionIDs = %v, want [10000001 10000002] (ascending)", ids)
	}
}

func TestRegionIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.RegionIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("RegionIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("RegionIDs = %v, want empty", ids)
	}
}

func TestPutRegion_ThenRegion_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Region{ID: 10000003, Name: map[string]string{"en": "New Region"}, Description: map[string]string{"en": "Written via read-through."}, ConstellationIDs: []int64{20000099}}

	if err := store.PutRegion(want); err != nil {
		t.Fatalf("PutRegion returned error: %v", err)
	}

	got, ok, err := store.Region(context.Background(), "test-agent", 10000003)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if !ok {
		t.Fatal("Region ok = false after PutRegion, want true")
	}
	if got.ID != want.ID || got.Name["en"] != want.Name["en"] || got.Description["en"] != want.Description["en"] {
		t.Errorf("got = %+v, want %+v", got, want)
	}
	if len(got.ConstellationIDs) != 1 || got.ConstellationIDs[0] != 20000099 {
		t.Errorf("ConstellationIDs = %v, want [20000099]", got.ConstellationIDs)
	}
}
