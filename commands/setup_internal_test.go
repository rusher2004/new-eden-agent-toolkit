package commands

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

// syncBuffer is a bytes.Buffer safe to read while checkKeychain writes to it.
type syncBuffer struct {
	ch  chan struct{}
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	n, err := b.buf.Write(p)
	close(b.ch)
	return n, err
}

func stubKeychain(t *testing.T, fn func() error, notice time.Duration) {
	t.Helper()
	origVerify, origNotice := verifyKeychain, keychainWaitNotice
	verifyKeychain, keychainWaitNotice = fn, notice
	t.Cleanup(func() { verifyKeychain, keychainWaitNotice = origVerify, origNotice })
}

func TestCheckKeychain_SlowCheck_PrintsWaitNoticeThenReturnsResult(t *testing.T) {
	release := make(chan struct{})
	wantErr := errors.New("unlock dismissed")
	stubKeychain(t, func() error { <-release; return wantErr }, time.Millisecond)

	out := &syncBuffer{ch: make(chan struct{})}
	result := make(chan error, 1)
	go func() { result <- checkKeychain(out) }()

	select {
	case <-out.ch:
	case <-time.After(5 * time.Second):
		t.Fatal("checkKeychain printed no wait notice while the keychain check was blocked")
	}
	if got := out.buf.String(); !strings.Contains(got, "Waiting for your keychain") {
		t.Errorf("output = %q, want the keychain wait notice", got)
	}

	close(release)
	if err := <-result; !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want the keychain check's own error %v", err, wantErr)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write: no space left on device")
}

func TestCheckKeychain_WaitNoticeWriteFails_StillReturnsKeychainResult(t *testing.T) {
	release := make(chan struct{})
	stubKeychain(t, func() error { <-release; return nil }, time.Millisecond)

	result := make(chan error, 1)
	go func() { result <- checkKeychain(failingWriter{}) }()

	select {
	case err := <-result:
		t.Fatalf("checkKeychain returned %v before the keychain check finished", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-result; err != nil {
		t.Errorf("err = %v, want the keychain check's own result (nil)", err)
	}
}

func TestCheckKeychain_FastCheck_PrintsNothing(t *testing.T) {
	stubKeychain(t, func() error { return nil }, time.Hour)

	var out bytes.Buffer
	if err := checkKeychain(&out); err != nil {
		t.Fatalf("checkKeychain returned error: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("output = %q, want nothing for a check that returns at once", out.String())
	}
}

func TestKeychainHint(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{"linux", "Passwords and Keys"},
		{"darwin", "macOS"},
		{"windows", ""},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			got := keychainHint(tt.goos)
			if tt.want == "" {
				if got != "" {
					t.Errorf("keychainHint(%q) = %q, want no hint", tt.goos, got)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("keychainHint(%q) = %q, want it to mention %q", tt.goos, got, tt.want)
			}
		})
	}
}
