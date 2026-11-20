package analytics_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rusher2004/new-eden-agent-toolkit/analytics"
)

func TestNew_AppKeys(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"eu", "A-EU-123", false},
		{"us", "A-US-123", false},
		{"empty", "", true},
		{"self-hosted", "A-SH-123", true},
		{"malformed", "EU-123", true},
		{"too many parts", "A-EU-1-2", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := analytics.New(tt.key, "1.0.0", "neat/1.0.0")
			if (err != nil) != tt.wantErr {
				t.Errorf("New(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestTrackStart_SendsOnlyExpectedFields(t *testing.T) {
	var gotHeader http.Header
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v0/events" {
			t.Errorf("request = %s %s, want POST /api/v0/events", r.Method, r.URL.Path)
		}
		gotHeader = r.Header.Clone()
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
	}))
	defer srv.Close()

	c := analytics.NewWithBaseURL("A-EU-123", srv.URL, "1.2.3", "neat/1.2.3 (test)")
	now := time.Date(2026, 11, 19, 12, 30, 45, 0, time.UTC)
	if err := c.TrackStart(context.Background(), "abc", now); err != nil {
		t.Fatalf("TrackStart: %v", err)
	}

	if got := gotHeader.Get("App-Key"); got != "A-EU-123" {
		t.Errorf("App-Key = %q, want A-EU-123", got)
	}
	if got := gotHeader.Get("User-Agent"); got != "neat/1.2.3 (test)" {
		t.Errorf("User-Agent = %q", got)
	}

	var events []map[string]any
	if err := json.Unmarshal(gotBody, &events); err != nil {
		t.Fatalf("body is not a JSON array: %v (%s)", err, gotBody)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	ev := events[0]
	if ev["eventName"] != analytics.StartEvent {
		t.Errorf("eventName = %v", ev["eventName"])
	}
	if ev["timestamp"] != "2026-11-19T12:30:45Z" {
		t.Errorf("timestamp = %v", ev["timestamp"])
	}
	sid, ok := ev["sessionId"].(string)
	if !ok || !strings.HasPrefix(sid, strconv.FormatInt(now.Unix(), 10)) || len(sid) != len(strconv.FormatInt(now.Unix(), 10))+8 {
		t.Errorf("sessionId = %v, want epoch seconds followed by 8 random digits", ev["sessionId"])
	}

	props, ok := ev["props"].(map[string]any)
	if !ok {
		t.Fatalf("props = %v", ev["props"])
	}
	wantProps := map[string]any{"monthlyId": "abc", "arch": runtime.GOARCH}
	if len(props) != len(wantProps) {
		t.Errorf("props = %v, want exactly %v", props, wantProps)
	}
	for k, v := range wantProps {
		if props[k] != v {
			t.Errorf("props[%q] = %v, want %v", k, props[k], v)
		}
	}

	sys, ok := ev["systemProps"].(map[string]any)
	if !ok {
		t.Fatalf("systemProps = %v", ev["systemProps"])
	}
	if sys["appVersion"] != "1.2.3" || sys["isDebug"] != false || sys["sdkVersion"] != "neat@1.2.3" {
		t.Errorf("systemProps = %v", sys)
	}
	for _, k := range []string{"locale", "osVersion", "deviceModel", "appBuildNumber"} {
		if sys[k] != "" {
			t.Errorf("systemProps[%q] = %v, want empty", k, sys[k])
		}
	}
	if sys["osName"] == "" {
		t.Error("systemProps.osName is empty")
	}
}

func TestTrackStart_DevBuildIsDebug(t *testing.T) {
	var isDebug any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var events []struct {
			SystemProps map[string]any `json:"systemProps"`
		}
		if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
			t.Errorf("decode: %v", err)
			return
		}
		isDebug = events[0].SystemProps["isDebug"]
	}))
	defer srv.Close()

	c := analytics.NewWithBaseURL("A-EU-123", srv.URL, "dev", "neat/dev")
	if err := c.TrackStart(context.Background(), "abc", time.Now()); err != nil {
		t.Fatalf("TrackStart: %v", err)
	}
	if isDebug != true {
		t.Errorf("isDebug = %v, want true for a dev build", isDebug)
	}
}

