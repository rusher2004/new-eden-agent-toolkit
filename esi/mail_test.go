package esi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
	esiapi "github.com/fnt-eve/goesi-openapi/esi"

	"github.com/rusher2004/new-eden-agent-toolkit/esi"
)

func newTestClient(srv *httptest.Server) *esi.Client {
	return esi.ExportedNewClientWithOptions(srv.Client(), 998877, goesi.ClientOptions{
		BaseURL:   srv.URL,
		UserAgent: "test-agent",
	})
}

// --- CharacterMailHeaders ---

func TestCharacterMailHeaders_ParsesResponse(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"from":90000001,"is_read":true,"labels":[1,2],"mail_id":123,"recipients":[{"recipient_id":90000002,"recipient_type":"character"}],"subject":"Hi","timestamp":"2026-01-01T00:00:00Z"}]`))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	headers, err := client.CharacterMailHeaders(context.Background(), []int64{1, 2}, 456)
	if err != nil {
		t.Fatalf("CharacterMailHeaders returned error: %v", err)
	}
	if len(headers) != 1 {
		t.Fatalf("headers = %+v, want one entry", headers)
	}
	h := headers[0]
	if h.From == nil || *h.From != 90000001 || h.MailId == nil || *h.MailId != 123 || h.Subject == nil || *h.Subject != "Hi" {
		t.Errorf("headers[0] = %+v, want From=90000001 MailId=123 Subject=\"Hi\"", h)
	}
	if h.IsRead == nil || !*h.IsRead {
		t.Errorf("headers[0].IsRead = %v, want true", h.IsRead)
	}
	wantTimestamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if h.Timestamp == nil || !h.Timestamp.Equal(wantTimestamp) {
		t.Errorf("headers[0].Timestamp = %v, want %v", h.Timestamp, wantTimestamp)
	}
	if len(h.Labels) != 2 || h.Labels[0] != 1 || h.Labels[1] != 2 {
		t.Errorf("headers[0].Labels = %v, want [1 2]", h.Labels)
	}
	if len(h.Recipients) != 1 || h.Recipients[0].RecipientId != 90000002 || h.Recipients[0].RecipientType != "character" {
		t.Errorf("headers[0].Recipients = %+v, want one entry {RecipientId:90000002 RecipientType:\"character\"}", h.Recipients)
	}
	if gotQuery == "" {
		t.Fatal("request query string was empty, want labels and last_mail_id params")
	}
}

func TestCharacterMailHeaders_OmitsQueryParamsWhenUnset(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if _, err := client.CharacterMailHeaders(context.Background(), nil, 0); err != nil {
		t.Fatalf("CharacterMailHeaders returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query string = %q, want empty (no labels/last_mail_id filters set)", gotQuery)
	}
}

func TestCharacterMailHeaders_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if _, err := client.CharacterMailHeaders(context.Background(), nil, 0); err == nil {
		t.Fatal("CharacterMailHeaders returned nil error, want an error for HTTP 404")
	}
}

// --- CharacterMailLabels ---

func TestCharacterMailLabels_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"labels":[{"color":"#ffffff","label_id":1,"name":"Inbox","unread_count":3}],"total_unread_count":3}`))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	labels, err := client.CharacterMailLabels(context.Background())
	if err != nil {
		t.Fatalf("CharacterMailLabels returned error: %v", err)
	}
	if labels.TotalUnreadCount == nil || *labels.TotalUnreadCount != 3 {
		t.Errorf("TotalUnreadCount = %v, want 3", labels.TotalUnreadCount)
	}
	if len(labels.Labels) != 1 {
		t.Fatalf("Labels = %+v, want one entry", labels.Labels)
	}
	l := labels.Labels[0]
	if l.LabelId == nil || *l.LabelId != 1 || l.Name == nil || *l.Name != "Inbox" || l.Color == nil || *l.Color != "#ffffff" || l.UnreadCount == nil || *l.UnreadCount != 3 {
		t.Errorf("Labels[0] = %+v, want LabelId=1 Name=\"Inbox\" Color=\"#ffffff\" UnreadCount=3", l)
	}
}

