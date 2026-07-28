package sde

import (
	"context"
	"testing"
)

func TestAncestries_ReturnsAll(t *testing.T) {
	store := seedStore(t)
	shortDesc := "Progressive members of the upper class."
	putRecord(t, store, "ancestries", 1,
		`{"_key": 1, "bloodlineID": 5, "description": {"en": "Holders are..."}, "iconID": 1641, "name": {"en": "Liberal Holders"}, "shortDescription": "Progressive members of the upper class."}`)

	got, err := store.Ancestries(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Ancestries returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Ancestries = %+v, want one entry", got)
	}
	a := got[0]
	if a.ID != 1 || a.BloodlineID != 5 {
		t.Errorf("got = %+v, want ID=1 BloodlineID=5", a)
	}
	if a.Name["en"] != "Liberal Holders" {
		t.Errorf("Name[en] = %q, want \"Liberal Holders\"", a.Name["en"])
	}
	if a.IconID == nil || *a.IconID != 1641 {
		t.Errorf("IconID = %v, want 1641", a.IconID)
	}
	if a.ShortDescription == nil || *a.ShortDescription != shortDesc {
		t.Errorf("ShortDescription = %v, want %q", a.ShortDescription, shortDesc)
	}
}

func TestAncestries_EmptyBucket_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	got, err := store.Ancestries(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("Ancestries returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Ancestries = %+v, want empty", got)
	}
}
