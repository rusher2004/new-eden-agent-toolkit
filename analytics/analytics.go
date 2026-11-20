// Package analytics sends neat's anonymous usage statistics to Aptabase
// (https://aptabase.com), so the project can count monthly users and see
// which versions, platforms and tools are in use, without learning who anyone is.
//
// neat sends two events, and nothing else:
//
//   - "app_started", once per `neat mcp` launch, sent straight away: neat's
//     version, the OS and CPU architecture it was built for (runtime.GOOS
//     and runtime.GOARCH), a random session ID, and the random monthly ID
//     (see MonthlyID), which is what counts monthly users.
//   - "tool_called", once per MCP tool call, queued and sent in batches by a
//     ToolTracker: the tool's name and whether it succeeded, under a second,
//     separate random session ID and with no monthly ID, so feature usage can
//     be counted overall but never tied to one install's launch or month.
//
// Tool arguments, results, error text, durations, characters, accounts,
// paths, hostnames, locale and hardware identifiers are never sent. Aptabase sees the client's IP address in transit, derives a
// country from it and a hash that rotates daily, and doesn't store it.
//
// Sending is best-effort: a disabled config, a missing app key, or any
// network or server failure means nothing is sent, and none of them affects
// the rest of neat.
package analytics

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Event names neat sends.
const (
	StartEvent = "app_started"
	ToolEvent  = "tool_called"
)

// maxBatch is Aptabase's limit on events per request; maxQueue bounds the
// tool-event queue so an unreachable Aptabase can't grow it without limit.
const (
	maxBatch = 25
	maxQueue = 1000
)

// regionHosts maps an Aptabase app key's region segment (A-<region>-...) to
// its ingestion host. Self-hosted ("SH") keys aren't supported.
var regionHosts = map[string]string{
	"EU": "https://eu.aptabase.com",
	"US": "https://us.aptabase.com",
}

// Client sends events to one Aptabase app.
type Client struct {
	appKey    string
	baseURL   string
	version   string
	userAgent string
	http      *http.Client
}

// New returns a Client for appKey, or an error if the key is empty or its
// region isn't one Aptabase hosts. version is neat's build version;
// userAgent is sent on the request.
func New(appKey, version, userAgent string) (*Client, error) {
	parts := strings.Split(appKey, "-")
	if len(parts) != 3 || parts[0] != "A" {
		return nil, fmt.Errorf("aptabase app key %q is not of the form A-<region>-<id>", appKey)
	}
	host, ok := regionHosts[parts[1]]
	if !ok {
		return nil, fmt.Errorf("aptabase app key region %q is not supported", parts[1])
	}
	return NewWithBaseURL(appKey, host, version, userAgent), nil
}

// NewWithBaseURL returns a Client that posts to baseURL instead of the host
// the app key's region names, for tests.
func NewWithBaseURL(appKey, baseURL, version, userAgent string) *Client {
	return &Client{
		appKey:    appKey,
		baseURL:   strings.TrimRight(baseURL, "/"),
		version:   version,
		userAgent: userAgent,
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

// event is one entry in Aptabase's ingestion payload (POST /api/v0/events,
// a JSON array of these).
type event struct {
	Timestamp   string            `json:"timestamp"`
	SessionID   string            `json:"sessionId"`
	EventName   string            `json:"eventName"`
	SystemProps systemProps       `json:"systemProps"`
	Props       map[string]string `json:"props"`
}

// systemProps is Aptabase's fixed set of per-event platform fields. OS name
// comes from runtime.GOOS, the platform neat was compiled for, so it
// describes the release build, not the computer. OS version, locale, device
// model and build number are deliberately left empty: neat doesn't need
// them, so it doesn't send them.
type systemProps struct {
	IsDebug        bool   `json:"isDebug"`
	Locale         string `json:"locale"`
	OSName         string `json:"osName"`
	OSVersion      string `json:"osVersion"`
	AppVersion     string `json:"appVersion"`
	AppBuildNumber string `json:"appBuildNumber"`
	SDKVersion     string `json:"sdkVersion"`
	DeviceModel    string `json:"deviceModel"`
}

// TrackStart sends the "app_started" event with monthlyID, timestamped now.
func (c *Client) TrackStart(ctx context.Context, monthlyID string, now time.Time) error {
	sessionID, err := newSessionID(now)
	if err != nil {
		return fmt.Errorf("generate session id: %w", err)
	}
	ev := c.event(sessionID, StartEvent, now, map[string]string{
		"monthlyId": monthlyID,
		"arch":      runtime.GOARCH,
	})
	return c.send(ctx, []event{ev})
}

// event builds one event with neat's fixed system props.
func (c *Client) event(sessionID, name string, now time.Time, props map[string]string) event {
	return event{
		Timestamp: now.UTC().Format("2006-01-02T15:04:05Z"),
		SessionID: sessionID,
		EventName: name,
		SystemProps: systemProps{
			IsDebug:    c.version == "dev",
			OSName:     osName(),
			AppVersion: c.version,
			SDKVersion: "neat@" + c.version,
		},
		Props: props,
	}
}

// send posts events (at most maxBatch) in one request.
func (c *Client) send(ctx context.Context, events []event) error {
	body, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v0/events", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("App-Key", c.appKey)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send events: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		detail, err := io.ReadAll(io.LimitReader(resp.Body, 512))
		if err != nil {
			return fmt.Errorf("send events: aptabase returned %s (body unreadable: %w)", resp.Status, err)
		}
		return fmt.Errorf("send events: aptabase returned %s: %s", resp.Status, strings.TrimSpace(string(detail)))
	}
	return nil
}

// ToolTracker queues a "tool_called" event per tool call and sends them in
// batches. Its session ID is its own, never the one TrackStart used, and its
// events carry no monthly ID. A nil *ToolTracker is valid and records
// nothing, which is what analytics being off looks like to callers.
type ToolTracker struct {
	client    *Client
	sessionID string
	now       func() time.Time

	mu    sync.Mutex
	queue []event
}

// NewToolTracker returns a ToolTracker sending through c.
func (c *Client) NewToolTracker(now func() time.Time) (*ToolTracker, error) {
	sessionID, err := newSessionID(now())
	if err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}
	return &ToolTracker{client: c, sessionID: sessionID, now: now}, nil
}

