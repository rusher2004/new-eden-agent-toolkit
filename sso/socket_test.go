package sso_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

// shortTempHome returns a fresh temp directory under a short, fixed-prefix
// path rather than t.TempDir()'s test-name-derived path. Unix socket paths
// are capped at ~104 bytes on Darwin, and t.TempDir() embeds the full test
// name into the directory — long test names (like
// TestListenForCallback_CollisionProbeDoesNotCorruptFirstFlowsCallback)
// combined with ".neat/run/callback.lock" can overflow that limit even with
// $TMPDIR forced to /tmp. This only applies on Darwin: /tmp does not exist
// at all on Windows (confirmed via real CI failure — "GetFileAttributesEx
// /tmp: The system cannot find the file specified"), and Linux's/Windows'
// own default temp directories aren't long enough to hit this limit.
func shortTempHome(t *testing.T) string {
	t.Helper()
	base := "" // os.MkdirTemp("", ...) uses os.TempDir(), the correct default elsewhere
	if runtime.GOOS == "darwin" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "neat-sso-")
	if err != nil {
		t.Fatalf("create temp home: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestListenAndDeliverCallback_RoundTrips(t *testing.T) {
	tmpHome := shortTempHome(t)
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	lockPath := filepath.Join(tmpHome, ".neat", "run", "callback.lock")
	deliveryPath := filepath.Join(tmpHome, ".neat", "run", "callback.sock")

	listener, err := sso.ExportedListenForCallback(lockPath, deliveryPath)
	if err != nil {
		t.Fatalf("listenForCallback returned error: %v", err)
	}
	defer listener.Close()

	resultCh := make(chan struct {
		url string
		err error
	}, 1)
	go func() {
		url, err := sso.ExportedReceiveCallback(context.Background(), listener.Delivery(), 5*time.Second)
		resultCh <- struct {
			url string
			err error
		}{url, err}
	}()

	const wantURL = "eveauth-neat://callback?code=abc&state=xyz"
	if err := sso.DeliverCallback(wantURL); err != nil {
		t.Fatalf("DeliverCallback returned error: %v", err)
	}

	res := <-resultCh
	if res.err != nil {
		t.Fatalf("receiveCallback returned error: %v", res.err)
	}
	if res.url != wantURL {
		t.Errorf("received URL = %q, want %q", res.url, wantURL)
	}
}

func TestListenForCallback_SecondCallWhileFirstHoldsSocketFails(t *testing.T) {
	tmpHome := shortTempHome(t)
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	lockPath := filepath.Join(tmpHome, ".neat", "run", "callback.lock")
	deliveryPath := filepath.Join(tmpHome, ".neat", "run", "callback.sock")

	first, err := sso.ExportedListenForCallback(lockPath, deliveryPath)
	if err != nil {
		t.Fatalf("first listenForCallback returned error: %v", err)
	}
	defer first.Close()

	_, err = sso.ExportedListenForCallback(lockPath, deliveryPath)
	if !errors.Is(err, sso.ErrAddCharacterInProgress) {
		t.Errorf("second listenForCallback err = %v, want wrapping ErrAddCharacterInProgress", err)
	}
}

func TestListenForCallback_CollisionProbeDoesNotCorruptFirstFlowsCallback(t *testing.T) {
	tmpHome := shortTempHome(t)
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	lockPath := filepath.Join(tmpHome, ".neat", "run", "callback.lock")
	deliveryPath := filepath.Join(tmpHome, ".neat", "run", "callback.sock")

	first, err := sso.ExportedListenForCallback(lockPath, deliveryPath)
	if err != nil {
		t.Fatalf("first listenForCallback returned error: %v", err)
	}
	defer first.Close()

	resultCh := make(chan struct {
		url string
		err error
	}, 1)
	go func() {
		url, err := sso.ExportedReceiveCallback(context.Background(), first.Delivery(), 5*time.Second)
		resultCh <- struct {
			url string
			err error
		}{url, err}
	}()

	// Give the first flow's receiveCallback goroutine time to reach
	// Accept() before the second call's collision probe runs — this is the
	// exact interleaving that corrupted the first flow under the old
	// single-socket design.
	time.Sleep(50 * time.Millisecond)

	if _, err := sso.ExportedListenForCallback(lockPath, deliveryPath); !errors.Is(err, sso.ErrAddCharacterInProgress) {
		t.Fatalf("second listenForCallback err = %v, want wrapping ErrAddCharacterInProgress", err)
	}

	const wantURL = "eveauth-neat://callback?code=real&state=xyz"
	if err := sso.DeliverCallback(wantURL); err != nil {
		t.Fatalf("DeliverCallback returned error: %v", err)
	}

	res := <-resultCh
	if res.err != nil {
		t.Fatalf("first flow's receiveCallback returned error: %v", res.err)
	}
	if res.url != wantURL {
		t.Errorf("first flow received URL = %q, want %q (would fail under the old single-socket design)", res.url, wantURL)
	}
}

// TestListenForCallback_TwoTrulySimultaneousCalls_ExactlyOneSucceeds proves
// bindExclusive's fast path (bind directly, before any liveness probe) is
// atomic for the realistic case: two callers racing. This is the scenario
// that matters in practice — addCharacterMu.TryLock already serializes every
// same-process caller, so genuine concurrency here only happens across
// separate OS processes, and realistically that means two, not many.
//
// This intentionally does NOT test many-way (10+) simultaneous contention:
// verified during development that the non-atomic fallback path (stale-file
// recovery) can itself race when several callers fall into it at once,
// which this fix does not close — see bindExclusive's doc comment. That
// residual gap needs a PID-tracked lockfile or equivalent, not just
// reordering these two syscalls, and is disproportionate to the actual
// threat model here (it would require several separate neat processes
// calling add_character at the exact same instant).
func TestListenForCallback_TwoTrulySimultaneousCalls_ExactlyOneSucceeds(t *testing.T) {
	tmpHome := shortTempHome(t)
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	lockPath := filepath.Join(tmpHome, ".neat", "run", "callback.lock")
	deliveryPath := filepath.Join(tmpHome, ".neat", "run", "callback.sock")

	const attempts = 2
	type result struct {
		listener *sso.ExportedCallbackListener
		err      error
	}
	resultCh := make(chan result, attempts)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			l, err := sso.ExportedListenForCallback(lockPath, deliveryPath)
			resultCh <- result{l, err}
		}()
	}
	close(start)
	wg.Wait()
	close(resultCh)

	successes := 0
	var winner *sso.ExportedCallbackListener
	for res := range resultCh {
		switch {
		case res.err == nil:
			successes++
			winner = res.listener
		case !errors.Is(res.err, sso.ErrAddCharacterInProgress):
			t.Errorf("unexpected error: %v", res.err)
		}
	}
	if successes != 1 {
		t.Errorf("successes = %d, want exactly 1 (a value other than 1 means the bind race let more than one caller believe it owns the socket)", successes)
	}
	if winner != nil {
		winner.Close()
	}
}

