package sso

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	goesi "github.com/fnt-eve/goesi-openapi"
	"golang.org/x/oauth2"

	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

// cachedClient is one character's authenticated client and the token
// source behind it, which GrantedScopes reads the current token from.
type cachedClient struct {
	client *http.Client
	source oauth2.TokenSource
}

var (
	clientCacheMu sync.Mutex
	clientCache   = map[int32]cachedClient{}
)

// AuthenticatedClient returns an authenticated, auto-refreshing HTTP client
// for characterID, suitable for making real ESI API calls. The first call
// for a given character builds and caches the client; later calls for the
// same character reuse it — golang.org/x/oauth2's own token source
// transparently refreshes the access token only when it actually expires,
// so callers never need to check expiry themselves.
//
// The token source's context is deliberately detached from ctx's
// cancellation (see refreshContext): golang.org/x/oauth2 stores whatever
// context it's given at construction time and reuses it for every future
// refresh, not just the call that built the client. Since ctx here is the
// triggering MCP request's context, and the MCP framework cancels that
// context as soon as that request's response is written, using it directly
// would make every refresh after the first one fail with "context
// canceled" — permanently breaking the cached client for the rest of the
// process's life. Confirmed by direct reproduction before this fix, and
// confirmed fixed after it.
//
// Holding clientCacheMu across the entire body (including the keychain
// read) also makes this function atomic with respect to RemoveCharacter:
// the two can never interleave, so a concurrent RemoveCharacter can't have
// its keychain delete "beaten" by a stale-but-still-valid read here that
// would otherwise repopulate the cache for a character that's supposed to
// be gone.
func AuthenticatedClient(ctx context.Context, clientID string, characterID int32, userAgent string) (*http.Client, error) {
	clientCacheMu.Lock()
	defer clientCacheMu.Unlock()

	if c, ok := clientCache[characterID]; ok {
		return c.client, nil
	}

	refreshToken, err := creds.CharacterRefreshToken(characterID)
	if err != nil {
		return nil, fmt.Errorf("load refresh token for character %d: %w", characterID, err)
	}

	// No JWKS keyfunc needed: this path never verifies/parses JWT claims,
	// only golang.org/x/oauth2's own refresh-token grant flow.
	cfg, err := goesi.NewConfig(clientID, RedirectURI, nil)
	if err != nil {
		return nil, fmt.Errorf("build ESI OAuth2 config: %w", err)
	}

	token := &oauth2.Token{RefreshToken: refreshToken}
	rCtx := refreshContext(ctx, userAgent)
	source := &persistingTokenSource{
		characterID: characterID,
		base:        cfg.TokenSource(rCtx, token),
		lastSaved:   refreshToken,
	}
	client := oauth2.NewClient(rCtx, source)

	clientCache[characterID] = cachedClient{client: client, source: source}
	return client, nil
}

// refreshContext builds the context used for the token source underlying an
// authenticated client's future refreshes. It carries userAgentClient via
// oauth2.HTTPClient but deliberately drops ctx's own cancellation — see
// AuthenticatedClient's doc comment for why that matters.
func refreshContext(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(context.WithoutCancel(ctx), oauth2.HTTPClient, userAgentClient(userAgent))
}

// persistingTokenSource wraps another oauth2.TokenSource, persisting the
// refresh token to creds whenever it changes. golang.org/x/oauth2's own
// token source updates its in-memory refresh token if FC ever returns a
// different one on refresh, but nothing else observes that change — left
// unpersisted, the keychain copy would go stale and the character would
// silently break on the next process restart with an opaque invalid_grant.
// ESI's own SSO docs describe refresh tokens as reusable indefinitely,
// which suggests FC does not do this in practice, but this closes the gap
// defensively regardless.
type persistingTokenSource struct {
	characterID int32
	base        oauth2.TokenSource

	mu        sync.Mutex
	lastSaved string
}

func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if tok.RefreshToken != "" && tok.RefreshToken != s.lastSaved {
		// Best-effort: a failure here doesn't invalidate the access token
		// we're about to return (still perfectly usable for this process's
		// remaining lifetime), and Token()'s only return channel has no way
		// to report a side-effect failure without also failing the
		// (successful) primary result. Only a future process restart would
		// be affected, which is the exact residual risk this exists to
		// close, not one this can make worse.
		if saveErr := creds.SaveCharacterRefreshToken(s.characterID, tok.RefreshToken); saveErr == nil {
			s.lastSaved = tok.RefreshToken
		}
	}
	return tok, nil
}

// replaceCharacterRefreshToken persists a freshly-issued refresh token for
// characterID and evicts any cached authenticated client for it, holding
// clientCacheMu across both steps. Without the eviction, re-adding a
// character whose old refresh token FC had revoked would save the new
// token to the keychain while AuthenticatedClient kept returning the cached
// client still built around the dead one — every ESI call for that
// character would keep failing with "invalid_grant" until the process
// restarted. Holding the mutex across the keychain write (same as
// RemoveCharacter) means a concurrent AuthenticatedClient call either sees
// the old cache entry before the swap or rebuilds from the new token after
// it, never caches a client built from the old token after eviction.
func replaceCharacterRefreshToken(characterID int32, refreshToken string) error {
	clientCacheMu.Lock()
	defer clientCacheMu.Unlock()

	if err := creds.SaveCharacterRefreshToken(characterID, refreshToken); err != nil {
		return fmt.Errorf("save refresh token for character %d: %w", characterID, err)
	}
	delete(clientCache, characterID)
	return nil
}

// RemoveCharacter deletes characterID's refresh token from the keychain and
// evicts any cached authenticated client for it, atomically with respect to
// AuthenticatedClient (both hold clientCacheMu across their entire keychain
// operation). This closes a TOCTOU where a concurrent AuthenticatedClient
// call could read the still-valid refresh token and populate the cache
// after the credential was already deleted, leaving a live, working client
// cached indefinitely for a character that's supposed to be gone.
func RemoveCharacter(characterID int32) error {
	clientCacheMu.Lock()
	defer clientCacheMu.Unlock()

	if err := creds.DeleteCharacterRefreshToken(characterID); err != nil {
		return fmt.Errorf("delete refresh token for character %d: %w", characterID, err)
	}
	delete(clientCache, characterID)
	return nil
}