func TestTrackStart_ServerErrorIsReturned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad app key", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := analytics.NewWithBaseURL("A-EU-123", srv.URL, "1.0.0", "neat/1.0.0")
	err := c.TrackStart(context.Background(), "abc", time.Now())
	if err == nil || !strings.Contains(err.Error(), "bad app key") {
		t.Errorf("err = %v, want it to carry the server's message", err)
	}
}

func TestMonthlyID_StableWithinMonth(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")
	first, err := analytics.MonthlyID(path, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("MonthlyID: %v", err)
	}
	if len(first) != 32 {
		t.Errorf("id = %q, want 32 hex characters", first)
	}
	second, err := analytics.MonthlyID(path, time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC))
	if err != nil {
		t.Fatalf("MonthlyID: %v", err)
	}
	if first != second {
		t.Errorf("id changed within a month: %q then %q", first, second)
	}
}

func TestMonthlyID_RotatesEachMonth(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")
	nov, err := analytics.MonthlyID(path, time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("MonthlyID: %v", err)
	}
	dec, err := analytics.MonthlyID(path, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("MonthlyID: %v", err)
	}
	if nov == dec {
		t.Error("id did not change when the month changed")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), nov) {
		t.Error("the previous month's id is still on disk")
	}
}

func TestMonthlyID_CorruptFileIsReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	id, err := analytics.MonthlyID(path, time.Now())
	if err != nil {
		t.Fatalf("MonthlyID: %v", err)
	}
	if len(id) != 32 {
		t.Errorf("id = %q", id)
	}
}

func TestRemoveID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.json")
	if err := analytics.RemoveID(path); err != nil {
		t.Errorf("RemoveID on a missing file: %v", err)
	}
	if _, err := analytics.MonthlyID(path, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := analytics.RemoveID(path); err != nil {
		t.Fatalf("RemoveID: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file still exists after RemoveID: %v", err)
	}
}

type sentEvent struct {
	SessionID string            `json:"sessionId"`
	EventName string            `json:"eventName"`
	Props     map[string]string `json:"props"`
}

// recorder is a fake Aptabase that keeps every request's events.
type recorder struct {
	mu       sync.Mutex
	requests [][]sentEvent
}

func (r *recorder) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var events []sentEvent
		if err := json.NewDecoder(req.Body).Decode(&events); err != nil {
			t.Errorf("decode: %v", err)
		}
		r.mu.Lock()
		r.requests = append(r.requests, events)
		r.mu.Unlock()
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (r *recorder) events() []sentEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []sentEvent
	for _, req := range r.requests {
		all = append(all, req...)
	}
	return all
}

func newTracker(t *testing.T, url string) *analytics.ToolTracker {
	t.Helper()
	c := analytics.NewWithBaseURL("A-EU-123", url, "1.0.0", "neat/1.0.0")
	tr, err := c.NewToolTracker(time.Now)
	if err != nil {
		t.Fatalf("NewToolTracker: %v", err)
	}
	return tr
}

func TestToolTracker_FlushBatchesAndSendsOnlyToolAndOutcome(t *testing.T) {
	var rec recorder
	srv := rec.server(t)
	tr := newTracker(t, srv.URL)

	for i := 0; i < 30; i++ {
		tr.Record("plan_build", i%2 == 0)
	}
	if err := tr.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	if len(rec.requests) != 2 || len(rec.requests[0]) != 25 || len(rec.requests[1]) != 5 {
		t.Fatalf("batches = %d, want 25 then 5", len(rec.requests))
	}
	for _, ev := range rec.events() {
		if ev.EventName != analytics.ToolEvent {
			t.Errorf("eventName = %q", ev.EventName)
		}
		if len(ev.Props) != 2 || ev.Props["tool"] != "plan_build" {
			t.Errorf("props = %v, want exactly tool and outcome", ev.Props)
		}
		if o := ev.Props["outcome"]; o != "ok" && o != "error" {
			t.Errorf("outcome = %q", o)
		}
	}

	if err := tr.Flush(context.Background()); err != nil {
		t.Fatalf("second Flush: %v", err)
	}
	if len(rec.requests) != 2 {
		t.Errorf("an empty queue sent %d more requests", len(rec.requests)-2)
	}
}

