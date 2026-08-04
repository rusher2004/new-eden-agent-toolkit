package sde

import (
	"context"
	"testing"
)

func TestGraphic_KnownID_MergesSofDNAFromInMemoryAugmentation(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO eveGraphics (graphicID, sofFactionName, graphicFile, sofHullName, sofRaceName, iconFolder)
		VALUES (2, 'caldari', 'res:/dx9/model/graphic.red', 'caldari_hull', 'caldari', 'icon/Folder/')`)
	store.graphicSofDNA = map[int64]string{2: "caldari:caldari:cal"}

	got, ok, err := store.Graphic(context.Background(), testUserAgent, 2)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true")
	}
	if got.ID != 2 || got.SofFactionName == nil || *got.SofFactionName != "caldari" {
		t.Errorf("got = %+v, want ID=2 SofFactionName=caldari", got)
	}
	if got.SofDNA == nil || *got.SofDNA != "caldari:caldari:cal" {
		t.Errorf("SofDNA = %v, want caldari:caldari:cal", got.SofDNA)
	}
}

func TestGraphic_NoAugmentation_ReturnsNilSofDNA(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO eveGraphics (graphicID, sofFactionName) VALUES (2, 'caldari')`)

	got, ok, err := store.Graphic(context.Background(), testUserAgent, 2)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok || got.SofDNA != nil {
		t.Errorf("Graphic = %+v, ok=%v, want SofDNA=nil ok=true", got, ok)
	}
}

func TestGraphic_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)
	_, ok, err := store.Graphic(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if ok {
		t.Error("Graphic ok = true, want false for an unknown ID")
	}
}

func TestPutGraphic_ThenGraphic_ReturnsOverlaidValue(t *testing.T) {
	store := seedStore(t)
	sofDNA := "read-through:dna"
	want := Graphic{ID: 999, SofDNA: &sofDNA}

	if err := store.PutGraphic(want); err != nil {
		t.Fatalf("PutGraphic returned error: %v", err)
	}
	got, ok, err := store.Graphic(context.Background(), testUserAgent, 999)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true after PutGraphic")
	}
	if got.SofDNA == nil || *got.SofDNA != sofDNA {
		t.Errorf("SofDNA = %v, want %q", got.SofDNA, sofDNA)
	}
}

func TestGraphicIDs_ReturnsEveryCachedID(t *testing.T) {
	store := seedStore(t)
	seedExec(t, store, `INSERT INTO eveGraphics (graphicID) VALUES (2)`)
	seedExec(t, store, `INSERT INTO eveGraphics (graphicID) VALUES (3)`)

	got, err := store.GraphicIDs(context.Background(), testUserAgent)
	if err != nil {
		t.Fatalf("GraphicIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("GraphicIDs = %v, want [2 3]", got)
	}
}
