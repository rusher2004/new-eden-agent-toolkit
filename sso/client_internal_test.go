package sso

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/oauth2"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

// TestRefreshContext_DetachesCancellation guards against a confirmed,
// reproduced bug: golang.org/x/oauth2 stores whatever context it's given
// at token-source construction time and reuses it for every future
// refresh. If refreshContext didn't detach cancellation, cancelling the
// original (triggering-request) ctx would make every later refresh on the
// cached client fail with "context canceled" -- permanently, for the rest
// of the process's life, since the cache never rebuilds a client once
// created.
func TestRefreshContext_DetachesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	refreshCtx := refreshContext(ctx, "test-agent")

	cancel()

	select {
	case <-refreshCtx.Done():
		t.Error("refreshCtx.Done() fired after cancelling the original ctx, want cancellation detached")
	default:
	}
}

type fakeRotatingSource struct {
	tokens []*oauth2.Token
	i      int
}

func (f *fakeRotatingSource) Token() (*oauth2.Token, error) {
	if f.i >= len(f.tokens) {
		return nil, errors.New("no more fake tokens")
	}
	tok := f.tokens[f.i]
	f.i++
	return tok, nil
}

// TestPersistingTokenSource_PersistsRotatedRefreshToken guards against a
// confirmed gap: golang.org/x/oauth2 updates its own in-memory refresh
// token if the authorization server ever returns a different one, but
// nothing else observes or persists that change -- the keychain copy would
// go stale, silently breaking the character on the next process restart.
func TestPersistingTokenSource_PersistsRotatedRefreshToken(t *testing.T) {
	keyring.MockInit()
	const characterID int32 = 77
	if err := creds.SaveCharacterRefreshToken(characterID, "original-refresh"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}

	source := &persistingTokenSource{
		characterID: characterID,
		lastSaved:   "original-refresh",
		base: &fakeRotatingSource{tokens: []*oauth2.Token{
			{AccessToken: "access-1", RefreshToken: "original-refresh"}, // unchanged -- must NOT trigger a redundant save
			{AccessToken: "access-2", RefreshToken: "rotated-refresh"},  // CCP rotated it -- must persist
		}},
	}

	if _, err := source.Token(); err != nil {
		t.Fatalf("first Token(): %v", err)
	}
	got, err := creds.CharacterRefreshToken(characterID)
	if err != nil {
		t.Fatalf("CharacterRefreshToken after first Token(): %v", err)
	}
	if got != "original-refresh" {
		t.Errorf("after unchanged token, stored refresh token = %q, want unchanged %q", got, "original-refresh")
	}

	if _, err := source.Token(); err != nil {
		t.Fatalf("second Token(): %v", err)
	}
	got, err = creds.CharacterRefreshToken(characterID)
	if err != nil {
		t.Fatalf("CharacterRefreshToken after second Token(): %v", err)
	}
	if got != "rotated-refresh" {
		t.Errorf("after rotated token, stored refresh token = %q, want %q", got, "rotated-refresh")
	}
}
