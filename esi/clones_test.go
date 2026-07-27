package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterClones_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"home_location":{"location_id":60003760,"location_type":"station"},"jump_clones":[{"implants":[19540],"jump_clone_id":1,"location_id":60003760,"location_type":"station"}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	clones, err := client.CharacterClones(context.Background())
	if err != nil {
		t.Fatalf("CharacterClones returned error: %v", err)
	}
	if clones.HomeLocation == nil || clones.HomeLocation.LocationId == nil || *clones.HomeLocation.LocationId != 60003760 {
		t.Errorf("clones.HomeLocation = %+v, want LocationId=*60003760", clones.HomeLocation)
	}
	if len(clones.JumpClones) != 1 || clones.JumpClones[0].JumpCloneId != 1 {
		t.Errorf("clones.JumpClones = %+v, want one entry with JumpCloneId=1", clones.JumpClones)
	}
}

func TestCharacterClones_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if _, err := client.CharacterClones(context.Background()); err == nil {
		t.Fatal("CharacterClones returned nil error, want an error for HTTP 403")
	}
}

func TestCharacterImplants_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[19540,19551]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	implants, err := client.CharacterImplants(context.Background())
	if err != nil {
		t.Fatalf("CharacterImplants returned error: %v", err)
	}
	if len(implants) != 2 || implants[0] != 19540 {
		t.Errorf("implants = %v, want [19540 19551]", implants)
	}
}
