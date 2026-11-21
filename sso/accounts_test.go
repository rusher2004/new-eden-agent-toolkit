package sso_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// accountBook is an in-memory stand-in for neat's config: the saved
// characters and their account labels, edited through sso.AccountEditor.
type accountBook struct {
	mu    sync.Mutex
	chars []sso.Character
}

func (b *accountBook) saved() ([]sso.Character, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]sso.Character(nil), b.chars...), nil
}

func (b *accountBook) add(c sso.Character) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, e := range b.chars {
		if e.ID == c.ID {
			return nil
		}
	}
	b.chars = append(b.chars, sso.Character{ID: c.ID, Name: c.Name})
	return nil
}

func (b *accountBook) account(id int32) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, c := range b.chars {
		if c.ID == id {
			return c.Account
		}
	}
	return "?"
}

func (b *accountBook) editor() *sso.AccountEditor {
	return &sso.AccountEditor{
		Set: func(id int32, label string) error {
			b.mu.Lock()
			defer b.mu.Unlock()
			if len(label) > 40 {
				return errors.New("too long")
			}
			for i := range b.chars {
				if b.chars[i].ID == id {
					b.chars[i].Account = strings.TrimSpace(label)
					return nil
				}
			}
			return errors.New("no such character")
		},
		Rename: func(from, to string) error {
			b.mu.Lock()
			defer b.mu.Unlock()
			n := 0
			for i := range b.chars {
				if b.chars[i].Account == from {
					b.chars[i].Account = to
					n++
				}
			}
			if n == 0 {
				return errors.New("no such account")
			}
			return nil
		},
		Delete: func(label string) error {
			b.mu.Lock()
			defer b.mu.Unlock()
			for i := range b.chars {
				if b.chars[i].Account == label {
					b.chars[i].Account = ""
				}
			}
			return nil
		},
	}
}

func startAccountSession(t *testing.T, book *accountBook) *session {
	t.Helper()
	var s *session
	s = startSessionWith(t, context.Background(), pilots(), "test-agent", nil, func(o *sso.AddOptions) {
		next := o.OnAdded
		o.OnAdded = func(c sso.Character) error {
			if err := book.add(c); err != nil {
				return err
			}
			return next(c)
		}
		o.SavedCharacters = book.saved
		o.Accounts = book.editor()
	})
	return s
}

func (s *session) postForm(path string, form url.Values) *http.Response {
	s.t.Helper()
	resp, err := s.client.PostForm(s.base+path, form)
	if err != nil {
		s.t.Fatalf("POST %s: %v", path, err)
	}
	resp.Body.Close()
	return resp
}

// groupOf returns the heading of the account group the character's tile is
// in on the page, or "" if it isn't on the page.
func groupOf(body string, id string) string {
	sections := strings.Split(body, `<section class="account`)
	h2 := regexp.MustCompile(`<h2[^>]*>([^<]*)`)
	for _, sec := range sections[1:] {
		if strings.Contains(sec, "characters/"+id+"/portrait") {
			if m := h2.FindStringSubmatch(sec); m != nil {
				return strings.TrimSpace(m[1])
			}
		}
	}
	return ""
}

