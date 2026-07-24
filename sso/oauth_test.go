package sso_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/creds"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

func TestMain(m *testing.M) {
	keyring.MockInit()
	m.Run()
}

type fakeOAuthConfig struct {
	token  *oauth2.Token
	claims *goesi.ESIJWTClaims
}

func (f *fakeOAuthConfig) AuthURL(state string, scopes []string) string {
	return "https://login.eveonline.com/v2/oauth/authorize?state=" + state
}

func (f *fakeOAuthConfig) Exchange(ctx context.Context, code, state, expectedState string) (*oauth2.Token, *goesi.ESIJWTClaims, error) {
	if state != expectedState {
		return nil, nil, errors.New("state mismatch")
	}
	if code == "" {
		return nil, nil, errors.New("missing code")
	}
	return f.token, f.claims, nil
}

func TestAddCharacter_FullFlow_Succeeds(t *testing.T) {
	// shortTempHome (not t.TempDir() directly), plus forcing TMPDIR, avoids
	// overflowing the ~104-byte Darwin sockaddr_un limit once ".neat/run/
	// callback.sock" is appended — see socket_test.go's shortTempHome for
	// the full rationale. Without it, listenForCallback fails before
	// openBrowser is ever called and this test hangs forever on
	// <-browserOpened.
	tmpHome := shortTempHome(t)
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	var capturedAuthURL string
	browserOpened := make(chan struct{})
	sso.SetOpenBrowserForTest(func(rawURL string) error {
		capturedAuthURL = rawURL
		close(browserOpened)
		return nil
	})
	t.Cleanup(sso.ResetOpenBrowserForTest)
	sso.SetCallbackTimeoutForTest(5 * time.Second)
	t.Cleanup(sso.ResetCallbackTimeoutForTest)

	fake := &fakeOAuthConfig{
		token: &oauth2.Token{RefreshToken: "refresh-123"},
		claims: &goesi.ESIJWTClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "CHARACTER:EVE:998877"},
			Name:             "Test Pilot",
		},
	}

	type addResult struct {
		char sso.Character
		err  error
	}
	resultCh := make(chan addResult, 1)
	go func() {
		char, err := sso.ExportedAddCharacter(context.Background(), fake, []string{"publicData"})
		resultCh <- addResult{char, err}
	}()

	<-browserOpened
	parsed, err := url.Parse(capturedAuthURL)
	if err != nil {
		t.Fatalf("parse captured auth URL: %v", err)
	}
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatal("captured auth URL had no state parameter")
	}

	if err := sso.DeliverCallback("eveauth-neat://callback?code=auth-code-123&state=" + state); err != nil {
		t.Fatalf("DeliverCallback returned error: %v", err)
	}

	res := <-resultCh
	if res.err != nil {
		t.Fatalf("addCharacter returned error: %v", res.err)
	}
	if res.char != (sso.Character{ID: 998877, Name: "Test Pilot"}) {
		t.Errorf("Character = %+v, want {998877 Test Pilot}", res.char)
	}

	got, err := creds.CharacterRefreshToken(998877)
	if err != nil {
		t.Fatalf("CharacterRefreshToken returned error: %v", err)
	}
	if got != "refresh-123" {
		t.Errorf("persisted refresh token = %q, want %q", got, "refresh-123")
	}
}

func TestAddCharacter_CallbackStateMismatch_Fails(t *testing.T) {
	// See TestAddCharacter_FullFlow_Succeeds for why shortTempHome (not
	// t.TempDir() directly) plus forcing TMPDIR is required here too: this
	// test also drives addCharacter through listenForCallback, which binds
	// a Unix socket under HOME and would overflow Darwin's ~104-byte
	// sockaddr_un limit with a test-name-derived path.
	tmpHome := shortTempHome(t)
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	browserOpened := make(chan struct{})
	sso.SetOpenBrowserForTest(func(rawURL string) error {
		close(browserOpened)
		return nil
	})
	t.Cleanup(sso.ResetOpenBrowserForTest)
	sso.SetCallbackTimeoutForTest(5 * time.Second)
	t.Cleanup(sso.ResetCallbackTimeoutForTest)

	fake := &fakeOAuthConfig{
		token: &oauth2.Token{RefreshToken: "refresh-123"},
		claims: &goesi.ESIJWTClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "CHARACTER:EVE:998877"},
			Name:             "Test Pilot",
		},
	}

	resultCh := make(chan struct {
		char sso.Character
		err  error
	}, 1)
	go func() {
		char, err := sso.ExportedAddCharacter(context.Background(), fake, []string{"publicData"})
		resultCh <- struct {
			char sso.Character
			err  error
		}{char, err}
	}()

	<-browserOpened

	// Deliver a callback with a state that does NOT match the state
	// AddCharacter actually generated (it never appears anywhere the test
	// can see, unlike the full-flow test which extracts it from the
	// captured auth URL) — this exercises the wiring that maps the
	// delivered callback's state into Exchange's "actual" argument against
	// the generated state as the "expected" argument.
	if err := sso.DeliverCallback("eveauth-neat://callback?code=auth-code-123&state=wrong-state"); err != nil {
		t.Fatalf("DeliverCallback returned error: %v", err)
	}

	res := <-resultCh
	if res.err == nil {
		t.Fatal("AddCharacter returned nil error, want an error from the state mismatch")
	}
}

func TestAddCharacter_SecondCallWhileFirstInFlight_Fails(t *testing.T) {
	// See TestAddCharacter_FullFlow_Succeeds for why shortTempHome is used
	// instead of t.TempDir() directly.
	tmpHome := shortTempHome(t)
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	sso.SetOpenBrowserForTest(func(rawURL string) error { return nil })
	sso.SetCallbackTimeoutForTest(5 * time.Second)

	fake := &fakeOAuthConfig{}

	// The first call is left running deliberately (to hold the lock while
	// the second call probes it) and unblocks on its own once
	// callbackTimeout elapses. t.Cleanup runs LIFO, so registering the
	// "wait for it to finish" cleanup last makes it run *before*
	// ResetOpenBrowserForTest/ResetCallbackTimeoutForTest reset the very
	// package-level vars that leaked goroutine still reads inside
	// addCharacter/receiveCallback — without this ordering, the reset
	// races with that read under -race (and is real undefined behavior,
	// not just a detector false positive).
	firstDone := make(chan struct{})
	t.Cleanup(sso.ResetCallbackTimeoutForTest)
	t.Cleanup(sso.ResetOpenBrowserForTest)
	t.Cleanup(func() { <-firstDone })

	go func() {
		defer close(firstDone)
		sso.ExportedAddCharacter(context.Background(), fake, nil) //nolint:errcheck // first call is left running deliberately; awaited in cleanup above
	}()

	// Give the first call's goroutine time to reach listenForCallback
	// before starting the second.
	time.Sleep(50 * time.Millisecond)

	_, err := sso.ExportedAddCharacter(context.Background(), fake, nil)
	if !errors.Is(err, sso.ErrAddCharacterInProgress) {
		t.Errorf("err = %v, want wrapping ErrAddCharacterInProgress", err)
	}
}
