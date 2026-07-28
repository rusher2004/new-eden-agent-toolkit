package sde

import (
	"context"
	"testing"
)

func TestStargate_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "stargates", 50000001,
		`{"_key": 50000001, "name": "Stargate (G-EURJ)", "typeId": 29633, "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000777, "destinationStargateId": 50000482, "destinationSystemId": 30000778}`)

	got, ok, err := store.Stargate(context.Background(), "test-agent", 50000001)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if !ok {
		t.Fatal("Stargate ok = false, want true")
	}
	if got.ID != 50000001 || got.Name != "Stargate (G-EURJ)" || got.TypeID != 29633 || got.SystemID != 30000777 {
		t.Errorf("got = %+v, want ID=50000001 Name=\"Stargate (G-EURJ)\" TypeID=29633 SystemID=30000777", got)
	}
	if got.DestinationStargateID != 50000482 || got.DestinationSystemID != 30000778 {
		t.Errorf("Destination = %d/%d, want 50000482/30000778", got.DestinationStargateID, got.DestinationSystemID)
	}
}

func TestStargate_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Stargate(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if ok {
		t.Error("Stargate ok = true, want false for an unknown ID")
	}
}

func TestStargate_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "stargates", 50000001,
		`{"_key": 999, "name": "Stargate (G-EURJ)", "typeId": 29633, "position": {"x": 1, "y": 2, "z": 3}, "systemId": 30000777, "destinationStargateId": 50000482, "destinationSystemId": 30000778}`)

	_, _, err := store.Stargate(context.Background(), "test-agent", 50000001)
	if err == nil {
		t.Fatal("Stargate returned nil error, want an error for a mismatched _key")
	}
}

func TestPutStargate_ThenStargate_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Stargate{ID: 50000099, Name: "New Stargate", TypeID: 29633, Position: Position{X: 9, Y: 8, Z: 7}, SystemID: 30000099, DestinationStargateID: 50000098, DestinationSystemID: 30000098}

	if err := store.PutStargate(want); err != nil {
		t.Fatalf("PutStargate returned error: %v", err)
	}

	got, ok, err := store.Stargate(context.Background(), "test-agent", 50000099)
	if err != nil {
		t.Fatalf("Stargate returned error: %v", err)
	}
	if !ok {
		t.Fatal("Stargate ok = false after PutStargate, want true")
	}
	if got != want {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
