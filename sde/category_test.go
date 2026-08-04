package sde

import (
	"context"
	"testing"
)

func TestCategory_KnownID_ReturnsRowWithGroupIDs(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invCategories (categoryID, categoryName, published) VALUES (6, 'Ship', 1)`)
	seedExec(t, store, `INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (25, 6, 'Frigate', 1)`)
	seedExec(t, store, `INSERT INTO invGroups (groupID, categoryID, groupName, published) VALUES (26, 6, 'Cruiser', 1)`)

	got, ok, err := store.Category(context.Background(), testUserAgent, 6)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok {
		t.Fatal("Category ok = false, want true")
	}
	if got.ID != 6 || got.Name != "Ship" || !got.Published {
		t.Errorf("got = %+v, want ID=6 Name=Ship Published=true", got)
	}
	if len(got.GroupIDs) != 2 || got.GroupIDs[0] != 25 || got.GroupIDs[1] != 26 {
		t.Errorf("GroupIDs = %v, want [25 26]", got.GroupIDs)
	}
}

func TestCategory_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Category(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if ok {
		t.Error("Category ok = true, want false for an unknown ID")
	}
}

func TestPutCategory_ThenCategory_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	want := Category{ID: 999, Name: "Read-Through Category", Published: true, GroupIDs: []int64{1, 2}}

	if err := store.PutCategory(want); err != nil {
		t.Fatalf("PutCategory returned error: %v", err)
	}
	got, ok, err := store.Category(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Category returned error: %v", err)
	}
	if !ok {
		t.Fatal("Category ok = false, want true after PutCategory")
	}
	if got.Name != want.Name || len(got.GroupIDs) != 2 {
		t.Errorf("Category = %+v, want %+v", got, want)
	}
}

func TestCategoryIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO invCategories (categoryID, categoryName, published) VALUES (6, 'Ship', 1)`)
	seedExec(t, store, `INSERT INTO invCategories (categoryID, categoryName, published) VALUES (7, 'Module', 1)`)

	got, err := store.CategoryIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("CategoryIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 6 || got[1] != 7 {
		t.Errorf("CategoryIDs = %v, want [6 7]", got)
	}
}