func TestToolTracker_SessionIsSeparateFromStart(t *testing.T) {
	var rec recorder
	srv := rec.server(t)
	c := analytics.NewWithBaseURL("A-EU-123", srv.URL, "1.0.0", "neat/1.0.0")
	tr, err := c.NewToolTracker(time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.TrackStart(context.Background(), "monthly", time.Now()); err != nil {
		t.Fatal(err)
	}
	tr.Record("get_status", true)
	if err := tr.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}

	evs := rec.events()
	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2", len(evs))
	}
	if evs[0].SessionID == evs[1].SessionID {
		t.Error("tool events share app_started's session id")
	}
	if _, ok := evs[1].Props["monthlyId"]; ok {
		t.Error("tool event carries the monthly id")
	}
}

func TestToolTracker_FailedFlushDropsEvents(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	tr := newTracker(t, srv.URL)
	tr.Record("get_status", true)
	if err := tr.Flush(context.Background()); err == nil {
		t.Fatal("Flush returned nil for a failing server")
	}
	if err := tr.Flush(context.Background()); err != nil {
		t.Errorf("second Flush: %v, want nil since the failed batch was dropped", err)
	}
	if calls != 1 {
		t.Errorf("server got %d requests, want 1 (no retry)", calls)
	}
}

func TestToolTracker_NilIsNoop(t *testing.T) {
	var tr *analytics.ToolTracker
	tr.Record("get_status", true)
	if err := tr.Flush(context.Background()); err != nil {
		t.Errorf("nil Flush: %v", err)
	}
	tr.Run(context.Background(), time.Millisecond, func(error) {})
}

type echoIn struct {
	Fail bool `json:"fail"`
}

type echoOut struct {
	OK bool `json:"ok"`
}

func TestToolMiddleware_RecordsRegisteredToolsOnly(t *testing.T) {
	var rec recorder
	srv := rec.server(t)
	tr := newTracker(t, srv.URL)

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "1"}, nil)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "echo", Description: "echo"}, func(ctx context.Context, req *mcpsdk.CallToolRequest, in echoIn) (*mcpsdk.CallToolResult, echoOut, error) {
		if in.Fail {
			return nil, echoOut{}, errors.New("boom")
		}
		return nil, echoOut{OK: true}, nil
	})
	server.AddReceivingMiddleware(analytics.ToolMiddleware(tr))

	ctx := context.Background()
	serverT, clientT := mcpsdk.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "client", Version: "1"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	if _, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "echo", Arguments: map[string]any{"fail": false}}); err != nil {
		t.Fatalf("echo ok: %v", err)
	}
	if _, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "echo", Arguments: map[string]any{"fail": true}}); err != nil {
		t.Fatalf("echo fail: %v", err)
	}
	if _, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "my secret character name"}); err == nil {
		t.Fatal("unknown tool call succeeded")
	}
	if err := tr.Flush(ctx); err != nil {
		t.Fatal(err)
	}

	evs := rec.events()
	if len(evs) != 2 {
		t.Fatalf("got %d events %v, want 2 (the unknown tool must not be recorded)", len(evs), evs)
	}
	if evs[0].Props["tool"] != "echo" || evs[0].Props["outcome"] != "ok" {
		t.Errorf("first = %v", evs[0].Props)
	}
	if evs[1].Props["tool"] != "echo" || evs[1].Props["outcome"] != "error" {
		t.Errorf("second = %v", evs[1].Props)
	}
}