func TestCharacterMailLabels_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if _, err := client.CharacterMailLabels(context.Background()); err == nil {
		t.Fatal("CharacterMailLabels returned nil error, want an error for HTTP 404")
	}
}

// --- CharacterMailLists ---

func TestCharacterMailLists_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"mailing_list_id":1,"name":"Test List"}]`))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	lists, err := client.CharacterMailLists(context.Background())
	if err != nil {
		t.Fatalf("CharacterMailLists returned error: %v", err)
	}
	if len(lists) != 1 || lists[0].MailingListId != 1 || lists[0].Name != "Test List" {
		t.Errorf("lists = %+v, want one entry {MailingListId:1 Name:\"Test List\"}", lists)
	}
}

func TestCharacterMailLists_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if _, err := client.CharacterMailLists(context.Background()); err == nil {
		t.Fatal("CharacterMailLists returned nil error, want an error for HTTP 404")
	}
}

// --- CharacterMail ---

func TestCharacterMail_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"body":"Hello there","from":90000001,"labels":[1],"read":true,"recipients":[{"recipient_id":90000002,"recipient_type":"character"}],"subject":"Hi","timestamp":"2026-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	mail, err := client.CharacterMail(context.Background(), 123)
	if err != nil {
		t.Fatalf("CharacterMail returned error: %v", err)
	}
	if mail.Body == nil || *mail.Body != "Hello there" || mail.From == nil || *mail.From != 90000001 || mail.Subject == nil || *mail.Subject != "Hi" {
		t.Errorf("mail = %+v, want Body=\"Hello there\" From=90000001 Subject=\"Hi\"", mail)
	}
	if mail.Read == nil || !*mail.Read {
		t.Errorf("mail.Read = %v, want true", mail.Read)
	}
	wantTimestamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if mail.Timestamp == nil || !mail.Timestamp.Equal(wantTimestamp) {
		t.Errorf("mail.Timestamp = %v, want %v", mail.Timestamp, wantTimestamp)
	}
	if len(mail.Labels) != 1 || mail.Labels[0] != 1 {
		t.Errorf("mail.Labels = %v, want [1]", mail.Labels)
	}
	if len(mail.Recipients) != 1 || mail.Recipients[0].RecipientId != 90000002 || mail.Recipients[0].RecipientType != "character" {
		t.Errorf("mail.Recipients = %+v, want one entry {RecipientId:90000002 RecipientType:\"character\"}", mail.Recipients)
	}
}

func TestCharacterMail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if _, err := client.CharacterMail(context.Background(), 123); err == nil {
		t.Fatal("CharacterMail returned nil error, want an error for HTTP 404")
	}
}

// --- SendCharacterMail ---

func TestSendCharacterMail_SendsCorrectRequestBodyAndParsesResponse(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`789`))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	recipients := []esiapi.PostCharactersCharacterIdMailRequestRecipientsInner{
		{RecipientId: 90000002, RecipientType: "character"},
	}
	mailID, err := client.SendCharacterMail(context.Background(), "Hi", "Hello there", recipients, nil)
	if err != nil {
		t.Fatalf("SendCharacterMail returned error: %v", err)
	}

	var got struct {
		Body       string `json:"body"`
		Subject    string `json:"subject"`
		Recipients []struct {
			RecipientId   int64  `json:"recipient_id"`
			RecipientType string `json:"recipient_type"`
		} `json:"recipients"`
	}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", gotBody, err)
	}
	if got.Body != "Hello there" || got.Subject != "Hi" {
		t.Errorf("request body = %+v, want Body=\"Hello there\" Subject=\"Hi\"", got)
	}
	if len(got.Recipients) != 1 || got.Recipients[0].RecipientId != 90000002 || got.Recipients[0].RecipientType != "character" {
		t.Errorf("request body Recipients = %+v, want one entry {RecipientId:90000002 RecipientType:\"character\"}", got.Recipients)
	}
	if mailID != 789 {
		t.Errorf("mailID = %d, want 789", mailID)
	}
}

func TestSendCharacterMail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	recipients := []esiapi.PostCharactersCharacterIdMailRequestRecipientsInner{
		{RecipientId: 90000002, RecipientType: "character"},
	}
	if _, err := client.SendCharacterMail(context.Background(), "Hi", "Hello there", recipients, nil); err == nil {
		t.Fatal("SendCharacterMail returned nil error, want an error for HTTP 400")
	}
}

// --- CreateCharacterMailLabel ---

func TestCreateCharacterMailLabel_SendsCorrectRequestBodyAndParsesResponse(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`42`))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	color := "#ffffff"
	labelID, err := client.CreateCharacterMailLabel(context.Background(), "Important", &color)
	if err != nil {
		t.Fatalf("CreateCharacterMailLabel returned error: %v", err)
	}

	var got struct {
		Color string `json:"color"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", gotBody, err)
	}
	if got.Name != "Important" || got.Color != "#ffffff" {
		t.Errorf("request body = %+v, want Name=\"Important\" Color=\"#ffffff\"", got)
	}
	if labelID != 42 {
		t.Errorf("labelID = %d, want 42", labelID)
	}
}

