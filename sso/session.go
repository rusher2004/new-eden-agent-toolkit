package sso

import (
	"context"
	"crypto/subtle"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

//go:embed web/page.html web/page.js
var webFS embed.FS

var pageTemplate = template.Must(template.ParseFS(webFS, "web/page.html"))

// errSessionTimedOut is what wait returns when a session expires on its own.
var errSessionTimedOut = errors.New("timed out waiting for the browser login to complete")

// notice is the one message a session page shows above the character list.
type notice struct {
	Kind string // "success" or "error"
	Text string
}

// loginSession is the state behind one AddCharacters call's local web page.
// Every field below mu is guarded by it.
type loginSession struct {
	ctx        context.Context // carries the User-Agent HTTP client for Exchange
	cfg        oauthConfig
	opts       AddOptions
	host       string // the only Host header the server answers to
	token      string // anti-CSRF token every POST must carry
	authOrigin string // FC's login origin, allowed as a form-action target

	wake chan struct{} // signalled whenever expires or finished changes

	mu       sync.Mutex
	pending  string // state of the login in flight; "" when none is
	added    []Character
	notice   notice
	lastErr  error
	expires  time.Time
	finished bool
}

func newLoginSession(ctx context.Context, cfg oauthConfig, opts AddOptions, host string) (*loginSession, error) {
	token, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generate page token: %w", err)
	}
	authURL, err := url.Parse(cfg.AuthURL("", opts.Scopes))
	if err != nil {
		return nil, fmt.Errorf("parse auth URL: %w", err)
	}
	return &loginSession{
		ctx:        ctx,
		cfg:        cfg,
		opts:       opts,
		host:       host,
		token:      token,
		authOrigin: authURL.Scheme + "://" + authURL.Host,
		wake:       make(chan struct{}, 1),
	}, nil
}

// signal wakes wait so it re-reads expires and finished.
func (s *loginSession) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// expireIn sets the session to expire d from now. Callers hold mu.
func (s *loginSession) expireIn(d time.Duration) {
	s.expires = time.Now().Add(d)
	s.signal()
}

// extendBy pushes expiry out to d from now, never pulling it in. Callers
// hold mu.
func (s *loginSession) extendBy(d time.Duration) {
	if t := time.Now().Add(d); t.After(s.expires) {
		s.expires = t
		s.signal()
	}
}

// startLogin begins a new login, replacing any one still in flight, and
// returns the FC URL to send the browser to.
func (s *loginSession) startLogin() (string, error) {
	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = state
	s.expireIn(callbackTimeout)
	return s.cfg.AuthURL(state, s.opts.Scopes), nil
}

// wait blocks until the session finishes, expires, or ctx is done.
func (s *loginSession) wait(ctx context.Context) error {
	for {
		s.mu.Lock()
		finished, remaining := s.finished, time.Until(s.expires)
		s.mu.Unlock()
		if finished {
			return nil
		}
		if remaining <= 0 {
			return errSessionTimedOut
		}
		timer := time.NewTimer(remaining)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-s.wake:
			timer.Stop()
		case <-timer.C:
		}
	}
}

// result returns the characters added and the most recent login error.
func (s *loginSession) result() ([]Character, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Character(nil), s.added...), s.lastErr
}

func (s *loginSession) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handlePage)
	mux.HandleFunc("GET /callback", s.handleCallback)
	mux.HandleFunc("GET /page.js", s.handleScript)
	mux.HandleFunc("POST /login", s.requireToken(s.handleLogin))
	mux.HandleFunc("POST /done", s.requireToken(s.handleDone))
	mux.HandleFunc("POST /ping", s.requireToken(s.handlePing))
	mux.HandleFunc("POST /closed", s.requireToken(s.handleClosed))

	csp := "default-src 'none'; img-src https://images.evetech.net; style-src 'unsafe-inline'; " +
		"script-src 'self'; connect-src 'self'; form-action 'self' " + s.authOrigin +
		"; frame-ancestors 'none'; base-uri 'none'"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Answering only our own Host header defeats DNS rebinding: a
		// hostile page that rebinds its own name to 127.0.0.1 still sends
		// its own name as Host, so it can't read this page's character list.
		if r.Host != s.host {
			http.Error(w, "unexpected Host header", http.StatusMisdirectedRequest)
			return
		}
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}

