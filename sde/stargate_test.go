package sde

import (
	"context"
	"testing"
)

func TestStargate_KnownID_ReturnsRowWithDestination(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, x, y, z, itemName) VALUES (50008077, 16, 10, 30000142, 1.0, 2.0, 3.0, 'Stargate (Perimeter)')`)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (50008078, 16, 10, 30000144, 'Stargate (Jita)')`)
	seedExec(t, store, `INSERT INTO mapJumps (stargateID, destinationID) VALUES (50008077, 50008078)`)

	got, ok, err := store.Stargate(context.Background(), testUserAgent, 50008077)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if !ok {
		t.Fatal("Stargate ok = false, want true")
	}
	if got.ID != 50008077 || got.SystemID != 30000142 {
		t.Errorf("got = %+v, want ID=50008077 SystemID=30000142", got)
	}
	if got.DestinationStargateID != 50008078 {
		t.Errorf("DestinationStargateID = %d, want 50008078", got.DestinationStargateID)
	}
	if got.DestinationSystemID != 30000144 {
		t.Errorf("DestinationSystemID = %d, want 30000144 (joined from the destination gate's own solarSystemID)", got.DestinationSystemID)
	}
}

func TestStargate_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Stargate(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if ok {
		t.Error("Stargate ok = true, want false for an unknown ID")
	}
}

func TestPutStargate_ThenStargate_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Stargate{ID: 999, Name: "Read-Through Stargate", SystemID: 30000142, DestinationSystemID: 30000144}

	if err := store.PutStargate(want); err != nil {
		t.Fatalf("PutStargate returned error: %v", err)
	}
	got, ok, err := store.Stargate(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if !ok {
		t.Fatal("Stargate ok = false, want true after PutStargate")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}
