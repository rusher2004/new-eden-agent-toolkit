// Package sso owns ESI's OAuth2/PKCE login flow: opening the browser,
// receiving the callback, exchanging the code, and persisting the
// resulting refresh token via creds/.
package sso

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"os"
	"path/filepath"

	goesi "github.com/fnt-eve/goesi-openapi"
)

// Character is the result of a successful add-character flow.
type Character struct {
	ID   int32
	Name string
}

// generateState returns a fresh random OAuth2 state value.
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// characterIDFromClaims extracts the character ID from verified JWT claims,
// converting from the JWT's int64 to the int32 creds/config already commit
// to (EVE character IDs fit comfortably within int32's range in practice;
// this still checks explicitly rather than truncating silently).
func characterIDFromClaims(claims *goesi.ESIJWTClaims) (int32, error) {
	id64, err := claims.CharacterID()
	if err != nil {
		return 0, fmt.Errorf("extract character ID from claims: %w", err)
	}
	if id64 < 0 || id64 > math.MaxInt32 {
		return 0, fmt.Errorf("character ID %d is out of int32 range", id64)
	}
	return int32(id64), nil
}

// callbackSocketPath resolves the Unix domain socket path used to hand off
// the OAuth2 redirect URL from the callback-handler subcommand to the
// blocked add_character flow: ~/.neat/run/callback.sock.
func callbackSocketPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".neat", "run", "callback.sock"), nil
}

// callbackLockPath resolves the Unix domain socket path used purely as a
// liveness signal for detecting a concurrent add_character flow:
// ~/.neat/run/callback.lock. It is bound but never Accept()-ed — see
// listenForCallback in socket.go for why it is kept separate from the
// delivery socket.
func callbackLockPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".neat", "run", "callback.lock"), nil
}

// ExportedCharacterIDFromClaims exposes characterIDFromClaims for external tests.
func ExportedCharacterIDFromClaims(claims *goesi.ESIJWTClaims) (int32, error) {
	return characterIDFromClaims(claims)
}

// ExportedGenerateState exposes generateState for external tests.
func ExportedGenerateState() (string, error) {
	return generateState()
}
