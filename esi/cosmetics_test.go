package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterSkinrLicenses_UsesOwnCharacterID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"licenses":[{"skinr_id":"abc123","activated":true,"unactivated":2}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	licenses, err := client.CharacterSkinrLicenses(context.Background())
	if err != nil {
		t.Fatalf("CharacterSkinrLicenses returned error: %v", err)
	}
	if gotPath != "/characters/998877/cosmetics/skinr" {
		t.Errorf("path = %q, want it to reflect the client's own characterID 998877", gotPath)
	}
	if len(licenses) != 1 || licenses[0].SkinrId != "abc123" || !licenses[0].Activated || licenses[0].Unactivated != 2 {
		t.Errorf("licenses = %+v, want one activated abc123 license with 2 unactivated", licenses)
	}
}

// TestCharacterSkinrComponentLicenses_DecodesBothRunsVariants guards the generated oneOf decoder:
// Runs is either {"remaining": n} or {"unlimited": true}, and each must land in exactly its own
// variant rather than failing with "matches more than one schema".
func TestCharacterSkinrComponentLicenses_DecodesBothRunsVariants(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"licenses":[{"component_id":1254,"type":"material","runs":{"remaining":3}},{"component_id":1328,"type":"pattern","runs":{"unlimited":true}}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	licenses, err := client.CharacterSkinrComponentLicenses(context.Background())
	if err != nil {
		t.Fatalf("CharacterSkinrComponentLicenses returned error: %v", err)
	}
	if gotPath != "/characters/998877/cosmetics/skinr/components" {
		t.Errorf("path = %q, want /characters/998877/cosmetics/skinr/components", gotPath)
	}
	if len(licenses) != 2 {
		t.Fatalf("licenses = %+v, want 2 entries", licenses)
	}
	if r := licenses[0].Runs; r.Remaining == nil || r.Remaining.Remaining == nil || *r.Remaining.Remaining != 3 || r.Unlimited != nil {
		t.Errorf("licenses[0].Runs = %+v, want only Remaining=3", r)
	}
	if r := licenses[1].Runs; r.Unlimited == nil || r.Unlimited.Unlimited == nil || !*r.Unlimited.Unlimited || r.Remaining != nil {
		t.Errorf("licenses[1].Runs = %+v, want only Unlimited=true", r)
	}
}

func TestSkinrDetail_DecodesNanocoatingAndPatternSlots(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"381156f5","name":"Semper Fi","line":"Combat","creator_id":2119410889,"ship_type_id":78576,"tier":{"level":13},"layout":{"slots":[{"id":1,"configuration":{"nanocoating":{"id":1254}}},{"id":5,"configuration":{"pattern":{"id":1328,"configuration":{"projection":{"slot1":false,"slot2":true,"slot3":false,"slot4":true},"transform":{"position":{"x":-8685.0,"y":101.3,"z":2315.0},"rotation":{"x":-0.707,"y":0.014,"z":-0.002,"w":-0.707},"scaling":{"x":829.9,"y":829.9,"z":829.9}},"mirrored":true}}}}],"pattern_blend_mode":"normal"}}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	s, err := client.SkinrDetail(context.Background(), "381156f5")
	if err != nil {
		t.Fatalf("SkinrDetail returned error: %v", err)
	}
	if gotPath != "/cosmetics/skinr/381156f5" {
		t.Errorf("path = %q, want /cosmetics/skinr/381156f5", gotPath)
	}
	if s.Name != "Semper Fi" || s.Line == nil || *s.Line != "Combat" || s.ShipTypeId != 78576 || s.Tier.Level != 13 || s.Layout.PatternBlendMode != "normal" {
		t.Errorf("skinr = %+v, want Semper Fi / Combat / ship 78576 / tier 13 / normal blend", s)
	}
	if len(s.Layout.Slots) != 2 {
		t.Fatalf("slots = %+v, want 2", s.Layout.Slots)
	}
	if cfg := s.Layout.Slots[0].Configuration; cfg.Nanocoating == nil || cfg.Nanocoating.Nanocoating == nil || cfg.Nanocoating.Nanocoating.Id != 1254 || cfg.Pattern != nil {
		t.Errorf("slot 1 configuration = %+v, want only nanocoating 1254", cfg)
	}
	cfg := s.Layout.Slots[1].Configuration
	if cfg.Pattern == nil || cfg.Pattern.Pattern == nil || cfg.Nanocoating != nil {
		t.Fatalf("slot 5 configuration = %+v, want only a pattern", cfg)
	}
	if p := cfg.Pattern.Pattern; p.Id != 1328 || !p.Configuration.Mirrored || !p.Configuration.Projection.Slot2 || p.Configuration.Transform.Rotation.W != -0.707 {
		t.Errorf("pattern = %+v, want 1328 mirrored, projected on slot2, rotation.w=-0.707", p)
	}
}

func TestSkinrDetail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, err := client.SkinrDetail(context.Background(), "missing"); err == nil {
		t.Fatal("SkinrDetail returned nil error, want an error for HTTP 404")
	}
}
