package sde

import (
	"context"
	"testing"
)

func TestSkinrComponent_KnownID_ReturnsName(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO skinrComponents (componentID, name, categoryID, finish, published) VALUES (1254, 'Charcoal Covenant Satin', 1, 'Satin', 1)`)

	got, ok, err := store.SkinrComponent(context.Background(), testUserAgent, 1254)
	if err != nil {
		t.Fatalf("SkinrComponent returned error: %v", err)
	}
	if !ok {
		t.Fatal("SkinrComponent ok = false, want true")
	}
	if got.ID != 1254 || got.Name != "Charcoal Covenant Satin" {
		t.Errorf("got = %+v, want ID=1254 Name=\"Charcoal Covenant Satin\"", got)
	}
}

func TestSkinrComponent_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.SkinrComponent(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("SkinrComponent returned error: %v", err)
	}
	if ok {
		t.Error("SkinrComponent ok = true, want false for an unknown ID")
	}
}

func TestSkinrComponent_NilStore_ReturnsNotOK(t *testing.T) {
	var store *Store
	_, ok, err := store.SkinrComponent(context.Background(), testUserAgent, 1254)
	if err != nil || ok {
		t.Errorf("SkinrComponent on nil Store = (ok=%v, err=%v), want (false, nil)", ok, err)
	}
}

func TestSkinrSlot_KnownID_ReturnsName(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO skinrSlots (slotID, name, categoryID) VALUES (5, 'Primary Pattern Slot', 2)`)

	got, ok, err := store.SkinrSlot(context.Background(), testUserAgent, 5)
	if err != nil {
		t.Fatalf("SkinrSlot returned error: %v", err)
	}
	if !ok {
		t.Fatal("SkinrSlot ok = false, want true")
	}
	if got.ID != 5 || got.Name != "Primary Pattern Slot" {
		t.Errorf("got = %+v, want ID=5 Name=\"Primary Pattern Slot\"", got)
	}
}

func TestSkinrSlot_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.SkinrSlot(context.Background(), testUserAgent, 99)
	if err != nil {
		t.Fatalf("SkinrSlot returned error: %v", err)
	}
	if ok {
		t.Error("SkinrSlot ok = true, want false for an unknown ID")
	}
}

func TestSkinrSlot_NilStore_ReturnsNotOK(t *testing.T) {
	var store *Store
	_, ok, err := store.SkinrSlot(context.Background(), testUserAgent, 5)
	if err != nil || ok {
		t.Errorf("SkinrSlot on nil Store = (ok=%v, err=%v), want (false, nil)", ok, err)
	}
}
