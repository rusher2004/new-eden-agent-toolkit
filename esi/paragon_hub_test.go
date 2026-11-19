package esi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

// listingsJSON holds one ISK-priced and one PLEX-priced listing, trimmed from a real
// /paragon-hub/skinr response.
const listingsJSON = `{"cursor":{"before":"1.MTc5NTAzNjA1MjQ3Ny0w","after":"1.MTc5NTA0NTQ3MjQyOC0w"},"listings":[` +
	`{"id":"l-1","state":"active","last_modified":"2026-11-18T21:07:32Z","seller_id":2119410889,"skinr_id":"s-1","created":"2026-08-20T21:07:14Z","expires":"2026-11-18T21:07:14Z","quantity":1,"price":{"plex":3899}},` +
	`{"id":"l-2","state":"sold_out","last_modified":"2026-11-18T21:24:10Z","seller_id":2123243973,"skinr_id":"s-2","created":"2026-11-16T17:01:55Z","expires":"2027-02-14T17:01:55Z","quantity":0,"price":{"isk":125000000.5}}]}`

func TestParagonHubSkinrListings_DecodesBothPriceVariantsAndSendsCursor(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(listingsJSON))
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	listings, cursor, err := client.ParagonHubSkinrListings(context.Background(), "", "cur-b", 50)
	if err != nil {
		t.Fatalf("ParagonHubSkinrListings returned error: %v", err)
	}
	if gotPath != "/paragon-hub/skinr" {
		t.Errorf("path = %q, want /paragon-hub/skinr", gotPath)
	}
	if gotQuery.Has("after") || gotQuery.Get("before") != "cur-b" || gotQuery.Get("limit") != "50" {
		t.Errorf("query = %v, want before=cur-b limit=50 and no after", gotQuery)
	}
	if len(listings) != 2 {
		t.Fatalf("listings = %+v, want 2", listings)
	}
	if p := listings[0].Price; p.Plex == nil || p.Plex.Plex == nil || *p.Plex.Plex != 3899 || p.Isk != nil {
		t.Errorf("listings[0].Price = %+v, want only Plex=3899", p)
	}
	if p := listings[1].Price; p.Isk == nil || p.Isk.Isk == nil || *p.Isk.Isk != 125000000.5 || p.Plex != nil {
		t.Errorf("listings[1].Price = %+v, want only Isk=125000000.5", p)
	}
	if cursor == nil || cursor.Before == nil || *cursor.Before != "1.MTc5NTAzNjA1MjQ3Ny0w" {
		t.Errorf("cursor = %+v, want Before=1.MTc5NTAzNjA1MjQ3Ny0w", cursor)
	}
}

// TestCharacterParagonHubSkinrListings_DecodesTargetVariants guards the 4-variant Target oneOf
// that only a character's own posted listings carry.
func TestCharacterParagonHubSkinrListings_DecodesTargetVariants(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"listings":[` +
			`{"id":"l-1","state":"active","last_modified":"2026-11-18T00:00:00Z","seller_id":998877,"skinr_id":"s-1","created":"2026-11-01T00:00:00Z","expires":"2027-01-01T00:00:00Z","quantity":3,"price":{"isk":1000},"target":{"public":true}},` +
			`{"id":"l-2","state":"active","last_modified":"2026-11-18T00:00:00Z","seller_id":998877,"skinr_id":"s-2","created":"2026-11-01T00:00:00Z","expires":"2027-01-01T00:00:00Z","quantity":1,"price":{"plex":10},"target":{"corporation_id":98000001}}]}`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	listings, _, err := client.CharacterParagonHubSkinrListings(context.Background(), "", "", 0)
	if err != nil {
		t.Fatalf("CharacterParagonHubSkinrListings returned error: %v", err)
	}
	if gotPath != "/characters/998877/paragon-hub/skinr" {
		t.Errorf("path = %q, want it to reflect the client's own characterID 998877", gotPath)
	}
	if len(listings) != 2 {
		t.Fatalf("listings = %+v, want 2", listings)
	}
	if tgt := listings[0].Target; tgt.Public == nil || tgt.Public.Public == nil || !*tgt.Public.Public || tgt.CorporationId != nil {
		t.Errorf("listings[0].Target = %+v, want only Public=true", tgt)
	}
	if tgt := listings[1].Target; tgt.CorporationId == nil || tgt.CorporationId.CorporationId == nil || *tgt.CorporationId.CorporationId != 98000001 || tgt.Public != nil {
		t.Errorf("listings[1].Target = %+v, want only CorporationId=98000001", tgt)
	}
}

func TestParagonHubSkinrListingsForTargets_Paths(t *testing.T) {
	for _, tc := range []struct {
		name     string
		call     func(c *esi.Client) error
		wantPath string
	}{
		{"alliance", func(c *esi.Client) error {
			_, _, err := c.ParagonHubSkinrListingsForAlliance(context.Background(), 99000001, "", "", 0)
			return err
		}, "/paragon-hub/skinr/alliances/99000001"},
		{"corporation", func(c *esi.Client) error {
			_, _, err := c.ParagonHubSkinrListingsForCorporation(context.Background(), 98000001, "", "", 0)
			return err
		}, "/paragon-hub/skinr/corporations/98000001"},
		{"character (own)", func(c *esi.Client) error {
			_, _, err := c.ParagonHubSkinrListingsForCharacter(context.Background(), "", "", 0)
			return err
		}, "/paragon-hub/skinr/characters/998877"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(listingsJSON))
			}))
			defer srv.Close()

			client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
			if err := tc.call(client); err != nil {
				t.Fatalf("call returned error: %v", err)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}

func TestParagonHubSkinrListings_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	client := esi.ExportedNewPublicClientWithOptions(srv.Client(), goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})
	if _, _, err := client.ParagonHubSkinrListings(context.Background(), "", "", 5); err == nil {
		t.Fatal("ParagonHubSkinrListings returned nil error, want an error for HTTP 422")
	}
}
