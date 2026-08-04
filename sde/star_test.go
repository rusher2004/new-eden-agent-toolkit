package sde

import (
	"context"
	"testing"
)

func TestStar_KnownID_ReturnsRowWithCelestialStatistics(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (40009076, 3796, 6, 30000142, 'Jita - Sun')`)
	seedExec(t, store, `INSERT INTO mapCelestialStatistics (celestialID, temperature, spectralClass, luminosity, age, radius) VALUES (40009076, 7305.0, 'F1 V', 1.692, 21750000000000000.0, 696000000.0)`)

	got, ok, err := store.Star(context.Background(), testUserAgent, 40009076)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if !ok {
		t.Fatal("Star ok = false, want true")
	}
	if got.ID != 40009076 || got.TypeID != 3796 || got.SolarSystemID != 30000142 || got.Name != "Jita - Sun" {
		t.Errorf("got = %+v, want ID=40009076 TypeID=3796 SolarSystemID=30000142 Name=%q", got, "Jita - Sun")
	}
	if got.SpectralClass != "F1 V" || got.Luminosity != 1.692 || got.Temperature != 7305 {
		t.Errorf("SpectralClass/Luminosity/Temperature = %q/%v/%v, want F1 V/1.692/7305", got.SpectralClass, got.Luminosity, got.Temperature)
	}
}

func TestStar_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Star(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if ok {
		t.Error("Star ok = true, want false for an unknown ID")
	}
}

func TestStar_NoCelestialStatisticsRow_ReturnsZeroValuedPhysicalFields(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO mapDenormalize (itemID, typeID, groupID, solarSystemID, itemName) VALUES (40009076, 3796, 6, 30000142, 'Jita - Sun')`)

	got, ok, err := store.Star(context.Background(), testUserAgent, 40009076)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if !ok {
		t.Fatal("Star ok = false, want true even without a mapCelestialStatistics row")
	}
	if got.SpectralClass != "" || got.Luminosity != 0 {
		t.Errorf("SpectralClass/Luminosity = %q/%v, want zero values", got.SpectralClass, got.Luminosity)
	}
}

func TestPutStar_ThenStar_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Star{ID: 999, Name: "Read-Through Star", SolarSystemID: 30000142, TypeID: 3796}

	if err := store.PutStar(want); err != nil {
		t.Fatalf("PutStar returned error: %v", err)
	}
	got, ok, err := store.Star(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Star returned error: %v", err)
	}
	if !ok {
		t.Fatal("Star ok = false, want true after PutStar")
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}
