package sde

import (
	"context"
	"testing"
)

func TestRegion_KnownID_ReturnsRowWithNilDescription(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapRegions (regionID, regionName) VALUES (10000001, 'Derelik')`)
	seedExec(t, store, `INSERT INTO mapConstellations (constellationID, regionID, constellationName) VALUES (20000001, 10000001, 'San Matar')`)

	got, ok, err := store.Region(context.Background(), testUserAgent, 10000001)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if !ok {
		t.Fatal("Region ok = false, want true")
	}
	if got.ID != 10000001 || got.Name != "Derelik" {
		t.Errorf("got = %+v, want ID=10000001 Name=Derelik", got)
	}
	if got.Description != nil {
		t.Errorf("Description = %v, want nil (mapRegions has no description column)", *got.Description)
	}
	if len(got.ConstellationIDs) != 1 || got.ConstellationIDs[0] != 20000001 {
		t.Errorf("ConstellationIDs = %v, want [20000001]", got.ConstellationIDs)
	}
}

func TestRegion_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Region(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if ok {
		t.Error("Region ok = true, want false for an unknown ID")
	}
}

func TestPutRegion_ThenRegion_ReturnsOverlaidDescription(t *testing.T) {
	store := seedStore(t)
	description := "A dangerous frontier region."
	want := Region{ID: 999, Name: "Read-Through Region", Description: &description, ConstellationIDs: []int64{1}}

	if err := store.PutRegion(want); err != nil {
		t.Fatalf("PutRegion returned error: %v", err)
	}
	got, ok, err := store.Region(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Region returned error: %v", err)
	}
	if !ok {
		t.Fatal("Region ok = false, want true after PutRegion")
	}
	if got.Description == nil || *got.Description != description {
		t.Errorf("Description = %v, want %q (overlay values keep whatever PutRegion was given)", got.Description, description)
	}
}

func TestRegionIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapRegions (regionID, regionName) VALUES (10000001, 'Derelik')`)
	seedExec(t, store, `INSERT INTO mapRegions (regionID, regionName) VALUES (10000002, 'The Forge')`)

	got, err := store.RegionIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("RegionIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 10000001 || got[1] != 10000002 {
		t.Errorf("RegionIDs = %v, want [10000001 10000002]", got)
	}
}
