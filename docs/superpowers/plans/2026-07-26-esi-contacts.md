# ESI Contacts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cover the Contacts ESI resource group (9 endpoints: 6 reads across character/corp/alliance scope + 3 character-scoped writes — the project's 2nd/3rd/4th write endpoints).

**Architecture:** `esi/contacts.go` is a thin, `describeError`-wrapped facade. `mcp/contacts.go` adds 9 tools, all reusing the existing `characterExists(store, characterID) (bool, error)` helper already defined in `mcp/character.go` (package-level, visible to every file in package `mcp` — `contacts.go` must NOT redeclare it). The 3 write tools follow the established verb-first, effect-stating convention from `respond_calendar_event`.

**Tech Stack:** Go 1.26.1, `github.com/fnt-eve/goesi-openapi` v0.0.37, `github.com/modelcontextprotocol/go-sdk` v1.6.1.

## Global Constraints

- Go 1.26.1, module `github.com/rusher2004/new-eden-agent-toolkit`, flat package layout, no `internal/`.
- Wrap every error with context. Never discard an error.
- Every `esi/` method routes errors through `describeError`.
- External test packages by default; standard library only in tests.
- **`mcp/contacts.go` must call the existing `characterExists` helper (from `mcp/character.go`), not redefine it.** Redeclaring a function with the same name in the same package is a compile error — the helper is already package-visible.
- The 3 write tools (`add_character_contacts`, `update_character_contacts`, `remove_character_contacts`) are verb-first named with effect-stating descriptions, matching `respond_calendar_event`'s precedent. No client-side `standing` range validation — ESI enforces -10.0 to 10.0 server-side with a clear error; duplicating that check client-side isn't warranted (unlike `respond_calendar_event`'s validation, which exists because ESI's error for a bad `response` string is comparatively opaque).
- `remove_character_contacts` sends `contact_ids` as a query parameter via `.ContactIds()`, NOT a request body — verified live via `httptest.Server` that DELETE has no body at all.
- POST/PUT send `standing`/`label_ids`/`watched` as query parameters (via `.Standing()`/`.LabelIds()`/`.Watched()` builder methods) and the contact ID list as the request body (via `.RequestBody()`) — verified live, do not conflate the two.
- This branch is the shared PR branch (`worktree-esi-access-list-activities`), current baseline 39 tools.
- Design spec: `docs/superpowers/specs/2026-07-26-esi-contacts-design.md`.
- All code verified to compile against the real installed `github.com/fnt-eve/goesi-openapi@v0.0.37` and `github.com/modelcontextprotocol/go-sdk` v1.6.1 modules, including a live `httptest.Server` round-trip confirming the exact query-param/body split for POST/PUT/DELETE.

---

### Task 1: `esi/contacts.go` — Contacts facade (6 reads + 3 writes)

**Files:**
- Create: `esi/contacts.go`
- Create: `esi/contacts_test.go`

**Interfaces:**
- Consumes: `esi.Client`, `describeError`; `esiapi.APIClient.ContactsAPI.GetCharactersCharacterIdContacts(ctx, characterId int64) ...Execute() ([]esiapi.CharactersCharacterIdContactsGetInner, *http.Response, error)`, `.GetCharactersCharacterIdContactsLabels(ctx, characterId int64) ...Execute() ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, *http.Response, error)`, `.GetCorporationsCorporationIdContacts(ctx, corporationId int64) ...Execute() ([]esiapi.CorporationsCorporationIdContactsGetInner, *http.Response, error)`, `.GetCorporationsCorporationIdContactsLabels(ctx, corporationId int64) ...Execute() ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, *http.Response, error)`, `.GetAlliancesAllianceIdContacts(ctx, allianceId int64) ...Execute() ([]esiapi.AlliancesAllianceIdContactsGetInner, *http.Response, error)`, `.GetAlliancesAllianceIdContactsLabels(ctx, allianceId int64) ...Execute() ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, *http.Response, error)`, `.PostCharactersCharacterIdContacts(ctx, characterId int64).Standing(f).LabelIds(ids).Watched(b).RequestBody(ids) ...Execute() ([]int64, *http.Response, error)`, `.PutCharactersCharacterIdContacts(ctx, characterId int64).Standing(f).LabelIds(ids).Watched(b).RequestBody(ids) ...Execute() (*http.Response, error)`, `.DeleteCharactersCharacterIdContacts(ctx, characterId int64).ContactIds(ids) ...Execute() (*http.Response, error)` — all verified against the real installed module, including live request-shape confirmation.
- Produces: `(*esi.Client).CharacterContacts`, `.CharacterContactLabels`, `.CorporationContacts`, `.CorporationContactLabels`, `.AllianceContacts`, `.AllianceContactLabels`, `.AddCharacterContacts`, `.UpdateCharacterContacts`, `.RemoveCharacterContacts` — all consumed by Task 2.

- [ ] **Step 1: Write the failing tests**

Create `esi/contacts_test.go`:

```go
package esi_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	goesi "github.com/fnt-eve/goesi-openapi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func TestCharacterContacts_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"contact_id":998878,"contact_type":"character","standing":5.5}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contacts, err := client.CharacterContacts(context.Background())
	if err != nil {
		t.Fatalf("CharacterContacts returned error: %v", err)
	}
	if len(contacts) != 1 || contacts[0].Standing != 5.5 {
		t.Errorf("contacts = %+v, want one entry with Standing=5.5", contacts)
	}
}

func TestCharacterContactLabels_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"label_id":1,"label_name":"Friends"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	labels, err := client.CharacterContactLabels(context.Background())
	if err != nil {
		t.Fatalf("CharacterContactLabels returned error: %v", err)
	}
	if len(labels) != 1 || labels[0].LabelName != "Friends" {
		t.Errorf("labels = %+v, want one entry with LabelName=\"Friends\"", labels)
	}
}

func TestCorporationContacts_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"contact_id":998879,"contact_type":"character","standing":1.0}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contacts, err := client.CorporationContacts(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationContacts returned error: %v", err)
	}
	if len(contacts) != 1 || contacts[0].ContactId != 998879 {
		t.Errorf("contacts = %+v, want one entry with ContactId=998879", contacts)
	}
}

func TestCorporationContactLabels_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"label_id":2,"label_name":"Corp VIP"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	labels, err := client.CorporationContactLabels(context.Background(), 98000001)
	if err != nil {
		t.Fatalf("CorporationContactLabels returned error: %v", err)
	}
	if len(labels) != 1 || labels[0].LabelName != "Corp VIP" {
		t.Errorf("labels = %+v, want one entry with LabelName=\"Corp VIP\"", labels)
	}
}

func TestAllianceContacts_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"contact_id":998880,"contact_type":"corporation","standing":-5.0}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	contacts, err := client.AllianceContacts(context.Background(), 99000001)
	if err != nil {
		t.Fatalf("AllianceContacts returned error: %v", err)
	}
	if len(contacts) != 1 || contacts[0].Standing != -5.0 {
		t.Errorf("contacts = %+v, want one entry with Standing=-5.0", contacts)
	}
}

func TestAllianceContactLabels_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"label_id":3,"label_name":"Alliance Blues"}]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	labels, err := client.AllianceContactLabels(context.Background(), 99000001)
	if err != nil {
		t.Fatalf("AllianceContactLabels returned error: %v", err)
	}
	if len(labels) != 1 || labels[0].LabelName != "Alliance Blues" {
		t.Errorf("labels = %+v, want one entry with LabelName=\"Alliance Blues\"", labels)
	}
}

func TestAddCharacterContacts_SendsCorrectQueryAndBody(t *testing.T) {
	var gotQuery, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[998878]`))
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	added, err := client.AddCharacterContacts(context.Background(), []int64{998878}, 5.0, []int64{1}, true)
	if err != nil {
		t.Fatalf("AddCharacterContacts returned error: %v", err)
	}
	if gotQuery != "label_ids=1&standing=5&watched=true" {
		t.Errorf("query = %q, want \"label_ids=1&standing=5&watched=true\"", gotQuery)
	}
	if gotBody != "[998878]" && gotBody != "[998878]\n" {
		t.Errorf("body = %q, want \"[998878]\" (bare array, not wrapped)", gotBody)
	}
	if len(added) != 1 || added[0] != 998878 {
		t.Errorf("added = %v, want [998878]", added)
	}
}

func TestUpdateCharacterContacts_SendsCorrectQueryAndHandlesNoContent(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if err := client.UpdateCharacterContacts(context.Background(), []int64{998878}, 3.0, nil, false); err != nil {
		t.Fatalf("UpdateCharacterContacts returned error: %v", err)
	}
	if gotQuery != "standing=3" {
		t.Errorf("query = %q, want \"standing=3\"", gotQuery)
	}
}

func TestRemoveCharacterContacts_SendsContactIDsAsQueryParamNoBody(t *testing.T) {
	var gotQuery, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if err := client.RemoveCharacterContacts(context.Background(), []int64{998878}); err != nil {
		t.Fatalf("RemoveCharacterContacts returned error: %v", err)
	}
	if gotQuery != "contact_ids=998878" {
		t.Errorf("query = %q, want \"contact_ids=998878\"", gotQuery)
	}
	if gotBody != "" {
		t.Errorf("body = %q, want empty (DELETE has no body)", gotBody)
	}
}

