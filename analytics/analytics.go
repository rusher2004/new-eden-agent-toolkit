// Package analytics sends neat's anonymous usage ping to Aptabase
// (https://aptabase.com), so the project can count monthly users, and see
// which versions and platforms are in use, without learning who anyone is.
//
// Each `neat mcp` launch sends exactly one event, "app_started", carrying
// only: neat's version, the OS and CPU architecture it was built for
// (runtime.GOOS and runtime.GOARCH), a random per-process session ID, and a
// random monthly ID (see MonthlyID).
// No character, account, path, hostname, locale or hardware identifier is
// ever sent. Aptabase sees the client's IP address in transit, derives a
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
	"time"
)

// EventName is the only event neat sends.
const EventName = "app_started"

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
	ev := event{
		Timestamp: now.UTC().Format("2006-01-02T15:04:05Z"),
		SessionID: sessionID,
		EventName: EventName,
		SystemProps: systemProps{
			IsDebug:    c.version == "dev",
			OSName:     osName(),
			AppVersion: c.version,
			SDKVersion: "neat@" + c.version,
		},
		Props: map[string]string{
			"monthlyId": monthlyID,
			"arch":      runtime.GOARCH,
		},
	}
	body, err := json.Marshal([]event{ev})
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
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
		return fmt.Errorf("send event: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		detail, err := io.ReadAll(io.LimitReader(resp.Body, 512))
		if err != nil {
			return fmt.Errorf("send event: aptabase returned %s (body unreadable: %w)", resp.Status, err)
		}
		return fmt.Errorf("send event: aptabase returned %s: %s", resp.Status, strings.TrimSpace(string(detail)))
	}
	return nil
}

// newSessionID builds a session ID the way Aptabase's own SDKs do: epoch
// seconds followed by eight random digits. It's random per process, so it
// links nothing beyond the one launch.
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
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", fmt.Errorf("rename %s into place: %w", tmp, err)
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
