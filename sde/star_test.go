package sde

import (
	"context"
	"testing"
)

func TestStar_KnownID_ReturnsRecord(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "stars", 40000001,
		`{"_key": 40000001, "age": 14262808228, "luminosity": 0.01575, "name": "Tanoo - Star", "radius": 63350000, "solarSystemId": 30000001, "spectralClass": "K2 V", "temperature": 4567, "typeId": 45041}`)

	got, ok, err := store.Star(context.Background(), "test-agent", 40000001)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if !ok {
		t.Fatal("Star ok = false, want true")
	}
	if got.ID != 40000001 || got.Name != "Tanoo - Star" || got.SolarSystemID != 30000001 || got.TypeID != 45041 {
		t.Errorf("got = %+v, want ID=40000001 Name=\"Tanoo - Star\" SolarSystemID=30000001 TypeID=45041", got)
	}
	if got.Age != 14262808228 || got.Luminosity != 0.01575 || got.Radius != 63350000 || got.SpectralClass != "K2 V" || got.Temperature != 4567 {
		t.Errorf("got = %+v, want Age=14262808228 Luminosity=0.01575 Radius=63350000 SpectralClass=\"K2 V\" Temperature=4567", got)
	}
}

func TestStar_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Star(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if ok {
		t.Error("Star ok = true, want false for an unknown ID")
	}
}

func TestStar_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "stars", 40000001,
		`{"_key": 999, "age": 1, "luminosity": 0.1, "name": "X", "radius": 1, "solarSystemId": 1, "spectralClass": "K2 V", "temperature": 1, "typeId": 1}`)

	_, _, err := store.Star(context.Background(), "test-agent", 40000001)
	if err == nil {
		t.Fatal("Star returned nil error, want an error for a mismatched _key")
	}
}

func TestPutStar_ThenStar_RoundTrips(t *testing.T) {
	store := seedStore(t)
	want := Star{ID: 40000199, Age: 1, Luminosity: 0.5, Name: "New Star", Radius: 100, SolarSystemID: 30000099, SpectralClass: "G2 V", Temperature: 5000, TypeID: 45041}

	if err := store.PutStar(want); err != nil {
		t.Fatalf("PutStar returned error: %v", err)
	}

	got, ok, err := store.Star(context.Background(), "test-agent", 40000199)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if !ok {
		t.Fatal("Star ok = false after PutStar, want true")
	}
	if got != want {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
