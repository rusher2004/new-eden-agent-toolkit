package creds_test

import (
	"errors"
	"os"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

func TestMain(m *testing.M) {
	// Never touch the real OS keychain in tests.
	keyring.MockInit()
	os.Exit(m.Run())
}

func TestCharacterRefreshToken_MissingReturnsErrNotFound(t *testing.T) {
	_, err := creds.CharacterRefreshToken(999999)
	if !errors.Is(err, creds.ErrNotFound) {
		t.Errorf("err = %v, want wrapping creds.ErrNotFound", err)
	}
}

func TestSaveThenGetCharacterRefreshToken_RoundTrips(t *testing.T) {
	const characterID = 123456
	const token = "refresh-token-value"

	if err := creds.SaveCharacterRefreshToken(characterID, token); err != nil {
		t.Fatalf("SaveCharacterRefreshToken returned error: %v", err)
	}

	got, err := creds.CharacterRefreshToken(characterID)
	if err != nil {
		t.Fatalf("CharacterRefreshToken returned error: %v", err)
	}
	if got != token {
		t.Errorf("token = %q, want %q", got, token)
	}
}

func TestDeleteCharacterRefreshToken_RemovesIt(t *testing.T) {
	const characterID = 654321
	if err := creds.SaveCharacterRefreshToken(characterID, "temp"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken returned error: %v", err)
	}

	if err := creds.DeleteCharacterRefreshToken(characterID); err != nil {
		t.Fatalf("DeleteCharacterRefreshToken returned error: %v", err)
	}

	if _, err := creds.CharacterRefreshToken(characterID); !errors.Is(err, creds.ErrNotFound) {
		t.Errorf("err = %v, want wrapping creds.ErrNotFound after delete", err)
	}
}

func TestDeleteCharacterRefreshToken_AlreadyAbsentIsSuccess(t *testing.T) {
	if err := creds.DeleteCharacterRefreshToken(999998); err != nil {
		t.Errorf("DeleteCharacterRefreshToken returned error for absent key: %v", err)
	}
}

func TestVerifyAccess_SucceedsWithMockKeyring(t *testing.T) {
	if err := creds.VerifyAccess(); err != nil {
		t.Errorf("VerifyAccess returned error: %v", err)
	}
}

func TestVerifyAccess_ReturnsErrorOnBrokenKeychain(t *testing.T) {
	wantErr := errors.New("keychain locked")
	keyring.MockInitWithError(wantErr)
	t.Cleanup(func() { keyring.MockInit() })

	err := creds.VerifyAccess()
	if err == nil {
		t.Fatal("VerifyAccess returned nil error, want an error from the broken keychain")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wrapping %v", err, wantErr)
	}
}
