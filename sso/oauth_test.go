package sso_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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

const fakeAuthURL = "https://login.eveonline.com/v2/oauth/authorize"

// fakeOAuthConfig logs in whichever character its code names: code
// "char-<id>" returns the character in chars with that ID.
type fakeOAuthConfig struct {
	chars map[string]sso.Character
	// upstreamURL, if set, is fetched during Exchange with the context's
	// oauth2.HTTPClient, so a test can see what that client sends.
	upstreamURL string
}

func (f *fakeOAuthConfig) AuthURL(state string, scopes []string) string {
	return fakeAuthURL + "?state=" + url.QueryEscape(state)
}

func (f *fakeOAuthConfig) Exchange(ctx context.Context, code, state, expectedState string) (*oauth2.Token, *goesi.ESIJWTClaims, error) {
	if state != expectedState {
		return nil, nil, errors.New("state mismatch")
	}
	if f.upstreamURL != "" {
		client, _ := ctx.Value(oauth2.HTTPClient).(*http.Client)
		if client == nil {
			client = http.DefaultClient
		}
		resp, err := client.Get(f.upstreamURL)
		if err != nil {
			return nil, nil, err
		}
		resp.Body.Close()
	}
	c, ok := f.chars[code]
	if !ok {
		return nil, nil, errors.New("unknown code")
	}
	claims := &goesi.ESIJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "CHARACTER:EVE:" + strconv.Itoa(int(c.ID))},
		Name:             c.Name,
	}
	return &oauth2.Token{RefreshToken: "refresh-" + code}, claims, nil
}

type addResult struct {
	chars []sso.Character
	err   error
}

// session drives one ExportedAddCharacters call as the browser would.
type session struct {
	t        *testing.T
	base     string
	authURLs chan string
	result   chan addResult
	added    chan sso.Character
	client   *http.Client
}

// startSession starts an add-characters session against fake on an
// ephemeral port and waits for its first browser launch.
func startSession(t *testing.T, ctx context.Context, fake *fakeOAuthConfig, userAgent string) *session {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	s := &session{
		t:        t,
		authURLs: make(chan string, 8),
		result:   make(chan addResult, 1),
		added:    make(chan sso.Character, 8),
		// Don't follow redirects: tests assert on where each one points.
		client: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
	baseCh := make(chan string, 1)
	sso.SetListenForTest("127.0.0.1:0", func(base string) { baseCh <- base })
	t.Cleanup(sso.ResetListenForTest)
	sso.SetOpenBrowserForTest(func(rawURL string) error {
		s.authURLs <- rawURL
		return nil
	})
	t.Cleanup(sso.ResetOpenBrowserForTest)

	// Cancel on cleanup so a test that fails mid-session can't leave it
	// holding the per-process lock for the tests after it.
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	t.Cleanup(func() { cancel(); <-done })

	go func() {
		defer close(done)
		chars, err := sso.ExportedAddCharacters(ctx, fake, userAgent, sso.AddOptions{
			Scopes: []string{"publicData"},
			OnAdded: func(c sso.Character) error {
				s.added <- c
				return nil
			},
		})
		s.result <- addResult{chars, err}
	}()

	select {
	case s.base = <-baseCh:
	case res := <-s.result:
		t.Fatalf("session ended before listening: %v", res.err)
	}
	return s
}

// state waits for the next browser launch and returns its OAuth2 state.
func (s *session) state() string {
	s.t.Helper()
	select {
	case raw := <-s.authURLs:
		return stateOf(s.t, raw)
	case <-time.After(5 * time.Second):
		s.t.Fatal("browser was never opened")
		return ""
	}
}

// addAnother clicks the page's "Add another character" button and returns
// the OAuth2 state of the CCP login it redirects to.
func (s *session) addAnother(token string) string {
	s.t.Helper()
	resp, _ := s.post("/login", token)
	loc := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, fakeAuthURL) {
		s.t.Fatalf("POST /login = %d to %q, want 303 to CCP", resp.StatusCode, loc)
	}
	return stateOf(s.t, loc)
}

func stateOf(t *testing.T, authURL string) string {
	t.Helper()
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	return u.Query().Get("state")
}

func (s *session) get(path string) (*http.Response, string) {
	s.t.Helper()
	resp, err := s.client.Get(s.base + path)
	if err != nil {
		s.t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.t.Fatalf("read GET %s body: %v", path, err)
	}
	return resp, string(body)
}

