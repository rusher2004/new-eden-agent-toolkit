// Package creds wraps OS keychain access for neat's stored credentials.
package creds

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// Service is the OS keychain service name under which every neat
// credential is stored, regardless of which character it belongs to.
const Service = "neat"

// ErrNotFound is returned (wrapped) when a requested credential does not
// exist in the keychain.
var ErrNotFound = errors.New("credential not found")

// refreshTokenKey builds the per-character keychain key. neat supports zero
// or more characters, so keys are generated per character rather than drawn
// from a fixed set of named constants.
func refreshTokenKey(characterID int32) string {
	return fmt.Sprintf("character_refresh_token:%d", characterID)
}

// SaveCharacterRefreshToken persists the ESI refresh token for a character.
func SaveCharacterRefreshToken(characterID int32, token string) error {
	if err := keyring.Set(Service, refreshTokenKey(characterID), token); err != nil {
		return fmt.Errorf("save refresh token for character %d: %w", characterID, err)
	}
	return nil
}

// CharacterRefreshToken retrieves the ESI refresh token for a character.
func CharacterRefreshToken(characterID int32) (string, error) {
	token, err := keyring.Get(Service, refreshTokenKey(characterID))
	if errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("refresh token for character %d: %w", characterID, ErrNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("get refresh token for character %d: %w", characterID, err)
	}
	return token, nil
}

// DeleteCharacterRefreshToken removes the ESI refresh token for a character.
// Deleting an already-absent token is treated as success.
func DeleteCharacterRefreshToken(characterID int32) error {
	err := keyring.Delete(Service, refreshTokenKey(characterID))
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete refresh token for character %d: %w", characterID, err)
	}
	return nil
}

// VerifyAccess round-trips a throwaway value through the OS keychain so
// callers can fail loudly on a locked/unavailable keychain rather than
// treating it as "unconfigured".
func VerifyAccess() error {
	// "internal:" namespaces this against character-scoped keys
	// (character_refresh_token:<id>), which will never collide with it, and
	// against any future non-character key scheme that might otherwise.
	const canaryKey = "internal:setup_canary"
	const canaryValue = "ok"

	if err := keyring.Set(Service, canaryKey, canaryValue); err != nil {
		return fmt.Errorf("write canary credential: %w", err)
	}
	got, err := keyring.Get(Service, canaryKey)
	if err != nil {
		return fmt.Errorf("read canary credential: %w", err)
	}
	if got != canaryValue {
		return fmt.Errorf("read canary credential: got %q, want %q (keychain returned stale or corrupt data)", got, canaryValue)
	}
	if err := keyring.Delete(Service, canaryKey); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete canary credential: %w", err)
	}
	return nil
}
