package sso

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
	"golang.org/x/oauth2"
)

// addCharacterMu lets only one login session run per process. A second
// add_character call in the same process fails immediately with
// ErrAddCharacterInProgress instead of failing to bind the callback port.
// A second neat process is kept out by the port bind itself, which the OS
// makes exclusive.
var addCharacterMu sync.Mutex

// ErrAddCharacterInProgress is returned when add_character is called while
// another login session in this process is still open.
var ErrAddCharacterInProgress = errors.New("a character login session is already open in the browser; finish it there (or close its tab) first")

// callbackPort is the fixed loopback port of the OAuth2 redirect. FC's SSO
// requires the redirect URI to match the one registered on the ESI
// application exactly, port included, so it can't be chosen at runtime.
const callbackPort = "48211"

// RedirectURI is the ESI OAuth2 redirect URI registered for this app. It
// uses the 127.0.0.1 literal rather than "localhost", per RFC 8252 §8.3, so
// the browser can't resolve it to an IPv6 or non-loopback address that
// nothing is listening on.
const RedirectURI = "http://127.0.0.1:" + callbackPort + "/callback"

// callbackListenAddr is where a login session's HTTP server listens.
// Overridable in tests, which listen on an ephemeral port instead.
var callbackListenAddr = "127.0.0.1:" + callbackPort

// onListening is told the base URL of each session's server once it is
// listening. A no-op outside tests, which use it to find the ephemeral port.
var onListening = func(string) {}

// callbackTimeout bounds how long a session waits for the browser to come
// back from FC's login page after a login starts. Overridable in tests.
var callbackTimeout = 5 * time.Minute

// pageIdleTimeout ends a session once its page has stopped checking in for
// this long (the tab was closed, but its close beacon never arrived). The
// page pings every 10 seconds while open. Overridable in tests.
var pageIdleTimeout = 30 * time.Second

// closeGrace is how long a session lingers after its page reports being
// closed, so that a reload, which also fires the close beacon, picks the
// session back up instead of finding the server gone. The reloaded page
// pings as soon as it loads, since the old page's beacon arrives only after
// the new page was already served.
var closeGrace = 5 * time.Second

func defaultOpenBrowser(rawURL string) error {
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

// openBrowser launches the user's default browser at rawURL. Overridable
// in tests so no real browser is launched during automated runs.
var openBrowser = defaultOpenBrowser

// oauthConfig is the subset of *goesi.Config's behavior AddCharacters needs.
// Defined here rather than imported from goesi so tests can substitute a
// fake with no network or JWT-verification dependency.
type oauthConfig interface {
	AuthURL(state string, scopes []string) string
	Exchange(ctx context.Context, code, state, expectedState string) (*oauth2.Token, *goesi.ESIJWTClaims, error)
}

// AddOptions configures a login session.
type AddOptions struct {
	// Scopes are the ESI scopes requested for every character.
	Scopes []string
	// OnAdded is called after each character's refresh token is saved,
	// before the browser is shown the result, so the caller can record the
	// character (e.g. in config) while the session is still open. An error
	// is shown on the page and the character is not counted as added.
	OnAdded func(Character) error
	// SavedCharacters lists the characters neat already knows, for the
	// session page. Optional: without it, the page lists only the
	// characters added in this session.
	SavedCharacters func() ([]Character, error)
}

// AddCharacters runs a browser login session: it serves the OAuth2 redirect
// on a loopback port, opens the browser at FC's login page, and keeps
// serving a page that lists the user's characters and lets them log in
// more, until they click Done, close the tab, or a login times out. Each
// character's refresh token is saved as soon as its login completes. It
// returns every character added; it fails only if none was.
func AddCharacters(ctx context.Context, clientID, userAgent string, opts AddOptions) ([]Character, error) {
	cfg, err := newOAuthConfig(ctx, clientID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build OAuth config: %w", err)
	}
	return addCharacters(ctx, cfg, userAgent, opts)
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

func addCharacters(ctx context.Context, cfg oauthConfig, userAgent string, opts AddOptions) ([]Character, error) {
	if !addCharacterMu.TryLock() {
		return nil, ErrAddCharacterInProgress
	}
	defer addCharacterMu.Unlock()

	listener, err := net.Listen("tcp", callbackListenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen for the login callback on %s (another neat process may have a login open, or another program is using the port): %w", callbackListenAddr, err)
	}

	sessionCtx := context.WithValue(ctx, oauth2.HTTPClient, userAgentClient(userAgent))
	s, err := newLoginSession(sessionCtx, cfg, opts, listener.Addr().String())
	if err != nil {
		if closeErr := listener.Close(); closeErr != nil {
			slog.Default().WarnContext(ctx, "sso: failed to close the login callback listener", "error", closeErr)
		}
		return nil, err
	}

	server := &http.Server{Handler: s.handler(), ReadHeaderTimeout: 10 * time.Second}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			if closeErr := server.Close(); closeErr != nil {
				slog.Default().WarnContext(ctx, "sso: failed to close the login session server", "shutdownError", err, "error", closeErr)
			}
		}
		<-serveErr
	}()
	onListening("http://" + listener.Addr().String())

	authURL, err := s.startLogin()
	if err != nil {
		return nil, err
	}
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}

	waitErr := s.wait(ctx)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	added, lastErr := s.result()
	switch {
	case len(added) > 0:
		return added, nil
	case lastErr != nil:
		return nil, lastErr
	case waitErr != nil:
		return nil, waitErr
	default:
		return nil, errors.New("the browser login session ended without adding a character")
	}
}

// ExportedAddCharacters exposes addCharacters for external tests.
func ExportedAddCharacters(ctx context.Context, cfg interface {
	AuthURL(state string, scopes []string) string
	Exchange(ctx context.Context, code, state, expectedState string) (*oauth2.Token, *goesi.ESIJWTClaims, error)
}, userAgent string, opts AddOptions) ([]Character, error) {
	return addCharacters(ctx, cfg, userAgent, opts)
}

// SetOpenBrowserForTest overrides the browser launcher for tests.
func SetOpenBrowserForTest(fn func(rawURL string) error) {
	openBrowser = fn
}

// ResetOpenBrowserForTest restores the real browser launcher.
func ResetOpenBrowserForTest() {
	openBrowser = defaultOpenBrowser
}

// SetListenForTest makes sessions listen on addr (e.g. "127.0.0.1:0") and
// report their base URL to onListen, for tests.
func SetListenForTest(addr string, onListen func(baseURL string)) {
	callbackListenAddr = addr
	onListening = onListen
}

// ResetListenForTest restores the real callback address.
func ResetListenForTest() {
	callbackListenAddr = "127.0.0.1:" + callbackPort
	onListening = func(string) {}
}

// SetTimeoutsForTest overrides the session's timeouts for tests.
func SetTimeoutsForTest(callback, pageIdle, grace time.Duration) {
	callbackTimeout, pageIdleTimeout, closeGrace = callback, pageIdle, grace
}

// ResetTimeoutsForTest restores the real timeouts.
func ResetTimeoutsForTest() {
	callbackTimeout, pageIdleTimeout, closeGrace = 5*time.Minute, 30*time.Second, 5*time.Second
}