func TestRemoveCharacterContacts_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{BaseURL: srv.URL, UserAgent: "test-agent"})

	if err := client.RemoveCharacterContacts(context.Background(), []int64{998878}); err == nil {
		t.Fatal("RemoveCharacterContacts returned nil error, want an error for HTTP 403")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./esi/... -run 'TestCharacterContacts|TestCharacterContactLabels|TestCorporationContacts|TestCorporationContactLabels|TestAllianceContacts|TestAllianceContactLabels|TestAddCharacterContacts|TestUpdateCharacterContacts|TestRemoveCharacterContacts' -v`

Expected: FAIL to compile.

- [ ] **Step 3: Implement `esi/contacts.go`**

```go
package esi

import (
	"context"
	"fmt"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// CharacterContacts returns the client's character's contacts.
func (c *Client) CharacterContacts(ctx context.Context) ([]esiapi.CharactersCharacterIdContactsGetInner, error) {
	result, resp, err := c.api.ContactsAPI.GetCharactersCharacterIdContacts(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character contacts: %w", describeError(err, resp))
	}
	return result, nil
}

// CharacterContactLabels returns the client's character's contact labels.
func (c *Client) CharacterContactLabels(ctx context.Context) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
	result, resp, err := c.api.ContactsAPI.GetCharactersCharacterIdContactsLabels(ctx, int64(c.characterID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("get character contact labels: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationContacts returns corporationID's contacts. The authenticated character must hold
// sufficient corp roles — ESI itself returns 403 otherwise; no client-side check needed.
func (c *Client) CorporationContacts(ctx context.Context, corporationID int64) ([]esiapi.CorporationsCorporationIdContactsGetInner, error) {
	result, resp, err := c.api.ContactsAPI.GetCorporationsCorporationIdContacts(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation contacts: %w", describeError(err, resp))
	}
	return result, nil
}

// CorporationContactLabels returns corporationID's contact labels.
func (c *Client) CorporationContactLabels(ctx context.Context, corporationID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
	result, resp, err := c.api.ContactsAPI.GetCorporationsCorporationIdContactsLabels(ctx, corporationID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get corporation contact labels: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceContacts returns allianceID's contacts.
func (c *Client) AllianceContacts(ctx context.Context, allianceID int64) ([]esiapi.AlliancesAllianceIdContactsGetInner, error) {
	result, resp, err := c.api.ContactsAPI.GetAlliancesAllianceIdContacts(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance contacts: %w", describeError(err, resp))
	}
	return result, nil
}

// AllianceContactLabels returns allianceID's contact labels.
func (c *Client) AllianceContactLabels(ctx context.Context, allianceID int64) ([]esiapi.AlliancesAllianceIdContactsLabelsGetInner, error) {
	result, resp, err := c.api.ContactsAPI.GetAlliancesAllianceIdContactsLabels(ctx, allianceID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get alliance contact labels: %w", describeError(err, resp))
	}
	return result, nil
}

// AddCharacterContacts adds contactIDs to the client's character's contact list with the given
// standing, optional label IDs, and watched flag. Returns the created contact IDs. This is a
// write: it changes state on CCP's server.
func (c *Client) AddCharacterContacts(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error) {
	result, resp, err := c.api.ContactsAPI.PostCharactersCharacterIdContacts(ctx, int64(c.characterID)).
		Standing(standing).LabelIds(labelIDs).Watched(watched).RequestBody(contactIDs).Execute()
	if err != nil {
		return nil, fmt.Errorf("add character contacts: %w", describeError(err, resp))
	}
	return result, nil
}

// UpdateCharacterContacts updates the standing/labels/watched flag of existing contacts. This is
// a write: it changes state on CCP's server. ESI returns no response body (204).
func (c *Client) UpdateCharacterContacts(ctx context.Context, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error {
	resp, err := c.api.ContactsAPI.PutCharactersCharacterIdContacts(ctx, int64(c.characterID)).
		Standing(standing).LabelIds(labelIDs).Watched(watched).RequestBody(contactIDs).Execute()
	if err != nil {
		return fmt.Errorf("update character contacts: %w", describeError(err, resp))
	}
	return nil
}

// RemoveCharacterContacts removes contactIDs from the client's character's contact list.
// contactIDs is sent as a query parameter, not a request body — verified against the real
// installed module. This is a write: it changes state on CCP's server. ESI returns no response
// body (204).
func (c *Client) RemoveCharacterContacts(ctx context.Context, contactIDs []int64) error {
	resp, err := c.api.ContactsAPI.DeleteCharactersCharacterIdContacts(ctx, int64(c.characterID)).
		ContactIds(contactIDs).Execute()
	if err != nil {
		return fmt.Errorf("remove character contacts: %w", describeError(err, resp))
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go build ./... && go vet ./... && go test ./esi/... -race -v`

Expected: all 10 new tests PASS, plus every pre-existing `esi` test.

- [ ] **Step 5: Commit**

```bash
git add esi/contacts.go esi/contacts_test.go
git commit -m "feat: add esi/contacts.go facade (2nd/3rd/4th write methods)"
```

---

### Task 2: `mcp/contacts.go` tools + registration + docs sync

**Files:**
- Create: `mcp/contacts.go`
- Create: `mcp/contacts_internal_test.go`
- Modify: `mcp/server.go`
- Modify: `mcp/server_test.go`
- Modify: `CLAUDE.md`
- Modify: `README.md`
- Modify: `docs/reference/esi-coverage-roadmap.md`

**Interfaces:**
- Consumes: `esi.NewClient`, all 9 `(*esi.Client).*Contact*` methods (Task 1); `sso.UserAgent`, `config.Store`, `ErrCharacterNotFound`, and **the existing `characterExists(store *config.Store, characterID int32) (bool, error)` helper already defined in `mcp/character.go`** (existing — this file must call it, not redeclare it).
- Produces: 9 new tool Input/Output types plus `mcp.Contact`, `mcp.ContactLabel` shared types. Nothing later depends on these.

- [ ] **Step 1: Write the failing tests**

Create `mcp/contacts_internal_test.go`:

```go
package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
)

func TestListCharacterContactsHandler_ReturnsContacts(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContacts
	t.Cleanup(func() { listCharacterContacts = orig })
	listCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Contact, error) {
		return []Contact{{ContactID: 998878, Standing: 5.5}}, nil
	}

	handler := listCharacterContactsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterContactsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Contacts) != 1 || out.Contacts[0].Standing != 5.5 {
		t.Errorf("out = %+v, want one contact with Standing=5.5", out)
	}
}

func TestListCharacterContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterContacts
	t.Cleanup(func() { listCharacterContacts = orig })
	listCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Contact, error) {
		t.Fatal("listCharacterContacts should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterContactsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterContactsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCharacterContactLabelsHandler_ReturnsLabels(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCharacterContactLabels
	t.Cleanup(func() { listCharacterContactLabels = orig })
	listCharacterContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]ContactLabel, error) {
		return []ContactLabel{{LabelID: 1, LabelName: "Friends"}}, nil
	}

	handler := listCharacterContactLabelsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCharacterContactLabelsInput{CharacterID: 7})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Labels) != 1 || out.Labels[0].LabelName != "Friends" {
		t.Errorf("out = %+v, want one label \"Friends\"", out)
	}
}

func TestListCharacterContactLabelsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCharacterContactLabels
	t.Cleanup(func() { listCharacterContactLabels = orig })
	listCharacterContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]ContactLabel, error) {
		t.Fatal("listCharacterContactLabels should not be called for an unknown character")
		return nil, nil
	}

	handler := listCharacterContactLabelsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCharacterContactLabelsInput{CharacterID: 999})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContactsHandler_ReturnsContacts(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContacts
	t.Cleanup(func() { listCorporationContacts = orig })
	listCorporationContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Contact, error) {
		return []Contact{{ContactID: 998879}}, nil
	}

	handler := listCorporationContactsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationContactsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.CorporationID != 98000001 || len(out.Contacts) != 1 {
		t.Errorf("out = %+v, want CorporationID=98000001 with one contact", out)
	}
}

func TestListCorporationContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationContacts
	t.Cleanup(func() { listCorporationContacts = orig })
	listCorporationContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Contact, error) {
		t.Fatal("listCorporationContacts should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationContactsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationContactsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListCorporationContactLabelsHandler_ReturnsLabels(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listCorporationContactLabels
	t.Cleanup(func() { listCorporationContactLabels = orig })
	listCorporationContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]ContactLabel, error) {
		return []ContactLabel{{LabelID: 2, LabelName: "Corp VIP"}}, nil
	}

	handler := listCorporationContactLabelsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListCorporationContactLabelsInput{CharacterID: 7, CorporationID: 98000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Labels) != 1 || out.Labels[0].LabelName != "Corp VIP" {
		t.Errorf("out = %+v, want one label \"Corp VIP\"", out)
	}
}

func TestListCorporationContactLabelsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listCorporationContactLabels
	t.Cleanup(func() { listCorporationContactLabels = orig })
	listCorporationContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]ContactLabel, error) {
		t.Fatal("listCorporationContactLabels should not be called for an unknown character")
		return nil, nil
	}

	handler := listCorporationContactLabelsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListCorporationContactLabelsInput{CharacterID: 999, CorporationID: 98000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListAllianceContactsHandler_ReturnsContacts(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listAllianceContacts
	t.Cleanup(func() { listAllianceContacts = orig })
	listAllianceContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, allianceID int64) ([]Contact, error) {
		return []Contact{{ContactID: 998880, Standing: -5.0}}, nil
	}

	handler := listAllianceContactsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListAllianceContactsInput{CharacterID: 7, AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if out.AllianceID != 99000001 || len(out.Contacts) != 1 {
		t.Errorf("out = %+v, want AllianceID=99000001 with one contact", out)
	}
}

func TestListAllianceContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listAllianceContacts
	t.Cleanup(func() { listAllianceContacts = orig })
	listAllianceContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, allianceID int64) ([]Contact, error) {
		t.Fatal("listAllianceContacts should not be called for an unknown character")
		return nil, nil
	}

	handler := listAllianceContactsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListAllianceContactsInput{CharacterID: 999, AllianceID: 99000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestListAllianceContactLabelsHandler_ReturnsLabels(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := listAllianceContactLabels
	t.Cleanup(func() { listAllianceContactLabels = orig })
	listAllianceContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string, allianceID int64) ([]ContactLabel, error) {
		return []ContactLabel{{LabelID: 3, LabelName: "Alliance Blues"}}, nil
	}

	handler := listAllianceContactLabelsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, ListAllianceContactLabelsInput{CharacterID: 7, AllianceID: 99000001})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.Labels) != 1 || out.Labels[0].LabelName != "Alliance Blues" {
		t.Errorf("out = %+v, want one label \"Alliance Blues\"", out)
	}
}

func TestListAllianceContactLabelsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := listAllianceContactLabels
	t.Cleanup(func() { listAllianceContactLabels = orig })
	listAllianceContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string, allianceID int64) ([]ContactLabel, error) {
		t.Fatal("listAllianceContactLabels should not be called for an unknown character")
		return nil, nil
	}

	handler := listAllianceContactLabelsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, ListAllianceContactLabelsInput{CharacterID: 999, AllianceID: 99000001})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestAddCharacterContactsHandler_ReturnsAddedIDs(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := addCharacterContacts
	t.Cleanup(func() { addCharacterContacts = orig })
	addCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error) {
		return []int64{998878}, nil
	}

	handler := addCharacterContactsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, AddCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{998878}, Standing: 5.0})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.AddedContactIDs) != 1 || out.AddedContactIDs[0] != 998878 {
		t.Errorf("out = %+v, want AddedContactIDs=[998878]", out)
	}
}

func TestAddCharacterContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := addCharacterContacts
	t.Cleanup(func() { addCharacterContacts = orig })
	addCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error) {
		t.Fatal("addCharacterContacts should not be called for an unknown character")
		return nil, nil
	}

	handler := addCharacterContactsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, AddCharacterContactsInput{CharacterID: 999, ContactIDs: []int64{998878}, Standing: 5.0})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestUpdateCharacterContactsHandler_Succeeds(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := updateCharacterContacts
	t.Cleanup(func() { updateCharacterContacts = orig })
	var gotStanding float64
	updateCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error {
		gotStanding = standing
		return nil
	}

	handler := updateCharacterContactsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, UpdateCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{998878}, Standing: 3.0})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if gotStanding != 3.0 || len(out.ContactIDs) != 1 {
		t.Errorf("gotStanding = %v, out = %+v, want Standing=3.0 with one contact ID echoed", gotStanding, out)
	}
}

func TestUpdateCharacterContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := updateCharacterContacts
	t.Cleanup(func() { updateCharacterContacts = orig })
	updateCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error {
		t.Fatal("updateCharacterContacts should not be called for an unknown character")
		return nil
	}

	handler := updateCharacterContactsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, UpdateCharacterContactsInput{CharacterID: 999, ContactIDs: []int64{998878}, Standing: 3.0})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}

func TestRemoveCharacterContactsHandler_ReturnsRemovedIDs(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")
	if err := store.Save(config.Config{Characters: []config.Character{{ID: 7, Name: "Test Pilot"}}}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	orig := removeCharacterContacts
	t.Cleanup(func() { removeCharacterContacts = orig })
	removeCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64) error {
		return nil
	}

	handler := removeCharacterContactsHandler("v1", "client-id", store)
	_, out, err := handler(context.Background(), nil, RemoveCharacterContactsInput{CharacterID: 7, ContactIDs: []int64{998878}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(out.RemovedContactIDs) != 1 || out.RemovedContactIDs[0] != 998878 {
		t.Errorf("out = %+v, want RemovedContactIDs=[998878]", out)
	}
}

func TestRemoveCharacterContactsHandler_UnknownCharacter_ReturnsErrCharacterNotFound(t *testing.T) {
	keyring.MockInit()
	store := config.NewStoreAt(t.TempDir() + "/config.json")

	orig := removeCharacterContacts
	t.Cleanup(func() { removeCharacterContacts = orig })
	removeCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64) error {
		t.Fatal("removeCharacterContacts should not be called for an unknown character")
		return nil
	}

	handler := removeCharacterContactsHandler("v1", "client-id", store)
	_, _, err := handler(context.Background(), nil, RemoveCharacterContactsInput{CharacterID: 999, ContactIDs: []int64{998878}})
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Errorf("err = %v, want wrapping ErrCharacterNotFound", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./mcp/... -run 'TestListCharacterContactsHandler|TestListCharacterContactLabelsHandler|TestListCorporationContactsHandler|TestListCorporationContactLabelsHandler|TestListAllianceContactsHandler|TestListAllianceContactLabelsHandler|TestAddCharacterContactsHandler|TestUpdateCharacterContactsHandler|TestRemoveCharacterContactsHandler' -v`

Expected: FAIL to compile.

- [ ] **Step 3: Implement `mcp/contacts.go`**

**Do not declare `characterExists` or `ErrCharacterNotFound` in this file — both already exist
package-wide** (`characterExists` in `mcp/character.go`, `ErrCharacterNotFound` in `mcp/remove_character.go`).

```go
package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/esi"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// Contact is one contact entry, shared by character/corporation/alliance listings.
type Contact struct {
	ContactID   int64   `json:"contactId"`
	ContactType string  `json:"contactType"`
	Standing    float64 `json:"standing"`
	IsBlocked   *bool   `json:"isBlocked,omitempty"`
	IsWatched   *bool   `json:"isWatched,omitempty"`
	LabelIDs    []int64 `json:"labelIds,omitempty"`
}

// ContactLabel is one contact label, shared by character/corporation/alliance listings.
type ContactLabel struct {
	LabelID   int64  `json:"labelId"`
	LabelName string `json:"labelName"`
}

// --- list_character_contacts ---

var listCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]Contact, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	contacts, err := client.CharacterContacts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Contact, len(contacts))
	for i, c := range contacts {
		out[i] = Contact{ContactID: c.ContactId, ContactType: c.ContactType, Standing: c.Standing, IsBlocked: c.IsBlocked, IsWatched: c.IsWatched, LabelIDs: c.LabelIds}
	}
	return out, nil
}

type ListCharacterContactsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterContactsOutput struct {
	CharacterID int32     `json:"characterId"`
	Contacts    []Contact `json:"contacts"`
}

func listCharacterContactsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterContactsInput, ListCharacterContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContactsInput) (*mcpsdk.CallToolResult, ListCharacterContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContactsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContactsOutput{}, fmt.Errorf("list character contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		contacts, err := listCharacterContacts(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterContactsOutput{}, fmt.Errorf("list character contacts: %w", err)
		}
		return nil, ListCharacterContactsOutput{CharacterID: in.CharacterID, Contacts: contacts}, nil
	}
}

// --- list_character_contact_labels ---

var listCharacterContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string) ([]ContactLabel, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	labels, err := client.CharacterContactLabels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ContactLabel, len(labels))
	for i, l := range labels {
		out[i] = ContactLabel{LabelID: l.LabelId, LabelName: l.LabelName}
	}
	return out, nil
}

type ListCharacterContactLabelsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
}
type ListCharacterContactLabelsOutput struct {
	CharacterID int32          `json:"characterId"`
	Labels      []ContactLabel `json:"labels"`
}

func listCharacterContactLabelsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCharacterContactLabelsInput, ListCharacterContactLabelsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCharacterContactLabelsInput) (*mcpsdk.CallToolResult, ListCharacterContactLabelsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCharacterContactLabelsOutput{}, err
		}
		if !found {
			return nil, ListCharacterContactLabelsOutput{}, fmt.Errorf("list character contact labels for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		labels, err := listCharacterContactLabels(ctx, clientID, in.CharacterID, sso.UserAgent(version))
		if err != nil {
			return nil, ListCharacterContactLabelsOutput{}, fmt.Errorf("list character contact labels: %w", err)
		}
		return nil, ListCharacterContactLabelsOutput{CharacterID: in.CharacterID, Labels: labels}, nil
	}
}

// --- list_corporation_contacts ---

var listCorporationContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]Contact, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	contacts, err := client.CorporationContacts(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]Contact, len(contacts))
	for i, c := range contacts {
		out[i] = Contact{ContactID: c.ContactId, ContactType: c.ContactType, Standing: c.Standing, IsWatched: c.IsWatched, LabelIDs: c.LabelIds}
	}
	return out, nil
}

type ListCorporationContactsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationContactsOutput struct {
	CharacterID   int32     `json:"characterId"`
	CorporationID int64     `json:"corporationId"`
	Contacts      []Contact `json:"contacts"`
}

func listCorporationContactsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationContactsInput, ListCorporationContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContactsInput) (*mcpsdk.CallToolResult, ListCorporationContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContactsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContactsOutput{}, fmt.Errorf("list corporation contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		contacts, err := listCorporationContacts(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationContactsOutput{}, fmt.Errorf("list corporation contacts: %w", err)
		}
		return nil, ListCorporationContactsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Contacts: contacts}, nil
	}
}

// --- list_corporation_contact_labels ---

var listCorporationContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string, corporationID int64) ([]ContactLabel, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	labels, err := client.CorporationContactLabels(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	out := make([]ContactLabel, len(labels))
	for i, l := range labels {
		out[i] = ContactLabel{LabelID: l.LabelId, LabelName: l.LabelName}
	}
	return out, nil
}

type ListCorporationContactLabelsInput struct {
	CharacterID   int32 `json:"characterId" jsonschema:"the character ID to check"`
	CorporationID int64 `json:"corporationId" jsonschema:"the corporation ID to look up"`
}
type ListCorporationContactLabelsOutput struct {
	CharacterID   int32          `json:"characterId"`
	CorporationID int64          `json:"corporationId"`
	Labels        []ContactLabel `json:"labels"`
}

func listCorporationContactLabelsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListCorporationContactLabelsInput, ListCorporationContactLabelsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListCorporationContactLabelsInput) (*mcpsdk.CallToolResult, ListCorporationContactLabelsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListCorporationContactLabelsOutput{}, err
		}
		if !found {
			return nil, ListCorporationContactLabelsOutput{}, fmt.Errorf("list corporation contact labels for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		labels, err := listCorporationContactLabels(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.CorporationID)
		if err != nil {
			return nil, ListCorporationContactLabelsOutput{}, fmt.Errorf("list corporation contact labels: %w", err)
		}
		return nil, ListCorporationContactLabelsOutput{CharacterID: in.CharacterID, CorporationID: in.CorporationID, Labels: labels}, nil
	}
}

// --- list_alliance_contacts ---

var listAllianceContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, allianceID int64) ([]Contact, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	contacts, err := client.AllianceContacts(ctx, allianceID)
	if err != nil {
		return nil, err
	}
	out := make([]Contact, len(contacts))
	for i, c := range contacts {
		out[i] = Contact{ContactID: c.ContactId, ContactType: c.ContactType, Standing: c.Standing, LabelIDs: c.LabelIds}
	}
	return out, nil
}

type ListAllianceContactsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	AllianceID  int64 `json:"allianceId" jsonschema:"the alliance ID to look up"`
}
type ListAllianceContactsOutput struct {
	CharacterID int32     `json:"characterId"`
	AllianceID  int64     `json:"allianceId"`
	Contacts    []Contact `json:"contacts"`
}

func listAllianceContactsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListAllianceContactsInput, ListAllianceContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAllianceContactsInput) (*mcpsdk.CallToolResult, ListAllianceContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListAllianceContactsOutput{}, err
		}
		if !found {
			return nil, ListAllianceContactsOutput{}, fmt.Errorf("list alliance contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		contacts, err := listAllianceContacts(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.AllianceID)
		if err != nil {
			return nil, ListAllianceContactsOutput{}, fmt.Errorf("list alliance contacts: %w", err)
		}
		return nil, ListAllianceContactsOutput{CharacterID: in.CharacterID, AllianceID: in.AllianceID, Contacts: contacts}, nil
	}
}

// --- list_alliance_contact_labels ---

var listAllianceContactLabels = func(ctx context.Context, clientID string, characterID int32, userAgent string, allianceID int64) ([]ContactLabel, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	labels, err := client.AllianceContactLabels(ctx, allianceID)
	if err != nil {
		return nil, err
	}
	out := make([]ContactLabel, len(labels))
	for i, l := range labels {
		out[i] = ContactLabel{LabelID: l.LabelId, LabelName: l.LabelName}
	}
	return out, nil
}

type ListAllianceContactLabelsInput struct {
	CharacterID int32 `json:"characterId" jsonschema:"the character ID to check"`
	AllianceID  int64 `json:"allianceId" jsonschema:"the alliance ID to look up"`
}
type ListAllianceContactLabelsOutput struct {
	CharacterID int32          `json:"characterId"`
	AllianceID  int64          `json:"allianceId"`
	Labels      []ContactLabel `json:"labels"`
}

func listAllianceContactLabelsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[ListAllianceContactLabelsInput, ListAllianceContactLabelsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in ListAllianceContactLabelsInput) (*mcpsdk.CallToolResult, ListAllianceContactLabelsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, ListAllianceContactLabelsOutput{}, err
		}
		if !found {
			return nil, ListAllianceContactLabelsOutput{}, fmt.Errorf("list alliance contact labels for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		labels, err := listAllianceContactLabels(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.AllianceID)
		if err != nil {
			return nil, ListAllianceContactLabelsOutput{}, fmt.Errorf("list alliance contact labels: %w", err)
		}
		return nil, ListAllianceContactLabelsOutput{CharacterID: in.CharacterID, AllianceID: in.AllianceID, Labels: labels}, nil
	}
}

// --- add_character_contacts (write) ---

var addCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64, standing float64, labelIDs []int64, watched bool) ([]int64, error) {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return nil, fmt.Errorf("build ESI client: %w", err)
	}
	return client.AddCharacterContacts(ctx, contactIDs, standing, labelIDs, watched)
}

type AddCharacterContactsInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	ContactIDs  []int64 `json:"contactIds" jsonschema:"character/corporation/alliance IDs to add as contacts"`
	Standing    float64 `json:"standing" jsonschema:"standing to assign, from -10.0 to 10.0"`
	LabelIDs    []int64 `json:"labelIds,omitempty" jsonschema:"label IDs to apply, from list_character_contact_labels"`
	Watched     bool    `json:"watched,omitempty" jsonschema:"whether to watch these contacts"`
}
type AddCharacterContactsOutput struct {
	CharacterID     int32   `json:"characterId"`
	AddedContactIDs []int64 `json:"addedContactIds"`
}

func addCharacterContactsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[AddCharacterContactsInput, AddCharacterContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in AddCharacterContactsInput) (*mcpsdk.CallToolResult, AddCharacterContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, AddCharacterContactsOutput{}, err
		}
		if !found {
			return nil, AddCharacterContactsOutput{}, fmt.Errorf("add character contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		added, err := addCharacterContacts(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ContactIDs, in.Standing, in.LabelIDs, in.Watched)
		if err != nil {
			return nil, AddCharacterContactsOutput{}, fmt.Errorf("add character contacts: %w", err)
		}
		return nil, AddCharacterContactsOutput{CharacterID: in.CharacterID, AddedContactIDs: added}, nil
	}
}

// --- update_character_contacts (write) ---

var updateCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64, standing float64, labelIDs []int64, watched bool) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.UpdateCharacterContacts(ctx, contactIDs, standing, labelIDs, watched)
}

type UpdateCharacterContactsInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	ContactIDs  []int64 `json:"contactIds" jsonschema:"contact IDs to update"`
	Standing    float64 `json:"standing" jsonschema:"new standing to assign, from -10.0 to 10.0"`
	LabelIDs    []int64 `json:"labelIds,omitempty" jsonschema:"label IDs to apply"`
	Watched     bool    `json:"watched,omitempty" jsonschema:"whether to watch these contacts"`
}
type UpdateCharacterContactsOutput struct {
	CharacterID int32   `json:"characterId"`
	ContactIDs  []int64 `json:"contactIds"`
}

func updateCharacterContactsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[UpdateCharacterContactsInput, UpdateCharacterContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in UpdateCharacterContactsInput) (*mcpsdk.CallToolResult, UpdateCharacterContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, UpdateCharacterContactsOutput{}, err
		}
		if !found {
			return nil, UpdateCharacterContactsOutput{}, fmt.Errorf("update character contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := updateCharacterContacts(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ContactIDs, in.Standing, in.LabelIDs, in.Watched); err != nil {
			return nil, UpdateCharacterContactsOutput{}, fmt.Errorf("update character contacts: %w", err)
		}
		return nil, UpdateCharacterContactsOutput{CharacterID: in.CharacterID, ContactIDs: in.ContactIDs}, nil
	}
}

