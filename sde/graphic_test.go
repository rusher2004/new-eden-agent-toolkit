package sde

import (
	"context"
	"testing"
)

func TestGraphic_KnownID_ReturnsFile(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 10, `{"_key": 10, "graphicFile": "res:/dx9/model/worldobject/planet/moon.red"}`)

	got, ok, err := store.Graphic(context.Background(), "test-agent", 10)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true")
	}
	if got.ID != 10 {
		t.Errorf("got.ID = %d, want 10", got.ID)
	}
	if got.GraphicFile == nil || *got.GraphicFile != "res:/dx9/model/worldobject/planet/moon.red" {
		t.Errorf("GraphicFile = %v, want \"res:/dx9/model/worldobject/planet/moon.red\"", got.GraphicFile)
	}
}

func TestGraphic_KnownID_ReturnsSOFFields(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 38,
		`{"_key": 38, "iconFolder": "res:/dx9/model/ship/caldari/frigate/cf1/icons", "sofFactionName": "caldaribase", "sofHullName": "cf1_t1", "sofRaceName": "caldari"}`)

	got, ok, err := store.Graphic(context.Background(), "test-agent", 38)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true")
	}
	if got.GraphicFile != nil {
		t.Errorf("GraphicFile = %v, want nil (this record has none)", got.GraphicFile)
	}
	if got.IconFolder == nil || *got.IconFolder != "res:/dx9/model/ship/caldari/frigate/cf1/icons" {
		t.Errorf("IconFolder = %v, want the caldari frigate icons path", got.IconFolder)
	}
	if got.SofFactionName == nil || *got.SofFactionName != "caldaribase" {
		t.Errorf("SofFactionName = %v, want caldaribase", got.SofFactionName)
	}
	if got.SofHullName == nil || *got.SofHullName != "cf1_t1" {
		t.Errorf("SofHullName = %v, want cf1_t1", got.SofHullName)
	}
	if got.SofRaceName == nil || *got.SofRaceName != "caldari" {
		t.Errorf("SofRaceName = %v, want caldari", got.SofRaceName)
	}
}

func TestGraphic_KnownID_ReturnsSofDNA(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 38,
		`{"_key": 38, "sofDna": "cf1_t1:caldaribase:caldari", "sofHullName": "cf1_t1", "sofRaceName": "caldari"}`)

	got, ok, err := store.Graphic(context.Background(), "test-agent", 38)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true")
	}
	if got.SofDNA == nil || *got.SofDNA != "cf1_t1:caldaribase:caldari" {
		t.Errorf("SofDNA = %v, want \"cf1_t1:caldaribase:caldari\"", got.SofDNA)
	}
}

func TestGraphic_NoSofDNA_ReturnsNil(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 10, `{"_key": 10, "graphicFile": "res:/x"}`)

	got, ok, err := store.Graphic(context.Background(), "test-agent", 10)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false, want true")
	}
	if got.SofDNA != nil {
		t.Errorf("SofDNA = %v, want nil", got.SofDNA)
	}
}

func TestGraphic_UnknownID_ReturnsNotOK(t *testing.T) {
	store := seedStore(t)

	_, ok, err := store.Graphic(context.Background(), "test-agent", 999)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if ok {
		t.Error("Graphic ok = true, want false for an unknown ID")
	}
}

func TestGraphic_MismatchedKey_ReturnsError(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 10, `{"_key": 999, "graphicFile": "res:/x"}`)

	_, _, err := store.Graphic(context.Background(), "test-agent", 10)
	if err == nil {
		t.Fatal("Graphic returned nil error, want an error for a mismatched _key")
	}
}

func TestGraphicIDs_ReturnsSortedIDs(t *testing.T) {
	store := seedStore(t)
	putRecord(t, store, "graphics", 38, `{"_key": 38, "sofRaceName": "caldari"}`)
	putRecord(t, store, "graphics", 10, `{"_key": 10, "graphicFile": "res:/x"}`)

	ids, err := store.GraphicIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("GraphicIDs returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10 || ids[1] != 38 {
		t.Errorf("GraphicIDs = %v, want [10 38] (ascending)", ids)
	}
}

func TestGraphicIDs_NeverSynced_ReturnsEmptySlice(t *testing.T) {
	store := seedStore(t)

	ids, err := store.GraphicIDs(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("GraphicIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("GraphicIDs = %v, want empty", ids)
	}
}

func TestPutGraphic_ThenGraphic_RoundTrips(t *testing.T) {
	store := seedStore(t)
	file := "res:/dx9/model/worldobject/newthing.red"
	race := "caldari"
	sofDNA := "cf1_t1:caldaribase:caldari"
	want := Graphic{ID: 99, GraphicFile: &file, SofRaceName: &race, SofDNA: &sofDNA}

	if err := store.PutGraphic(want); err != nil {
		t.Fatalf("PutGraphic returned error: %v", err)
	}

	got, ok, err := store.Graphic(context.Background(), "test-agent", 99)
	if err != nil {
		t.Fatalf("Graphic returned error: %v", err)
	}
	if !ok {
		t.Fatal("Graphic ok = false after PutGraphic, want true")
	}
	if got.ID != want.ID || *got.GraphicFile != *want.GraphicFile || *got.SofRaceName != *want.SofRaceName || *got.SofDNA != *want.SofDNA {
		t.Errorf("got = %+v, want %+v", got, want)
	}
}
