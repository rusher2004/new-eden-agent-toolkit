package commands_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestSetupCommand_CreatesConfigAndVerifiesKeychain(t *testing.T) {
	keyring.MockInit()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	var out bytes.Buffer
	root := commands.Root("dev")
	root.Writer = &out

	if err := root.Run(context.Background(), []string{"neat", "setup"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	wantConfigPath := filepath.Join(tmpHome, ".neat", "config.json")
	if _, err := os.Stat(wantConfigPath); err != nil {
		t.Errorf("expected config file at %s, stat error: %v", wantConfigPath, err)
	}

	got := out.String()
	if !strings.Contains(got, "Config directory ready") {
		t.Errorf("output = %q, want it to mention config directory readiness", got)
	}
	if !strings.Contains(got, "Keychain access verified") {
		t.Errorf("output = %q, want it to mention keychain verification", got)
	}
}

func TestSetupCommand_BrokenKeychain_CreatesConfigButReportsError(t *testing.T) {
	wantErr := errors.New("keychain locked")
	keyring.MockInitWithError(wantErr)
	t.Cleanup(func() { keyring.MockInit() })

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	var out bytes.Buffer
	root := commands.Root("dev")
	root.Writer = &out

	err := root.Run(context.Background(), []string{"neat", "setup"})
	if err == nil {
		t.Fatal("Run returned nil error, want an error from the broken keychain")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}

	wantConfigPath := filepath.Join(tmpHome, ".neat", "config.json")
	if _, statErr := os.Stat(wantConfigPath); statErr != nil {
		t.Errorf("expected config file to exist at %s despite keychain failure, stat error: %v", wantConfigPath, statErr)
	}

	got := out.String()
	if !strings.Contains(got, "Config directory ready") {
		t.Errorf("output = %q, want it to mention config directory readiness before the keychain check runs", got)
	}
	if strings.Contains(got, "Keychain access verified") {
		t.Errorf("output = %q, want it to NOT reach keychain verification success", got)
	}
}

func TestSetupCommand_RunTwice_DoesNotClobberExistingConfig(t *testing.T) {
	keyring.MockInit()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	if err := commands.Root("dev").Run(context.Background(), []string{"neat", "setup"}); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}

	// Simulate a character already having been added between setup runs.
	store := config.NewStoreAt(filepath.Join(tmpHome, ".neat", "config.json"))
	want := config.Config{Characters: []config.Character{{ID: 1, Name: "Existing Pilot"}}}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := commands.Root("dev").Run(context.Background(), []string{"neat", "setup"}); err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(got.Characters) != 1 || got.Characters[0] != want.Characters[0] {
		t.Errorf("Characters = %v, want existing config preserved: %v", got.Characters, want.Characters)
	}
}
