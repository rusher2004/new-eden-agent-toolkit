package main

import (
	"bytes"
	"context"
	"os"
	"testing"
)

// TestToolDocsUpToDate fails when docs/reference/tools.md no longer matches
// the tools neat registers.
func TestToolDocsUpToDate(t *testing.T) {
	want, err := render(context.Background())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	got, err := os.ReadFile("../../docs/reference/tools.md")
	if err != nil {
		t.Fatalf("read committed tool reference: %v", err)
	}
	// Git on Windows may check the file out with CRLF line endings.
	got = bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(got, want) {
		t.Error("docs/reference/tools.md is stale; run `go generate ./...` and commit the result")
	}
}

func TestAreaOf(t *testing.T) {
	tests := []struct {
		tool, want string
	}{
		{"add_character", "Characters and login"},
		{"plan_build", "Industry planning"},
		{"get_universe_planet", "Universe and static data"},
		{"list_corporation_wallets", "Wallet"},
		{"list_corporation_killmails_recent", "Killmails and warfare"},
		{"list_corporation_mining_observers", "Industry, mining and planets"},
		{"list_corporation_container_logs", "Corporations and alliances"},
		{"get_character_location", "Character details"},
		{"get_status", "Server and ESI status"},
		{"open_market_details_window", "Game client"},
		{"list_character_mail", "Mail, calendar, contacts and notifications"},
		{"list_character_killmails_recent", "Killmails and warfare"},
	}
	for _, tt := range tests {
		i := areaOf(tt.tool)
		if i < 0 {
			t.Errorf("areaOf(%q) = none, want %q", tt.tool, tt.want)
			continue
		}
		if got := areas[i].title; got != tt.want {
			t.Errorf("areaOf(%q) = %q, want %q", tt.tool, got, tt.want)
		}
	}
	if got := areaOf("frobnicate"); got != -1 {
		t.Errorf("areaOf(unknown) = %d, want -1", got)
	}
}
