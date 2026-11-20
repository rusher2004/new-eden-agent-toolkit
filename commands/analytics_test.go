package commands_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func runAnalytics(t *testing.T, arg string) string {
	t.Helper()
	var out bytes.Buffer
	root := commands.Root("dev", "test-client-id", "")
	root.Writer = &out
	if err := root.Run(context.Background(), []string{"neat", "analytics", arg}); err != nil {
		t.Fatalf("neat analytics %s: %v", arg, err)
	}
	return out.String()
}

func TestAnalyticsCommand_OffOnStatus(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	idPath := filepath.Join(tmpHome, ".neat", "analytics.json")
	store := config.NewStoreAt(filepath.Join(tmpHome, ".neat", "config.json"))

	if got := runAnalytics(t, "status"); !strings.Contains(got, "is on") {
		t.Errorf("default status = %q, want on", got)
	}

	if err := os.MkdirAll(filepath.Dir(idPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(idPath, []byte(`{"month":"2026-11","id":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runAnalytics(t, "off")
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AnalyticsDisabled {
		t.Error("AnalyticsDisabled = false after `analytics off`")
	}
	if _, err := os.Stat(idPath); !os.IsNotExist(err) {
		t.Errorf("monthly id file still exists after `analytics off`: %v", err)
	}
	if got := runAnalytics(t, "status"); !strings.Contains(got, "is off") {
		t.Errorf("status after off = %q", got)
	}

	runAnalytics(t, "on")
	cfg, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AnalyticsDisabled {
		t.Error("AnalyticsDisabled = true after `analytics on`")
	}
}
