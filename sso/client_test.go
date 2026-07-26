package sso_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/creds"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

func TestAuthenticatedClient_CachesPerCharacter(t *testing.T) {
	keyring.MockInit()
	if err := creds.SaveCharacterRefreshToken(1, "refresh-1"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}

	c1, err := sso.AuthenticatedClient(context.Background(), "client-id", 1, "test-agent")
	if err != nil {
		t.Fatalf("first AuthenticatedClient: %v", err)
	}
	c2, err := sso.AuthenticatedClient(context.Background(), "client-id", 1, "test-agent")
	if err != nil {
		t.Fatalf("second AuthenticatedClient: %v", err)
	}
	if c1 != c2 {
		t.Errorf("c1 = %p, c2 = %p, want same pointer (cache hit)", c1, c2)
	}
}

func TestAuthenticatedClient_DifferentCharactersGetDifferentClients(t *testing.T) {
	keyring.MockInit()
	if err := creds.SaveCharacterRefreshToken(2, "refresh-2"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}
	if err := creds.SaveCharacterRefreshToken(3, "refresh-3"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}

	c1, err := sso.AuthenticatedClient(context.Background(), "client-id", 2, "test-agent")
	if err != nil {
		t.Fatalf("AuthenticatedClient(2): %v", err)
	}
	c2, err := sso.AuthenticatedClient(context.Background(), "client-id", 3, "test-agent")
	if err != nil {
		t.Fatalf("AuthenticatedClient(3): %v", err)
	}
	if c1 == c2 {
		t.Error("c1 == c2, want distinct pointers for different characters")
	}
}

func TestAuthenticatedClient_UnknownCharacter_ReturnsError(t *testing.T) {
	keyring.MockInit()
	_, err := sso.AuthenticatedClient(context.Background(), "client-id", 999, "test-agent")
	if !errors.Is(err, creds.ErrNotFound) {
		t.Errorf("err = %v, want wrapping creds.ErrNotFound", err)
	}
}

func TestRemoveCharacter_RemovesCachedEntry(t *testing.T) {
	keyring.MockInit()
	if err := creds.SaveCharacterRefreshToken(4, "refresh-4"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}

	if _, err := sso.AuthenticatedClient(context.Background(), "client-id", 4, "test-agent"); err != nil {
		t.Fatalf("first AuthenticatedClient: %v", err)
	}
	if err := sso.RemoveCharacter(4); err != nil {
		t.Fatalf("RemoveCharacter: %v", err)
	}

	// A fresh AuthenticatedClient call must now fail -- the credential is
	// gone, and no stale cached entry should still satisfy it.
	if _, err := sso.AuthenticatedClient(context.Background(), "client-id", 4, "test-agent"); !errors.Is(err, creds.ErrNotFound) {
		t.Errorf("err = %v, want wrapping creds.ErrNotFound after RemoveCharacter", err)
	}
}

// TestAuthenticatedClient_RemoveCharacter_NoResurrection guards against a
// confirmed TOCTOU: previously, a concurrent AuthenticatedClient call could
// read the still-valid refresh token and populate the cache after
// RemoveCharacter had already deleted the credential, leaving a live,
// working client cached indefinitely for a character that's supposed to be
// gone. clientCacheMu now serializes AuthenticatedClient and
// RemoveCharacter completely, so regardless of which one actually runs
// first, the outcome afterward is deterministic: the cache can never end
// up holding a client for a character whose credential is already deleted.
func TestAuthenticatedClient_RemoveCharacter_NoResurrection(t *testing.T) {
	keyring.MockInit()
	const characterID int32 = 55
	if err := creds.SaveCharacterRefreshToken(characterID, "refresh-55"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		sso.AuthenticatedClient(context.Background(), "client-id", characterID, "test-agent") //nolint:errcheck // outcome checked below regardless of which goroutine wins
	}()
	go func() {
		defer wg.Done()
		sso.RemoveCharacter(characterID) //nolint:errcheck // outcome checked below regardless of which goroutine wins
	}()
	wg.Wait()

	if _, err := sso.AuthenticatedClient(context.Background(), "client-id", characterID, "test-agent"); !errors.Is(err, creds.ErrNotFound) {
		t.Errorf("err = %v, want wrapping creds.ErrNotFound -- a cached client survived concurrent removal (TOCTOU not closed)", err)
	}
}