// Record queues one tool call: the tool's name and whether it succeeded.
// When the queue is full the oldest event is dropped.
func (t *ToolTracker) Record(tool string, ok bool) {
	if t == nil {
		return
	}
	outcome := "ok"
	if !ok {
		outcome = "error"
	}
	ev := t.client.event(t.sessionID, ToolEvent, t.now(), map[string]string{
		"tool":    tool,
		"outcome": outcome,
	})
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.queue) >= maxQueue {
		t.queue = t.queue[1:]
	}
	t.queue = append(t.queue, ev)
}

// Flush sends every queued event, maxBatch per request. Events in a batch
// that fails to send are dropped, not retried: losing a few counts is better
// than holding usage data on the user's computer.
func (t *ToolTracker) Flush(ctx context.Context) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	pending := t.queue
	t.queue = nil
	t.mu.Unlock()

	var errs []error
	for len(pending) > 0 {
		n := min(len(pending), maxBatch)
		if err := t.client.send(ctx, pending[:n]); err != nil {
			errs = append(errs, err)
		}
		pending = pending[n:]
	}
	return errors.Join(errs...)
}

// ToolMiddleware returns MCP server middleware that records each tools/call
// on t. Only calls that reached a registered tool are recorded (a call the
// SDK rejects, such as an unknown tool name, returns a protocol error and
// is skipped), so the tool name sent is always one of neat's own, never
// arbitrary text from the client. A tool's own failure arrives as a result
// with IsError set and is recorded as "error".
func ToolMiddleware(t *ToolTracker) mcpsdk.Middleware {
	return func(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
		return func(ctx context.Context, method string, req mcpsdk.Request) (mcpsdk.Result, error) {
			res, err := next(ctx, method, req)
			if method != "tools/call" || err != nil {
				return res, err
			}
			params, ok := req.GetParams().(*mcpsdk.CallToolParamsRaw)
			if !ok {
				return res, err
			}
			succeeded := true
			if r, ok := res.(*mcpsdk.CallToolResult); ok && r.IsError {
				succeeded = false
			}
			t.Record(params.Name, succeeded)
			return res, err
		}
	}
}

// Run flushes every interval until ctx is done. It doesn't do the final
// flush; call Flush once more when the server stops.
func (t *ToolTracker) Run(ctx context.Context, interval time.Duration, onErr func(error)) {
	if t == nil {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := t.Flush(ctx); err != nil {
				onErr(err)
			}
		}
	}
}

// newSessionID builds a session ID the way Aptabase's own SDKs do: epoch
// seconds followed by eight random digits. It's random, so it links nothing beyond
// the one launch.
func newSessionID(now time.Time) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
	if err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return strconv.FormatInt(now.Unix()*100_000_000+n.Int64(), 10), nil
}

// osName returns the OS name in the form Aptabase's dashboard groups by.
func osName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}

// monthlyState is the on-disk form of the monthly ID, ~/.neat/analytics.json.
type monthlyState struct {
	Month string `json:"month"`
	ID    string `json:"id"`
}

// DefaultIDPath returns ~/.neat/analytics.json.
func DefaultIDPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".neat", "analytics.json"), nil
}

// MonthlyID returns this install's ID for now's calendar month (UTC),
// stored at path. The ID is 128 random bits, derived from nothing about the
// user or computer. When the saved ID belongs to an earlier month (or there
// is none, or it can't be read), a new one replaces it, so IDs from
// different months can't be linked: counting distinct IDs in a month counts
// installs active that month, and nothing more.
func MonthlyID(path string, now time.Time) (string, error) {
	month := now.UTC().Format("2006-01")

	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	if err == nil {
		var st monthlyState
		if json.Unmarshal(data, &st) == nil && st.Month == month && len(st.ID) == 32 {
			return st.ID, nil
		}
	}

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	st := monthlyState{Month: month, ID: hex.EncodeToString(buf)}
	out, err := json.Marshal(st)
	if err != nil {
		return "", fmt.Errorf("marshal monthly id: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create directory for %s: %w", path, err)
	}
	// A unique temp file per call, so two neat processes starting at once
	// can't write or rename each other's file.
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temp file for %s: %w", path, err)
	}
	tmp := f.Name()
	if _, err := f.Write(out); err != nil {
		return "", errors.Join(fmt.Errorf("write %s: %w", tmp, err), f.Close(), os.Remove(tmp))
	}
	if err := f.Close(); err != nil {
		return "", errors.Join(fmt.Errorf("close %s: %w", tmp, err), os.Remove(tmp))
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", errors.Join(fmt.Errorf("rename %s into place: %w", tmp, err), os.Remove(tmp))
	}
	return st.ID, nil
}

// RemoveID deletes the saved monthly ID, if any. Used when analytics are
// turned off, so nothing is left behind.
func RemoveID(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}
