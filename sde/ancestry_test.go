package sde

import (
	"context"
	"testing"
)

func TestAncestries_ReturnsEveryCachedAncestry(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO chrAncestries (ancestryID, ancestryName, bloodlineID, description, iconID, shortDescription)
		VALUES (1, 'Immigrant', 1, 'An immigrant.', 100, 'Short.')`)

	got, err := store.Ancestries(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Ancestries returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(Ancestries) = %d, want 1", len(got))
	}
	a := got[0]
	if a.ID != 1 || a.Name != "Immigrant" || a.BloodlineID != 1 || a.Description != "An immigrant." {
		t.Errorf("got = %+v, want ID=1 Name=Immigrant BloodlineID=1 Description=%q", a, "An immigrant.")
	}
	if a.IconID == nil || *a.IconID != 100 {
		t.Errorf("IconID = %v, want 100", a.IconID)
	}
	if a.ShortDescription == nil || *a.ShortDescription != "Short." {
		t.Errorf("ShortDescription = %v, want Short.", a.ShortDescription)
	}
}

func TestAncestries_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := &Store{}
	got, err := store.Ancestries(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("Ancestries returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Ancestries = %v, want empty", got)
	}
}