// requireToken rejects a POST that doesn't carry the session's page token,
// so another web page can't drive the session through the user's browser.
func (s *loginSession) requireToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.PostFormValue("token")), []byte(s.token)) != 1 {
			http.Error(w, "missing or invalid page token", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (s *loginSession) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	state := q.Get("state")

	s.mu.Lock()
	if s.pending == "" || subtle.ConstantTimeCompare([]byte(state), []byte(s.pending)) != 1 {
		// Leave any login in flight alone: a stray or forged callback
		// mustn't cancel the user's real one.
		s.mu.Unlock()
		s.render(w, http.StatusBadRequest, notice{Kind: "error", Text: "This login response doesn't match the login neat started, so it was ignored. Use Add another character to try again."}, false)
		return
	}
	expected := s.pending
	s.pending = ""
	s.mu.Unlock()

	char, err := s.completeLogin(q, expected)

	s.mu.Lock()
	if err != nil {
		s.lastErr = err
		s.notice = notice{Kind: "error", Text: err.Error()}
	} else {
		s.recordAdded(char)
		s.notice = notice{Kind: "success", Text: "Added " + char.Name + "."}
	}
	s.expireIn(pageIdleTimeout)
	s.mu.Unlock()

	// Redirect rather than render, so a reload doesn't replay the
	// single-use authorization code.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// completeLogin exchanges a callback's code for tokens and saves the
// character.
func (s *loginSession) completeLogin(q url.Values, expectedState string) (Character, error) {
	if e := q.Get("error"); e != "" {
		if d := q.Get("error_description"); d != "" {
			e += ": " + d
		}
		return Character{}, fmt.Errorf("EVE login was not completed (%s)", e)
	}
	code := q.Get("code")
	if code == "" {
		return Character{}, errors.New("EVE login response is missing its authorization code")
	}
	token, claims, err := s.cfg.Exchange(s.ctx, code, q.Get("state"), expectedState)
	if err != nil {
		return Character{}, fmt.Errorf("exchange authorization code: %w", err)
	}
	characterID, err := characterIDFromClaims(claims)
	if err != nil {
		return Character{}, err
	}
	if err := replaceCharacterRefreshToken(characterID, token.RefreshToken); err != nil {
		return Character{}, err
	}
	char := Character{ID: characterID, Name: claims.Name, Scopes: []string(claims.Scopes)}
	if s.opts.OnAdded != nil {
		if err := s.opts.OnAdded(char); err != nil {
			return Character{}, fmt.Errorf("record %s: %w", char.Name, err)
		}
	}
	return char, nil
}

// recordAdded adds c to the session's results, replacing an earlier entry
// for the same character. Callers hold mu.
func (s *loginSession) recordAdded(c Character) {
	for i, existing := range s.added {
		if existing.ID == c.ID {
			s.added[i] = c
			return
		}
	}
	s.added = append(s.added, c)
}

func (s *loginSession) handlePage(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.extendBy(pageIdleTimeout)
	n := s.notice
	// A success shows once, as a toast; the character's "Added just now"
	// badge carries it after that. An error stays until the next login.
	if n.Kind == "success" {
		s.notice = notice{}
	}
	s.mu.Unlock()
	s.render(w, http.StatusOK, n, false)
}

func (s *loginSession) handleLogin(w http.ResponseWriter, r *http.Request) {
	authURL, err := s.startLogin()
	if err != nil {
		s.render(w, http.StatusInternalServerError, notice{Kind: "error", Text: err.Error()}, false)
		return
	}
	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func (s *loginSession) handleDone(w http.ResponseWriter, _ *http.Request) {
	s.render(w, http.StatusOK, notice{}, true)
	s.mu.Lock()
	s.finished = true
	s.signal()
	s.mu.Unlock()
}

func (s *loginSession) handlePing(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.extendBy(pageIdleTimeout)
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// handleClosed is the page's beacon on being hidden for good (tab closed,
// navigated away, or reloaded). It shortens the session to closeGrace
// unless a login is in flight, since navigating to FC's login page fires
// the same beacon.
func (s *loginSession) handleClosed(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	if s.pending == "" {
		if t := time.Now().Add(closeGrace); t.Before(s.expires) {
			s.expires = t
			s.signal()
		}
	}
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *loginSession) handleScript(w http.ResponseWriter, _ *http.Request) {
	js, err := webFS.ReadFile("web/page.js")
	if err != nil {
		http.Error(w, "page script missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	if _, err := w.Write(js); err != nil {
		slog.Default().WarnContext(s.ctx, "sso: failed to write page script", "error", err)
	}
}

type pageCharacter struct {
	ID   int32
	Name string
	New  bool
}

type pageData struct {
	Token      string
	Notice     notice
	Characters []pageCharacter
	Finished   bool
}

// characters lists the saved characters, marking those added in this
// session, plus any added character the saved list doesn't have yet.
func (s *loginSession) characters() []pageCharacter {
	s.mu.Lock()
	added := append([]Character(nil), s.added...)
	s.mu.Unlock()

	var saved []Character
	if s.opts.SavedCharacters != nil {
		var err error
		saved, err = s.opts.SavedCharacters()
		if err != nil {
			slog.Default().WarnContext(s.ctx, "sso: failed to list saved characters for the login page", "error", err)
			saved = nil
		}
	}

	isNew := make(map[int32]bool, len(added))
	for _, c := range added {
		isNew[c.ID] = true
	}
	seen := make(map[int32]bool, len(saved))
	out := make([]pageCharacter, 0, len(saved)+len(added))
	for _, c := range saved {
		seen[c.ID] = true
		out = append(out, pageCharacter{ID: c.ID, Name: c.Name, New: isNew[c.ID]})
	}
	for _, c := range added {
		if !seen[c.ID] {
			out = append(out, pageCharacter{ID: c.ID, Name: c.Name, New: true})
		}
	}
	return out
}

func (s *loginSession) render(w http.ResponseWriter, status int, n notice, finished bool) {
	data := pageData{Notice: n, Characters: s.characters(), Finished: finished}
	if !finished {
		data.Token = s.token
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := pageTemplate.Execute(w, data); err != nil {
		slog.Default().WarnContext(s.ctx, "sso: failed to render login page", "error", err)
	}
}
