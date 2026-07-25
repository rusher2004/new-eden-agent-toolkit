package sso

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"sync"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
	"golang.org/x/oauth2"

	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

// addCharacterMu closes the same-process version of a narrow TOCTOU window
// in the socket-level collision check (see Task 6's review): without it,
// two goroutines calling addCharacter at nearly the same instant could both
// pass the liveness probe before either finishes binding. TryLock makes
// that check atomic and immediate, no timing window at all, for callers in
// this process. It does not protect against two separate neat OS processes
// racing — accepted as a narrow residual limitation.
var addCharacterMu sync.Mutex

// RedirectURI is the ESI OAuth2 redirect URI registered for this app.
const RedirectURI = "eveauth-neat://callback"

// callbackTimeout bounds how long AddCharacter waits for the browser
// callback. Overridable in tests.
var callbackTimeout = 5 * time.Minute

// openBrowser launches the user's default browser at rawURL. Overridable
// in tests so no real browser is launched during automated runs.
var openBrowser = func(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}
	return nil
}

// oauthConfig is the subset of *goesi.Config's behavior AddCharacter needs.
// Defined here rather than imported from goesi so tests can substitute a
// fake with no network or JWT-verification dependency.
type oauthConfig interface {
	AuthURL(state string, scopes []string) string
	Exchange(ctx context.Context, code, state, expectedState string) (*oauth2.Token, *goesi.ESIJWTClaims, error)
}

// AddCharacter runs the full OAuth2/PKCE flow for a new character: opens
// the browser, waits for the callback, exchanges the code, and persists
// the refresh token.
func AddCharacter(ctx context.Context, clientID, userAgent string, scopes []string) (Character, error) {
	cfg, err := newOAuthConfig(ctx, clientID, userAgent)
	if err != nil {
		return Character{}, fmt.Errorf("build OAuth config: %w", err)
	}
	return addCharacter(ctx, cfg, userAgent, scopes)
}

func newOAuthConfig(ctx context.Context, clientID, userAgent string) (oauthConfig, error) {
	keyFunc, err := newKeyfunc(ctx, userAgent)
	if err != nil {
		return nil, err
	}
	cfg, err := goesi.NewConfig(clientID, RedirectURI, &keyFunc)
	if err != nil {
		return nil, fmt.Errorf("build ESI OAuth2 config: %w", err)
	}
	return cfg, nil
}

func addCharacter(ctx context.Context, cfg oauthConfig, userAgent string, scopes []string) (Character, error) {
	if !addCharacterMu.TryLock() {
		return Character{}, ErrAddCharacterInProgress
	}
	defer addCharacterMu.Unlock()

	ctx = context.WithValue(ctx, oauth2.HTTPClient, userAgentClient(userAgent))

	lockPath, err := callbackLockPath()
	if err != nil {
		return Character{}, err
	}
	deliveryPath, err := callbackSocketPath()
	if err != nil {
		return Character{}, err
	}

	listener, err := listenForCallback(lockPath, deliveryPath)
	if err != nil {
		return Character{}, err
	}
	defer listener.Close()

	state, err := generateState()
	if err != nil {
		return Character{}, fmt.Errorf("generate state: %w", err)
	}

	if err := openBrowser(cfg.AuthURL(state, scopes)); err != nil {
		return Character{}, fmt.Errorf("open browser: %w", err)
	}

	rawCallback, err := receiveCallback(ctx, listener.delivery, callbackTimeout)
	if err != nil {
		return Character{}, err
	}

	code, gotState, err := parseCallback(rawCallback)
	if err != nil {
		return Character{}, err
	}

	token, claims, err := cfg.Exchange(ctx, code, gotState, state)
	if err != nil {
		return Character{}, fmt.Errorf("exchange authorization code: %w", err)
	}

	characterID, err := characterIDFromClaims(claims)
	if err != nil {
		return Character{}, err
	}

	if err := creds.SaveCharacterRefreshToken(characterID, token.RefreshToken); err != nil {
		return Character{}, fmt.Errorf("save refresh token: %w", err)
	}

	return Character{ID: characterID, Name: claims.Name}, nil
}

func parseCallback(rawURL string) (code, state string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("parse callback URL: %w", err)
	}
	q := u.Query()
	code = q.Get("code")
	state = q.Get("state")
	if code == "" {
		return "", "", errors.New("callback URL missing code parameter")
	}
	if state == "" {
		return "", "", errors.New("callback URL missing state parameter")
	}
	return code, state, nil
}

// ExportedAddCharacter exposes addCharacter for external tests.
func ExportedAddCharacter(ctx context.Context, cfg interface {
	AuthURL(state string, scopes []string) string
	Exchange(ctx context.Context, code, state, expectedState string) (*oauth2.Token, *goesi.ESIJWTClaims, error)
}, userAgent string, scopes []string) (Character, error) {
	return addCharacter(ctx, cfg, userAgent, scopes)
}

// SetOpenBrowserForTest overrides the browser launcher for tests.
func SetOpenBrowserForTest(fn func(rawURL string) error) {
	openBrowser = fn
}

// ResetOpenBrowserForTest restores the real browser launcher. Since the
// real implementation is only ever assigned once (as openBrowser's
// initializer), tests must not depend on running before any other test has
// overridden it — resetting re-assigns the same real logic inline.
func ResetOpenBrowserForTest() {
	openBrowser = func(rawURL string) error {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", rawURL)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
		default:
			cmd = exec.Command("xdg-open", rawURL)
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("launch browser: %w", err)
		}
		return nil
	}
}

// SetCallbackTimeoutForTest overrides the callback wait timeout for tests.
func SetCallbackTimeoutForTest(d time.Duration) {
	callbackTimeout = d
}

// ResetCallbackTimeoutForTest restores the real 5-minute callback timeout.
func ResetCallbackTimeoutForTest() {
	callbackTimeout = 5 * time.Minute
}
