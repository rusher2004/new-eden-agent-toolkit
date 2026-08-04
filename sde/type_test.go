package sde

import (
	"context"
	"testing"
)

func TestType_KnownID_ReturnsRow(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, description, published) VALUES (587, 25, 'Rifter', 'A fast frigate.', 1)`)
	seedExec(t, store, `INSERT INTO dgmTypeAttributes (typeID, attributeID, valueFloat) VALUES (587, 182, 3386.0)`)
	seedExec(t, store, `INSERT INTO dgmTypeAttributes (typeID, attributeID, valueInt) VALUES (587, 4, 1)`)
	seedExec(t, store, `INSERT INTO dgmTypeEffects (typeID, effectID, isDefault) VALUES (587, 4, 1)`)
	store.typeVolumes = map[int64]float64{587: 2.5}

	got, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.ID != 587 || got.GroupID != 25 || !got.Published {
		t.Errorf("got = %+v, want ID=587 GroupID=25 Published=true", got)
	}
	if got.Name != "Rifter" {
		t.Errorf("Name = %q, want Rifter", got.Name)
	}
	if got.Description != "A fast frigate." {
		t.Errorf("Description = %q, want %q", got.Description, "A fast frigate.")
	}
	if got.PackagedVolume == nil || *got.PackagedVolume != 2.5 {
		t.Errorf("PackagedVolume = %v, want 2.5", got.PackagedVolume)
	}
	if len(got.DogmaAttributes) != 2 {
		t.Fatalf("len(DogmaAttributes) = %d, want 2", len(got.DogmaAttributes))
	}
	if len(got.DogmaEffects) != 1 || got.DogmaEffects[0].EffectID != 4 || !got.DogmaEffects[0].IsDefault {
		t.Errorf("DogmaEffects = %+v, want one {EffectID:4 IsDefault:true}", got.DogmaEffects)
	}
}

func TestType_NullNameAndPublished_DegradeToZeroValueInsteadOfErroring(t *testing.T) {
	// typeName/published are nullable in the fuzzwork schema (PRAGMA
	// table_info confirmed notnull=0) even though every real row today
	// happens to have both populated. A NULL in either must not turn a
	// cache hit into a hard scan error -- see PR #14 review finding #2.
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (587, 25, NULL, NULL)`)

	got, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil {
		t.Fatalf("Type returned error: %v, want nil (NULL name/published should degrade gracefully)", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true")
	}
	if got.Name != "" || got.Published {
		t.Errorf("Name/Published = %q/%v, want empty string/false", got.Name, got.Published)
	}
}

func TestType_DogmaAttribute_PrefersValueFloatOverValueInt(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (587, 25, 'Rifter', 1)`)
	seedExec(t, store, `INSERT INTO dgmTypeAttributes (typeID, attributeID, valueFloat) VALUES (587, 182, 3386.5)`)

	got, _, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if len(got.DogmaAttributes) != 1 || got.DogmaAttributes[0].Value != 3386.5 {
		t.Errorf("DogmaAttributes = %+v, want one {AttributeID:182 Value:3386.5}", got.DogmaAttributes)
	}
}

func TestType_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Type(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if ok {
		t.Error("Type ok = true, want false for an unknown ID")
	}
}

func TestType_NilStore_ReturnsNotOK(t *testing.T) {
	var store *Store
	_, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil {
		t.Fatalf("Type on nil Store returned error: %v", err)
	}
	if ok {
		t.Error("Type on nil Store ok = true, want false")
	}
}

func TestType_NeverSynced_ReturnsNotOK(t *testing.T) {
	store := &Store{}
	_, ok, err := store.Type(context.Background(), testUserAgent, 587)
	if err != nil {
		t.Fatalf("Type on never-synced Store returned error: %v", err)
	}
	if ok {
		t.Error("Type on never-synced Store ok = true, want false")
	}
}

func TestPutType_ThenType_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Type{ID: 999, GroupID: 25, Name: "Read-Through Rifter", Published: true}

	if err := store.PutType(want); err != nil {
		t.Fatalf("PutType returned error: %v", err)
	}
	got, ok, err := store.Type(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Type returned error: %v", err)
	}
	if !ok {
		t.Fatal("Type ok = false, want true after PutType")
	}
	if got.ID != want.ID || got.GroupID != want.GroupID || got.Name != want.Name || got.Published != want.Published {
		t.Errorf("Type = %+v, want %+v", got, want)
	}
}

func TestTypeIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (587, 25, 'Rifter', 1)`)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (588, 25, 'Slasher', 1)`)

	got, err := store.TypeIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("TypeIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 587 || got[1] != 588 {
		t.Errorf("TypeIDs = %v, want [587 588]", got)
	}
}
