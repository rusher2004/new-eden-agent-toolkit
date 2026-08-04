package sde

import (
	"context"
	"testing"
)

func TestGroup_KnownID_ReturnsRowWithTypeIDs(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (25, 6, 'Frigate', 1)`)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (587, 25, 'Rifter', 1)`)
	seedExec(t, store, `INSERT INTO invTypes (typeID, groupID, typeName, published) VALUES (588, 25, 'Slasher', 1)`)

	got, ok, err := store.Group(context.Background(), testUserAgent, 25)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if !ok {
		t.Fatal("Group ok = false, want true")
	}
	if got.ID != 25 || got.CategoryID != 6 || got.Name != "Frigate" || !got.Published {
		t.Errorf("got = %+v, want ID=25 CategoryID=6 Name=Frigate Published=true", got)
	}
	if len(got.TypeIDs) != 2 || got.TypeIDs[0] != 587 || got.TypeIDs[1] != 588 {
		t.Errorf("TypeIDs = %v, want [587 588]", got.TypeIDs)
	}
}

func TestGroup_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Group(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if ok {
		t.Error("Group ok = true, want false for an unknown ID")
	}
}

func TestPutGroup_ThenGroup_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Group{ID: 999, CategoryID: 6, Name: "Read-Through Group", Published: true, TypeIDs: []int64{1, 2}}

	if err := store.PutGroup(want); err != nil {
		t.Fatalf("PutGroup returned error: %v", err)
	}
	got, ok, err := store.Group(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Group returned error: %v", err)
	}
	if !ok {
		t.Fatal("Group ok = false, want true after PutGroup")
	}
	if got.Name != want.Name || len(got.TypeIDs) != 2 {
		t.Errorf("Group = %+v, want %+v", got, want)
	}
}

func TestGroupIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (25, 6, 'Frigate', 1)`)
	seedExec(t, store, `INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (26, 6, 'Cruiser', 1)`)

	got, err := store.GroupIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("GroupIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 25 || got[1] != 26 {
		t.Errorf("GroupIDs = %v, want [25 26]", got)
	}
}
