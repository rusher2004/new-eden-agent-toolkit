package sso_test

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/rusher2004/new-eden-agent-toolkit/sso"

	goesi "github.com/fnt-eve/goesi-openapi"
)

func TestCharacterIDFromClaims_ValidSubject(t *testing.T) {
	claims := &goesi.ESIJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "CHARACTER:EVE:998877"},
	}

	id, err := sso.ExportedCharacterIDFromClaims(claims)
	if err != nil {
		t.Fatalf("characterIDFromClaims returned error: %v", err)
	}
	if id != 998877 {
		t.Errorf("id = %d, want 998877", id)
	}
}

func TestCharacterIDFromClaims_TooLargeForInt32(t *testing.T) {
	claims := &goesi.ESIJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "CHARACTER:EVE:99999999999"},
	}

	_, err := sso.ExportedCharacterIDFromClaims(claims)
	if err == nil {
		t.Fatal("characterIDFromClaims returned nil error, want an out-of-range error")
	}
}

func TestCharacterIDFromClaims_NegativeIDRejected(t *testing.T) {
	claims := &goesi.ESIJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "CHARACTER:EVE:-1"},
	}

	_, err := sso.ExportedCharacterIDFromClaims(claims)
	if err == nil {
		t.Fatal("characterIDFromClaims returned nil error, want a negative-ID rejection error")
	}
}

func TestCharacterIDFromClaims_MalformedSubjectPropagatesClaimsError(t *testing.T) {
	claims := &goesi.ESIJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "CHARACTER:EVE:not-a-number"},
	}

	_, err := sso.ExportedCharacterIDFromClaims(claims)
	if err == nil {
		t.Fatal("characterIDFromClaims returned nil error, want an error from claims.CharacterID() failing to parse")
	}
}

func TestGenerateState_ProducesDistinctNonEmptyValues(t *testing.T) {
	a, err := sso.ExportedGenerateState()
	if err != nil {
		t.Fatalf("generateState returned error: %v", err)
	}
	b, err := sso.ExportedGenerateState()
	if err != nil {
		t.Fatalf("generateState returned error: %v", err)
	}
	if a == "" || b == "" {
		t.Fatal("generateState returned an empty string")
	}
	if a == b {
		t.Error("two calls to generateState produced the same value")
	}
}
