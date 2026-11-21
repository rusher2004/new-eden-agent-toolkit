// Package sso owns ESI's OAuth2/PKCE login flow: opening the browser,
// serving the loopback callback and session page, exchanging the code, and persisting the
// resulting refresh token via creds/.
package sso

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"

	goesi "github.com/fnt-eve/goesi-openapi"
)

// Character is the result of a successful add-character flow.
type Character struct {
	ID   int32
	Name string
	// Account is the user's label for the EVE account the character is on,
	// for characters neat already knows; empty means unknown.
	Account string
	// Scopes are the ESI scopes FC granted in this login, from the
	// token's scp claim.
	Scopes []string
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

// ExportedCharacterIDFromClaims exposes characterIDFromClaims for external tests.
func ExportedCharacterIDFromClaims(claims *goesi.ESIJWTClaims) (int32, error) {
	return characterIDFromClaims(claims)
}

// ExportedGenerateState exposes generateState for external tests.
func ExportedGenerateState() (string, error) {
	return generateState()
}