// --- remove_character_contacts (write) ---

var removeCharacterContacts = func(ctx context.Context, clientID string, characterID int32, userAgent string, contactIDs []int64) error {
	client, err := esi.NewClient(ctx, clientID, characterID, userAgent)
	if err != nil {
		return fmt.Errorf("build ESI client: %w", err)
	}
	return client.RemoveCharacterContacts(ctx, contactIDs)
}

type RemoveCharacterContactsInput struct {
	CharacterID int32   `json:"characterId" jsonschema:"the character ID to check"`
	ContactIDs  []int64 `json:"contactIds" jsonschema:"contact IDs to remove"`
}
type RemoveCharacterContactsOutput struct {
	CharacterID       int32   `json:"characterId"`
	RemovedContactIDs []int64 `json:"removedContactIds"`
}

func removeCharacterContactsHandler(version, clientID string, store *config.Store) mcpsdk.ToolHandlerFor[RemoveCharacterContactsInput, RemoveCharacterContactsOutput] {
	return func(ctx context.Context, _ *mcpsdk.CallToolRequest, in RemoveCharacterContactsInput) (*mcpsdk.CallToolResult, RemoveCharacterContactsOutput, error) {
		found, err := characterExists(store, in.CharacterID)
		if err != nil {
			return nil, RemoveCharacterContactsOutput{}, err
		}
		if !found {
			return nil, RemoveCharacterContactsOutput{}, fmt.Errorf("remove character contacts for character %d: %w", in.CharacterID, ErrCharacterNotFound)
		}
		if err := removeCharacterContacts(ctx, clientID, in.CharacterID, sso.UserAgent(version), in.ContactIDs); err != nil {
			return nil, RemoveCharacterContactsOutput{}, fmt.Errorf("remove character contacts: %w", err)
		}
		return nil, RemoveCharacterContactsOutput{CharacterID: in.CharacterID, RemovedContactIDs: in.ContactIDs}, nil
	}
}
```

- [ ] **Step 4: Register the nine tools in `mcp/server.go`**

Add 9 new `mcpsdk.AddTool` calls right after `list_character_implants`'s (the last tool from the
prior batch), updating the `NewServer` doc comment to mention all 9. Use these exact `Name`/
`Description`/handler pairs:

```go
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_contacts",
		Description: "List a logged-in EVE Online character's contacts.",
	}, listCharacterContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_character_contact_labels",
		Description: "List a logged-in EVE Online character's contact labels.",
	}, listCharacterContactLabelsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_contacts",
		Description: "List a corporation's contacts, using a logged-in character's roles to authorize.",
	}, listCorporationContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_corporation_contact_labels",
		Description: "List a corporation's contact labels, using a logged-in character's roles to authorize.",
	}, listCorporationContactLabelsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_alliance_contacts",
		Description: "List an alliance's contacts, using a logged-in character's roles to authorize.",
	}, listAllianceContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_alliance_contact_labels",
		Description: "List an alliance's contact labels, using a logged-in character's roles to authorize.",
	}, listAllianceContactLabelsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "add_character_contacts",
		Description: "Add contacts to a character's contact list with a given standing — this creates new contacts on EVE's server.",
	}, addCharacterContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "update_character_contacts",
		Description: "Update the standing/label/watched status of existing contacts — this changes state on EVE's server.",
	}, updateCharacterContactsHandler(version, clientID, store))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "remove_character_contacts",
		Description: "Remove contacts from a character's contact list — this deletes contacts on EVE's server.",
	}, removeCharacterContactsHandler(version, clientID, store))
```

- [ ] **Step 5: Update `mcp/server_test.go`'s expected-tools list**

Append these 9 names (39 → 48):

```go
		"list_character_contacts", "list_character_contact_labels",
		"list_corporation_contacts", "list_corporation_contact_labels",
		"list_alliance_contacts", "list_alliance_contact_labels",
		"add_character_contacts", "update_character_contacts", "remove_character_contacts",
```

- [ ] **Step 6: Update docs**

- `CLAUDE.md`/`README.md`: bump tool count from "thirty-nine" to "forty-eight", append the 9 new
  tool names to the enumerated list, add "and Contacts" to README's covered-tags sentence.
- `docs/reference/esi-coverage-roadmap.md`: change the Contacts row from "Planned" to "This plan".

- [ ] **Step 7: Run all tests to verify they pass**

Run: `go build ./... && go vet ./... && gofmt -l . && go test ./... -race -v`

Expected: all 18 new tests PASS, `TestServer_AllToolsRegistered` PASSes with all 48 names,
`gofmt -l .` prints nothing, every pre-existing test passes.

- [ ] **Step 8: Commit**

```bash
git add mcp/contacts.go mcp/contacts_internal_test.go mcp/server.go mcp/server_test.go CLAUDE.md README.md docs/reference/esi-coverage-roadmap.md
git commit -m "feat: add contacts MCP tools (9 tools: 6 reads + 3 writes), update docs and roadmap"
```