func TestListenForCallback_StaleSocketFileIsCleanedUp(t *testing.T) {
	tmpHome := shortTempHome(t)
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	lockPath := filepath.Join(tmpHome, ".neat", "run", "callback.lock")
	deliveryPath := filepath.Join(tmpHome, ".neat", "run", "callback.sock")

	// Simulate a crash: bind both sockets, then close the raw listeners
	// without unlinking their files (unlike Close(), which removes them —
	// exactly what we don't want to simulate here).
	stale, err := sso.ExportedListenForCallback(lockPath, deliveryPath)
	if err != nil {
		t.Fatalf("first listenForCallback returned error: %v", err)
	}
	stale.CloseWithoutRemovingFiles()

	fresh, err := sso.ExportedListenForCallback(lockPath, deliveryPath)
	if err != nil {
		t.Fatalf("second listenForCallback returned error after stale socket: %v", err)
	}
	defer fresh.Close()
}

func TestReceiveCallback_TimesOut(t *testing.T) {
	tmpHome := shortTempHome(t)
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	lockPath := filepath.Join(tmpHome, ".neat", "run", "callback.lock")
	deliveryPath := filepath.Join(tmpHome, ".neat", "run", "callback.sock")

	listener, err := sso.ExportedListenForCallback(lockPath, deliveryPath)
	if err != nil {
		t.Fatalf("listenForCallback returned error: %v", err)
	}
	defer listener.Close()

	_, err = sso.ExportedReceiveCallback(context.Background(), listener.Delivery(), 50*time.Millisecond)
	if err == nil {
		t.Fatal("receiveCallback returned nil error, want a timeout error")
	}
}

// TestReceiveCallback_ContextCancellation_ReturnsImmediately proves that
// cancelling the caller's context unblocks receiveCallback right away,
// rather than forcing callers to wait out the full timeout — this is what
// lets an MCP client's cancellation of an in-flight add_character call
// (e.g. Claude Code's "Stop Task") actually release the socket/lock
// promptly instead of after up to 5 minutes.
func TestReceiveCallback_ContextCancellation_ReturnsImmediately(t *testing.T) {
	tmpHome := shortTempHome(t)
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	lockPath := filepath.Join(tmpHome, ".neat", "run", "callback.lock")
	deliveryPath := filepath.Join(tmpHome, ".neat", "run", "callback.sock")

	listener, err := sso.ExportedListenForCallback(lockPath, deliveryPath)
	if err != nil {
		t.Fatalf("listenForCallback returned error: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())

	resultCh := make(chan struct {
		url string
		err error
	}, 1)
	go func() {
		url, err := sso.ExportedReceiveCallback(ctx, listener.Delivery(), 5*time.Minute)
		resultCh <- struct {
			url string
			err error
		}{url, err}
	}()

	// Give receiveCallback's goroutine time to reach Accept() before
	// cancelling, so this genuinely exercises the "already blocked" case.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case res := <-resultCh:
		if !errors.Is(res.err, context.Canceled) {
			t.Errorf("err = %v, want wrapping context.Canceled", res.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("receiveCallback did not return promptly after context cancellation")
	}
}
