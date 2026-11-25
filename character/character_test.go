package character_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/character"
	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

type fakeLoader struct {
	cfg config.Config
	err error
}

func (f fakeLoader) Load() (config.Config, error) { return f.cfg, f.err }

func TestKnown_ReportsLoggedInCharacters(t *testing.T) {
	store := fakeLoader{cfg: config.Config{Characters: []config.Character{{ID: 7, Name: "Seven"}}}}

	got, err := character.Known(store, 7)
	if err != nil || !got {
		t.Fatalf("Known(7) = %v, %v; want true, nil", got, err)
	}
	got, err = character.Known(store, 8)
	if err != nil || got {
		t.Fatalf("Known(8) = %v, %v; want false, nil", got, err)
	}
}

func TestKnown_WrapsLoadError(t *testing.T) {
	boom := errors.New("disk on fire")

	got, err := character.Known(fakeLoader{err: boom}, 7)
	if got {
		t.Error("Known reported true on a load error")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap %v", err, boom)
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Errorf("err = %q, want it to name \"load config\"", err)
	}
}

func TestErrNotFound_Message(t *testing.T) {
	if got := character.ErrNotFound.Error(); got != "character not found" {
		t.Errorf("ErrNotFound.Error() = %q, want %q", got, "character not found")
	}
}