func (s *session) post(path, token string) (*http.Response, string) {
	s.t.Helper()
	resp, err := s.client.PostForm(s.base+path, url.Values{"token": {token}})
	if err != nil {
		s.t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.t.Fatalf("read POST %s body: %v", path, err)
	}
	return resp, string(body)
}

// callback plays CCP redirecting the browser back with code and state.
func (s *session) callback(code, state string) *http.Response {
	s.t.Helper()
	resp, _ := s.get("/callback?" + url.Values{"code": {code}, "state": {state}}.Encode())
	return resp
}

var tokenRE = regexp.MustCompile(`data-token="([^"]+)"`)

// page loads the session page and returns its body and page token.
func (s *session) page() (body, token string) {
	s.t.Helper()
	resp, body := s.get("/")
	if resp.StatusCode != http.StatusOK {
		s.t.Fatalf("GET / status = %d, want 200", resp.StatusCode)
	}
	m := tokenRE.FindStringSubmatch(body)
	if m == nil {
		s.t.Fatalf("page has no token:\n%s", body)
	}
	return body, m[1]
}

func (s *session) wait() addResult {
	s.t.Helper()
	select {
	case res := <-s.result:
		return res
	case <-time.After(5 * time.Second):
		s.t.Fatal("session did not end")
		return addResult{}
	}
}

func pilots() *fakeOAuthConfig {
	return &fakeOAuthConfig{chars: map[string]sso.Character{
		"char-998877": {ID: 998877, Name: "Test Pilot"},
		"char-998878": {ID: 998878, Name: "Alt Pilot"},
	}}
}

func TestAddCharacters_LoginThenDone_ReturnsCharacterAndSavesToken(t *testing.T) {
	s := startSession(t, context.Background(), pilots(), "test-agent")

	resp := s.callback("char-998877", s.state())
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
		t.Fatalf("callback = %d to %q, want 303 to /", resp.StatusCode, resp.Header.Get("Location"))
	}
	if got := <-s.added; got != (sso.Character{ID: 998877, Name: "Test Pilot"}) {
		t.Errorf("OnAdded got %+v", got)
	}

	body, token := s.page()
	for _, want := range []string{"Added Test Pilot.", "characters/998877/portrait", "Added just now", "Add another character", "close this tab"} {
		if !strings.Contains(body, want) {
			t.Errorf("page is missing %q", want)
		}
	}

	resp, body = s.post("/done", token)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "All set") {
		t.Errorf("POST /done = %d, body missing 'All set'", resp.StatusCode)
	}

	res := s.wait()
	if res.err != nil {
		t.Fatalf("AddCharacters error: %v", res.err)
	}
	if want := []sso.Character{{ID: 998877, Name: "Test Pilot"}}; !reflect.DeepEqual(res.chars, want) {
		t.Errorf("characters = %+v, want %+v", res.chars, want)
	}
	if got, err := creds.CharacterRefreshToken(998877); err != nil || got != "refresh-char-998877" {
		t.Errorf("saved refresh token = %q, %v; want refresh-char-998877", got, err)
	}
}

func TestAddCharacters_AddAnother_ReturnsBothCharacters(t *testing.T) {
	s := startSession(t, context.Background(), pilots(), "test-agent")
	s.callback("char-998877", s.state())
	_, token := s.page()

	s.callback("char-998878", s.addAnother(token))

	body, token := s.page()
	if !strings.Contains(body, "Added Alt Pilot.") {
		t.Error("page doesn't confirm the second character")
	}
	s.post("/done", token)

	res := s.wait()
	want := []sso.Character{{ID: 998877, Name: "Test Pilot"}, {ID: 998878, Name: "Alt Pilot"}}
	if res.err != nil || !reflect.DeepEqual(res.chars, want) {
		t.Errorf("result = %+v, %v; want %+v", res.chars, res.err, want)
	}
}

func TestAddCharacters_MismatchedState_IgnoredAndRealLoginStillWorks(t *testing.T) {
	s := startSession(t, context.Background(), pilots(), "test-agent")
	state := s.state()

	if resp := s.callback("char-998878", "forged-state"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("forged callback status = %d, want 400", resp.StatusCode)
	}
	s.callback("char-998877", state)
	_, token := s.page()
	s.post("/done", token)

	res := s.wait()
	if want := []sso.Character{{ID: 998877, Name: "Test Pilot"}}; res.err != nil || !reflect.DeepEqual(res.chars, want) {
		t.Errorf("result = %+v, %v; want %+v", res.chars, res.err, want)
	}
}

