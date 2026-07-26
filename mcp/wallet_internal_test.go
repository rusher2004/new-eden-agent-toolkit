package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestGetWalletBalanceHandler_ReturnsBalance(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getWalletBalance
	t.Cleanup(func() { getWalletBalance = orig })
	getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
		return 42.5, nil
	}

	handler := getWalletBalanceHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, GetWalletBalanceInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CharacterID != 7 || out.BalanceISK != 42.5 {
		t.Errorf("out = %+v, want {7 42.5}", out)
	}
}

func TestGetWalletBalanceHandler_PropagatesError(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := getWalletBalance
	t.Cleanup(func() { getWalletBalance = orig })
	getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
		return 0, errors.New("simulated failure")
	}

	handler := getWalletBalanceHandler("v1", "client-id", store)
	if _, _, err := handler(context.Background(), nil, GetWalletBalanceInput{CharacterID: 7}); err == nil {
		t.Fatal("handler returned nil error, want the underlying error wrapped")
	}
}

// TestGetWalletBalanceHandler_UnknownCharacter_ReturnsErrCharacterNotFound
// guards against a confirmed consistency gap: unlike add_character/
// list_characters/remove_character, this handler used to skip checking
// config.Store entirely, so an unknown character ID surfaced as a deeply
// wrapped creds.ErrNotFound string instead of the same ErrCharacterNotFound
// sentinel the sibling tools use.
func TestGetWalletBalanceHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := getWalletBalance
	t.Cleanup(func() { getWalletBalance = orig })
	getWalletBalance = func(ctx context.Context, clientID string, characterID int32, userAgent string) (float64, error) {
		t.Fatal("getWalletBalance should not be called for an unknown character")
		return 0, nil
	}

	handler := getWalletBalanceHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, GetWalletBalanceInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
