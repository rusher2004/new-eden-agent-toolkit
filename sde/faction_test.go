package sde

import (
	"context"
	"testing"
)

func TestFactions_ReturnsEveryCachedFactionWithNilIsUnique(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrFactions (factionID, factionName, description, solarSystemID, corporationID, sizeFactor, stationCount, stationSystemCount, militiaCorporationID)
		VALUES (500001, 'Caldari State', 'The Caldari State.', 30000142, 1000035, 1.0, 10, 5, 1000180)`)

	got, err := store.Factions(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(Factions) = %d, want 1", len(got))
	}
	f := got[0]
	if f.ID != 500001 || f.Name != "Caldari State" || f.Description != "The Caldari State." {
		t.Errorf("got = %+v, want ID=500001 Name=%q Description=%q", f, "Caldari State", "The Caldari State.")
	}
	if f.IsUnique != nil {
		t.Errorf("IsUnique = %v, want nil (chrFactions has no uniqueName-equivalent column)", *f.IsUnique)
	}
	if f.StationCount != 10 || f.StationSystemCount != 5 {
		t.Errorf("StationCount/StationSystemCount = %d/%d, want 10/5", f.StationCount, f.StationSystemCount)
	}
	if f.CorporationID == nil || *f.CorporationID != 1000035 {
		t.Errorf("CorporationID = %v, want 1000035", f.CorporationID)
	}
}

func TestFactions_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := &Store{}
	got, err := store.Factions(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Factions returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Factions = %v, want empty", got)
	}
}

func TestFaction_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrFactions (factionID, factionName, sizeFactor, stationCount, stationSystemCount) VALUES (500001, 'Caldari State', 1.0, 0, 0)`)

	got, ok, err := store.Faction(context.Background(), testUserAgent, 500001)
	if err != nil {
		t.Fatalf("Faction returned error: %v", err)
	}
	if !ok {
		t.Fatal("Faction ok = false, want true")
	}
	if got.Name != "Caldari State" {
		t.Errorf("Name = %q, want Caldari State", got.Name)
	}
}

func TestFaction_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Faction(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Faction returned error: %v", err)
	}
	if ok {
		t.Error("Faction ok = true, want false for an unknown ID")
	}
}