func TestAddCharacters_StateIsSingleUse(t *testing.T) {
	s := startSession(t, context.Background(), pilots(), "test-agent")
	state := s.state()
	s.callback("char-998877", state)

	if resp := s.callback("char-998877", state); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("replayed callback status = %d, want 400", resp.StatusCode)
	}
	_, token := s.page()
	s.post("/done", token)
	s.wait()
}

func TestAddCharacters_SSOError_ShownAndReturnedWhenNothingAdded(t *testing.T) {
	s := startSession(t, context.Background(), pilots(), "test-agent")
	state := s.state()
	s.get("/callback?" + url.Values{"error": {"access_denied"}, "state": {state}}.Encode())

	body, token := s.page()
	if !strings.Contains(body, "access_denied") {
		t.Error("page doesn't show the SSO error")
	}
	s.post("/done", token)

	res := s.wait()
	if res.err == nil || !strings.Contains(res.err.Error(), "access_denied") {
		t.Errorf("err = %v, want the SSO error", res.err)
	}
}

func TestAddCharacters_PageClosed_EndsAfterGrace(t *testing.T) {
	sso.SetTimeoutsForTest(time.Minute, time.Minute, 50*time.Millisecond)
	t.Cleanup(sso.ResetTimeoutsForTest)
	s := startSession(t, context.Background(), pilots(), "test-agent")
	s.callback("char-998877", s.state())
	_, token := s.page()

	s.post("/closed", token)

	res := s.wait()
	if want := []sso.Character{{ID: 998877, Name: "Test Pilot"}}; res.err != nil || !reflect.DeepEqual(res.chars, want) {
		t.Errorf("result = %+v, %v; want %+v", res.chars, res.err, want)
	}
}

func TestAddCharacters_PageClosedDuringLogin_KeepsWaitingForCallback(t *testing.T) {
	sso.SetTimeoutsForTest(time.Minute, time.Minute, 50*time.Millisecond)
	t.Cleanup(sso.ResetTimeoutsForTest)
	s := startSession(t, context.Background(), pilots(), "test-agent")
	s.callback("char-998877", s.state())
	_, token := s.page()

	// "Add another" navigates the page to CCP, firing the close beacon.
	second := s.addAnother(token)
	s.post("/closed", token)
	time.Sleep(150 * time.Millisecond)

	if resp := s.callback("char-998878", second); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("callback after beacon = %d, want 303", resp.StatusCode)
	}
	_, token = s.page()
	s.post("/done", token)
	if res := s.wait(); len(res.chars) != 2 {
		t.Errorf("characters = %+v, want both", res.chars)
	}
}

func TestAddCharacters_PageIdle_Ends(t *testing.T) {
	sso.SetTimeoutsForTest(time.Minute, 100*time.Millisecond, time.Minute)
	t.Cleanup(sso.ResetTimeoutsForTest)
	s := startSession(t, context.Background(), pilots(), "test-agent")
	s.callback("char-998877", s.state())

	if res := s.wait(); res.err != nil || len(res.chars) != 1 {
		t.Errorf("result = %+v, %v; want the one character", res.chars, res.err)
	}
}

func TestAddCharacters_NoCallback_TimesOut(t *testing.T) {
	sso.SetTimeoutsForTest(100*time.Millisecond, time.Minute, time.Minute)
	t.Cleanup(sso.ResetTimeoutsForTest)
	s := startSession(t, context.Background(), pilots(), "test-agent")
	s.state()

	res := s.wait()
	if res.err == nil || !strings.Contains(res.err.Error(), "timed out") {
		t.Errorf("err = %v, want a timeout", res.err)
	}
}

func TestAddCharacters_ContextCancelled_ReturnsAndFreesForRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := startSession(t, ctx, pilots(), "test-agent")
	s.state()
	cancel()
	if res := s.wait(); !errors.Is(res.err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", res.err)
	}

	retry := startSession(t, context.Background(), pilots(), "test-agent")
	retry.callback("char-998877", retry.state())
	_, token := retry.page()
	retry.post("/done", token)
	if res := retry.wait(); res.err != nil {
		t.Errorf("retry err = %v", res.err)
	}
}

