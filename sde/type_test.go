package sde

import (
	"context"
	"testing"
)

func TestType_KnownID_ReturnsAllLocales(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587,
		`{"_key": 587, "groupID": 25, "name": {"en": "Rifter", "de": "Rifter"}, "published": true}`)

	got, ok, err := store.Type(context.Background(), "test-agent", 587)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.ID != 587 || got.GroupID != 25 || !got.Published {
		t.Errorf("got = %+v, want ID=587 GroupID=25 Published=true", got)
	}
	if got.Name["en"] != "Rifter" || got.Name["de"] != "Rifter" {
		t.Errorf("Name = %+v, want en=Rifter de=Rifter", got.Name)
	}
}

func TestType_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Type(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if ok {
		t.Error("Type ok = true, want false for an unknown ID")
	}
}

func TestType_MalformedCachedRecord_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587, `not valid json`)

	_, _, err := store.Type(context.Background(), "test-agent", 587)
	if err == nil {
		t.Fatal("Type returned nil error, want a parse error for a corrupted cache entry")
	}
}

func TestType_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587,
		`{"_key": 12345, "groupID": 25, "name": {"en": "Rifter"}, "published": true}`)

	_, _, err := store.Type(context.Background(), "test-agent", 587)
	if err == nil {
		t.Fatal("Type returned nil error, want an error for a mismatched _key")
	}
}

func TestType_KnownID_ReturnsFullFieldSetAndDogma(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587,
		`{"_key": 587, "capacity": 140.0, "description": {"en": "The Rifter is fast."}, "graphicID": 46, "groupID": 25, "iconID": 100, "marketGroupID": 64, "mass": 1067000.0, "name": {"en": "Rifter"}, "portionSize": 1, "published": true, "radius": 31.0, "volume": 27289.0, "dogmaAttributes": [{"attributeID": 182, "value": 3386.0}], "dogmaEffects": [{"effectID": 4, "isDefault": true}]}`)

	got, ok, err := store.Type(context.Background(), "test-agent", 587)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.Description["en"] != "The Rifter is fast." {
		t.Errorf("Description[en] = %q, want %q", got.Description["en"], "The Rifter is fast.")
	}
	if got.Capacity == nil || *got.Capacity != 140.0 {
		t.Errorf("Capacity = %v, want 140.0", got.Capacity)
	}
	if got.GraphicID == nil || *got.GraphicID != 46 {
		t.Errorf("GraphicID = %v, want 46", got.GraphicID)
	}
	if got.IconID == nil || *got.IconID != 100 {
		t.Errorf("IconID = %v, want 100", got.IconID)
	}
	if got.MarketGroupID == nil || *got.MarketGroupID != 64 {
		t.Errorf("MarketGroupID = %v, want 64", got.MarketGroupID)
	}
	if got.Mass == nil || *got.Mass != 1067000.0 {
		t.Errorf("Mass = %v, want 1067000.0", got.Mass)
	}
	if got.PortionSize == nil || *got.PortionSize != 1 {
		t.Errorf("PortionSize = %v, want 1", got.PortionSize)
	}
	if got.Radius == nil || *got.Radius != 31.0 {
		t.Errorf("Radius = %v, want 31.0", got.Radius)
	}
	if got.Volume == nil || *got.Volume != 27289.0 {
		t.Errorf("Volume = %v, want 27289.0", got.Volume)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0].AttributeID != 182 || got.DogmaAttributes[0].Value != 3386.0 {
		t.Errorf("DogmaAttributes = %+v, want one entry {AttributeID:182 Value:3386.0}", got.DogmaAttributes)
	}
	if len(got.DogmaEffects) != 1 || got.DogmaEffects[0].EffectID != 4 || !got.DogmaEffects[0].IsDefault {
		t.Errorf("DogmaEffects = %+v, want one entry {EffectID:4 IsDefault:true}", got.DogmaEffects)
	}
}

func TestType_NoDogmaEffects_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 588,
		`{"_key": 588, "groupID": 25, "name": {"en": "No-effects Type"}, "published": true, "dogmaAttributes": [{"attributeID": 1, "value": 1.0}]}`)

	got, ok, err := store.Type(context.Background(), "test-agent", 588)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if len(got.DogmaEffects) != 0 {
		t.Errorf("DogmaEffects = %+v, want empty", got.DogmaEffects)
	}
}

func TestType_KnownID_ReturnsPackagedVolume(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587,
		`{"_key": 587, "groupID": 25, "name": {"en": "Rifter"}, "published": true, "packagedVolume": 2500}`)

	got, ok, err := store.Type(context.Background(), "test-agent", 587)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.PackagedVolume == nil || *got.PackagedVolume != 2500 {
		t.Errorf("PackagedVolume = %v, want 2500", got.PackagedVolume)
	}
}

func TestType_NoPackagedVolume_ReturnsNil(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 588,
		`{"_key": 588, "groupID": 25, "name": {"en": "No Volume"}, "published": true}`)

	got, ok, err := store.Type(context.Background(), "test-agent", 588)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.PackagedVolume != nil {
		t.Errorf("PackagedVolume = %v, want nil", got.PackagedVolume)
	}
}

func TestTypeIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "types", 587, `{"_key": 587, "groupID": 25, "name": {"en": "Rifter"}, "published": true}`)
	putRecord(t, store, "types", 588, `{"_key": 588, "groupID": 25, "name": {"en": "Breacher"}, "published": true}`)

	ids, err := store.TypeIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("TypeIDs returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 587 || ids[1] != 588 {
		t.Errorf("TypeIDs = %v, want [587 588] (ascending)", ids)
	}
}

func TestTypeIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.TypeIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("TypeIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("TypeIDs = %v, want empty", ids)
	}
}

func TestPutType_ThenType_RoundTrips(t *testing.T) {
	store := seedStore(t)
	mass := 1000.0
	packagedVolume := 500.0
	want := Type{ID: 999, GroupID: 25, Name: map[string]string{"en": "New Type"}, Description: map[string]string{"en": "A new type."}, Published: true, Mass: &mass, PackagedVolume: &packagedVolume, DogmaAttributes: []TypeDogmaAttribute{{AttributeID: 1, Value: 2.0}}}

	if err := store.PutType(want); err != nil {
		t.Fatalf("PutType returned error: %v", err)
	}

	got, ok, err := store.Type(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false after PutType, want true")
	}
	if got.ID != want.ID || got.GroupID != want.GroupID || *got.Mass != *want.Mass || got.Description["en"] != want.Description["en"] {
		t.Errorf("got = %+v, want %+v", got, want)
	}
	if got.PackagedVolume == nil || *got.PackagedVolume != packagedVolume {
		t.Errorf("PackagedVolume = %v, want %v", got.PackagedVolume, packagedVolume)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0] != want.DogmaAttributes[0] {
		t.Errorf("DogmaAttributes = %+v, want %+v", got.DogmaAttributes, want.DogmaAttributes)
	}
}
