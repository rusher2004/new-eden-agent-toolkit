package sde

import (
	"context"
	"testing"
)

func TestBloodlines_ReturnsEveryCachedBloodline(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrBloodlines (bloodlineID, bloodlineName, raceID, description, shipTypeID, corporationID, perception, willpower, charisma, memory, intelligence)
		VALUES (1, 'Deteis', 1, 'The Deteis.', 601, 1000035, 5, 6, 7, 8, 9)`)

	got, err := store.Bloodlines(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Bloodlines returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(Bloodlines) = %d, want 1", len(got))
	}
	b := got[0]
	if b.ID != 1 || b.Name != "Deteis" || b.RaceID != 1 || b.CorporationID != 1000035 {
		t.Errorf("got = %+v, want ID=1 Name=Deteis RaceID=1 CorporationID=1000035", b)
	}
	if b.ShipTypeID == nil || *b.ShipTypeID != 601 {
		t.Errorf("ShipTypeID = %v, want 601", b.ShipTypeID)
	}
	if b.Perception != 5 || b.Willpower != 6 || b.Charisma != 7 || b.Memory != 8 || b.Intelligence != 9 {
		t.Errorf("attribute fields = %+v, want Perception=5 Willpower=6 Charisma=7 Memory=8 Intelligence=9", b)
	}
}

func TestBloodline_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrBloodlines (bloodlineID, bloodlineName, raceID) VALUES (1, 'Deteis', 1)`)

	got, ok, err := store.Bloodline(context.Background(), testUserAgent, 1)
	if err != nil {
		t.Fatalf("Bloodline returned error: %v", err)
	}
	if !ok {
		t.Fatal("Bloodline ok = false, want true")
	}
	if got.Name != "Deteis" {
		t.Errorf("Name = %q, want Deteis", got.Name)
	}
}

func TestBloodline_NullRaceIDAndName_DegradeToZeroValueInsteadOfErroring(t *testing.T) {
	// raceID and bloodlineName are both nullable in the fuzzwork schema.
	// See PR #14 review finding #2 -- corporationID/the five attribute
	// columns were already guarded, raceID/name weren't.
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrBloodlines (bloodlineID, bloodlineName, raceID) VALUES (1, NULL, NULL)`)

	got, ok, err := store.Bloodline(context.Background(), testUserAgent, 1)
	if err != nil {
		t.Fatalf("Bloodline returned error: %v, want nil (NULL raceID/name should degrade gracefully)", err)
	}
	if !ok {
		t.Fatal("Bloodline ok = false, want true")
	}
	if got.Name != "" || got.RaceID != 0 {
		t.Errorf("Name/RaceID = %q/%d, want empty string/0", got.Name, got.RaceID)
	}
}

func TestBloodline_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Bloodline(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Bloodline returned error: %v", err)
	}
	if ok {
		t.Error("Bloodline ok = true, want false for an unknown ID")
	}
}