func TestAddCharacters_SecondCallWhileFirstOpen_Fails(t *testing.T) {
	s := startSession(t, context.Background(), pilots(), "test-agent")
	s.state()

	_, err := sso.ExportedAddCharacters(context.Background(), pilots(), "test-agent", sso.AddOptions{})
	if !errors.Is(err, sso.ErrAddCharacterInProgress) {
		t.Errorf("err = %v, want ErrAddCharacterInProgress", err)
	}

	_, token := s.page()
	s.post("/done", token)
	s.wait()
}

func TestAddCharacters_PortInUse_Fails(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer busy.Close()
	sso.SetListenForTest(busy.Addr().String(), func(string) {})
	t.Cleanup(sso.ResetListenForTest)
	sso.SetOpenBrowserForTest(func(string) error {
		t.Error("browser opened despite the port being taken")
		return nil
	})
	t.Cleanup(sso.ResetOpenBrowserForTest)

	_, err = sso.ExportedAddCharacters(context.Background(), pilots(), "test-agent", sso.AddOptions{})
	if err == nil || !strings.Contains(err.Error(), busy.Addr().String()) {
		t.Errorf("err = %v, want a listen error naming the port", err)
	}
}

func TestAddCharacters_RejectsForeignHostHeader(t *testing.T) {
	s := startSession(t, context.Background(), pilots(), "test-agent")
	s.state()

	req, err := http.NewRequest(http.MethodGet, s.base+"/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = "rebound.example:48211"
	resp, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMisdirectedRequest {
		t.Errorf("status = %d, want 421", resp.StatusCode)
	}

	_, token := s.page()
	s.post("/done", token)
	s.wait()
}

func TestAddCharacters_PostWithoutToken_Rejected(t *testing.T) {
	s := startSession(t, context.Background(), pilots(), "test-agent")
	s.state()

	for _, path := range []string{"/login", "/done", "/ping", "/closed"} {
		if resp, _ := s.post(path, "wrong"); resp.StatusCode != http.StatusForbidden {
			t.Errorf("POST %s with a bad token = %d, want 403", path, resp.StatusCode)
		}
	}

	_, token := s.page()
	s.post("/done", token)
	s.wait()
}

func TestAddCharacters_InjectsUserAgentIntoExchangeContext(t *testing.T) {
	var gotUserAgent string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
	}))
	t.Cleanup(upstream.Close)

	fake := pilots()
	fake.upstreamURL = upstream.URL
	s := startSession(t, context.Background(), fake, "neat/test-ua-check")
	s.callback("char-998877", s.state())
	_, token := s.page()
	s.post("/done", token)
	if res := s.wait(); res.err != nil {
		t.Fatalf("AddCharacters error: %v", res.err)
	}
	if gotUserAgent != "neat/test-ua-check" {
		t.Errorf("upstream received User-Agent = %q, want %q", gotUserAgent, "neat/test-ua-check")
	}
}

// TestAddCharacters_ReAdd_EvictsCachedClient guards against re-adding a
// character (e.g. after CCP revoked its refresh token) leaving
// AuthenticatedClient's cached client -- built around the old, dead token
// -- in place. The new token must be persisted AND the cache entry evicted,
// so the next AuthenticatedClient call rebuilds from the new token instead
// of failing with "invalid_grant" until the process restarts.
func TestAddCharacters_ReAdd_EvictsCachedClient(t *testing.T) {
	const characterID = 998877
	s := startSession(t, context.Background(), pilots(), "test-agent")
	state := s.state()

	if err := creds.SaveCharacterRefreshToken(characterID, "old-dead-token"); err != nil {
		t.Fatalf("SaveCharacterRefreshToken: %v", err)
	}
	stale, err := sso.AuthenticatedClient(context.Background(), "client-id", characterID, "test-agent")
	if err != nil {
		t.Fatalf("AuthenticatedClient before re-add: %v", err)
	}

	s.callback("char-998877", state)
	_, token := s.page()
	s.post("/done", token)
	if res := s.wait(); res.err != nil {
		t.Fatalf("AddCharacters: %v", res.err)
	}

	if got, err := creds.CharacterRefreshToken(characterID); err != nil || got != "refresh-char-998877" {
		t.Errorf("persisted refresh token = %q, %v; want refresh-char-998877", got, err)
	}
	fresh, err := sso.AuthenticatedClient(context.Background(), "client-id", characterID, "test-agent")
	if err != nil {
		t.Fatalf("AuthenticatedClient after re-add: %v", err)
	}
	if fresh == stale {
		t.Error("AuthenticatedClient returned the stale cached client after re-add, want a fresh one built from the new token")
	}
}