func TestAccounts_PageGroupsCharactersAndMovesThem(t *testing.T) {
	book := &accountBook{chars: []sso.Character{{ID: 1, Name: "Main", Account: "Main account"}}}
	s := startAccountSession(t, book)
	s.callback("char-998877", s.state())

	body, token := s.page()
	if g := groupOf(body, "1"); g != "Main account" {
		t.Errorf("Main is in %q", g)
	}
	if g := groupOf(body, "998877"); g != "Not assigned to an account" {
		t.Errorf("new character is in %q, want unassigned (the character before it was saved earlier, not logged in this session)", g)
	}

	resp := s.postForm("/account/set", url.Values{"token": {token}, "id": {"998877"}, "account": {"new"}, "new_account": {"  Alt   account "}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
		t.Fatalf("POST /account/set = %d to %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if got := book.account(998877); got != "Alt   account" {
		t.Errorf("saved account = %q", got)
	}
	body, _ = s.page()
	if !strings.Contains(body, "Moved Test Pilot to Alt account.") {
		t.Error("page doesn't confirm the move")
	}

	s.postForm("/account/set", url.Values{"token": {token}, "id": {"998877"}, "account": {"existing:Main account"}})
	if got := book.account(998877); got != "Main account" {
		t.Errorf("after moving to an existing account = %q", got)
	}
}

func TestAccounts_SuggestsPreviousLoginsAccountUntilConfirmed(t *testing.T) {
	book := &accountBook{}
	s := startAccountSession(t, book)
	s.callback("char-998877", s.state())
	_, token := s.page()
	s.postForm("/account/set", url.Values{"token": {token}, "id": {"998877"}, "account": {"new"}, "new_account": {"Main"}})

	s.callback("char-998878", s.addAnother(token))
	body, token := s.page()
	if g := groupOf(body, "998878"); g != "Main" {
		t.Errorf("second login is shown in %q, want the suggested Main", g)
	}
	if !strings.Contains(body, ">Same account</button>") {
		t.Error("the suggestion isn't marked")
	}
	if got := book.account(998878); got != "" {
		t.Errorf("a suggestion was saved before it was confirmed: %q", got)
	}

	s.postForm("/account/set", url.Values{"token": {token}, "id": {"998878"}, "account": {"existing:Main"}})
	if got := book.account(998878); got != "Main" {
		t.Errorf("confirmed account = %q", got)
	}
	if body, _ := s.page(); strings.Contains(body, ">Same account</button>") {
		t.Error("still suggesting after confirmation")
	}
}

func TestAccounts_DecliningSuggestionKeepsItUnassigned(t *testing.T) {
	book := &accountBook{}
	s := startAccountSession(t, book)
	s.callback("char-998877", s.state())
	_, token := s.page()
	s.postForm("/account/set", url.Values{"token": {token}, "id": {"998877"}, "account": {"new"}, "new_account": {"Main"}})
	s.callback("char-998878", s.addAnother(token))
	_, token = s.page()

	s.postForm("/account/set", url.Values{"token": {token}, "id": {"998878"}, "account": {"none"}})
	body, _ := s.page()
	if g := groupOf(body, "998878"); g != "Not assigned to an account" || strings.Contains(body, ">Same account</button>") {
		t.Errorf("after declining, character is in %q and suggested=%v", g, strings.Contains(body, ">Same account</button>"))
	}
}

func TestAccounts_RenameAndDelete(t *testing.T) {
	book := &accountBook{chars: []sso.Character{{ID: 1, Name: "Main", Account: "A"}, {ID: 2, Name: "Alt", Account: "A"}}}
	s := startAccountSession(t, book)
	_, token := s.page()

	s.postForm("/account/rename", url.Values{"token": {token}, "from": {"A"}, "to": {"Main account"}})
	if book.account(1) != "Main account" || book.account(2) != "Main account" {
		t.Errorf("after rename: %q, %q", book.account(1), book.account(2))
	}
	s.postForm("/account/delete", url.Values{"token": {token}, "label": {"Main account"}})
	if book.account(1) != "" || book.account(2) != "" {
		t.Errorf("after delete: %q, %q", book.account(1), book.account(2))
	}
	body, _ := s.page()
	if !strings.Contains(body, "Deleted Main account.") {
		t.Error("page doesn't confirm the delete")
	}
}

func TestAccounts_ErrorsShownOnPage(t *testing.T) {
	book := &accountBook{chars: []sso.Character{{ID: 1, Name: "Main"}}}
	s := startAccountSession(t, book)
	_, token := s.page()

	s.postForm("/account/set", url.Values{"token": {token}, "id": {"1"}, "account": {"new"}, "new_account": {"  "}})
	if body, _ := s.page(); !strings.Contains(body, "Type a name for the new account.") {
		t.Error("an empty new account name isn't reported")
	}
	s.postForm("/account/rename", url.Values{"token": {token}, "from": {"Nobody"}, "to": {"X"}})
	if body, _ := s.page(); !strings.Contains(body, "couldn&#39;t rename Nobody") {
		t.Error("a failed rename isn't reported")
	}
	if resp := s.postForm("/account/set", url.Values{"token": {token}, "id": {"424242"}, "account": {"existing:X"}}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("a character not on the page = %d, want 400", resp.StatusCode)
	}
}

func TestAccounts_RequireToken(t *testing.T) {
	book := &accountBook{chars: []sso.Character{{ID: 1, Name: "Main"}}}
	s := startAccountSession(t, book)
	for _, path := range []string{"/account/set", "/account/rename", "/account/delete"} {
		if resp := s.postForm(path, url.Values{"id": {"1"}, "account": {"Stolen"}, "from": {"x"}, "to": {"y"}, "label": {"x"}}); resp.StatusCode != http.StatusForbidden {
			t.Errorf("POST %s without token = %d, want 403", path, resp.StatusCode)
		}
	}
	if book.account(1) != "" {
		t.Error("a POST without the page token changed an account")
	}
}

func TestAccounts_LabelNamedLikeAChoiceIsAnOrdinaryAccount(t *testing.T) {
	for _, label := range []string{"new", "none", "__new__", "existing:x"} {
		t.Run(label, func(t *testing.T) {
			book := &accountBook{chars: []sso.Character{{ID: 1, Name: "Main", Account: label}, {ID: 2, Name: "Alt"}}}
			s := startAccountSession(t, book)
			_, token := s.page()
			s.postForm("/account/set", url.Values{"token": {token}, "id": {"2"}, "account": {"existing:" + label}})
			if got := book.account(2); got != label {
				t.Errorf("Alt's account = %q, want %q", got, label)
			}
		})
	}
}

func TestAccounts_NoControlsWithoutEditor(t *testing.T) {
	s := startSession(t, context.Background(), pilots(), "test-agent")
	s.callback("char-998877", s.state())
	body, token := s.page()
	if strings.Contains(body, `action="/account/set"`) || strings.Contains(body, "Not assigned to an account") {
		t.Error("account controls shown without an AccountEditor")
	}
	if resp := s.postForm("/account/set", url.Values{"token": {token}, "id": {"998877"}, "account": {"existing:X"}}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST /account/set without an editor = %d, want 404", resp.StatusCode)
	}
}