func TestCreateCharacterMailLabel_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if _, err := client.CreateCharacterMailLabel(context.Background(), "Important", nil); err == nil {
		t.Fatal("CreateCharacterMailLabel returned nil error, want an error for HTTP 400")
	}
}

// --- UpdateCharacterMail ---

func TestUpdateCharacterMail_SendsCorrectRequestBodyAndHandlesNoContentResponse(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	read := true
	if err := client.UpdateCharacterMail(context.Background(), 123, []int64{1, 2}, &read); err != nil {
		t.Fatalf("UpdateCharacterMail returned error: %v", err)
	}

	var got struct {
		Labels []int64 `json:"labels"`
		Read   bool    `json:"read"`
	}
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("failed to unmarshal request body %q: %v", gotBody, err)
	}
	if len(got.Labels) != 2 || got.Labels[0] != 1 || got.Labels[1] != 2 || !got.Read {
		t.Errorf("request body = %+v, want Labels=[1 2] Read=true", got)
	}
}

func TestUpdateCharacterMail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if err := client.UpdateCharacterMail(context.Background(), 123, nil, nil); err == nil {
		t.Fatal("UpdateCharacterMail returned nil error, want an error for HTTP 404")
	}
}

// --- DeleteCharacterMail ---

func TestDeleteCharacterMail_SendsNoBodyAndHandlesNoContentResponse(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if err := client.DeleteCharacterMail(context.Background(), 123); err != nil {
		t.Fatalf("DeleteCharacterMail returned error: %v", err)
	}
	if len(gotBody) != 0 {
		t.Errorf("request body = %q, want empty", gotBody)
	}
}

func TestDeleteCharacterMail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if err := client.DeleteCharacterMail(context.Background(), 123); err == nil {
		t.Fatal("DeleteCharacterMail returned nil error, want an error for HTTP 404")
	}
}

// --- DeleteCharacterMailLabel ---

func TestDeleteCharacterMailLabel_SendsNoBodyAndHandlesNoContentResponse(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if err := client.DeleteCharacterMailLabel(context.Background(), 1); err != nil {
		t.Fatalf("DeleteCharacterMailLabel returned error: %v", err)
	}
	if len(gotBody) != 0 {
		t.Errorf("request body = %q, want empty", gotBody)
	}
}

func TestDeleteCharacterMailLabel_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	if err := client.DeleteCharacterMailLabel(context.Background(), 1); err == nil {
		t.Fatal("DeleteCharacterMailLabel returned nil error, want an error for HTTP 404")
	}
}
