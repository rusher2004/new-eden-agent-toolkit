package commands_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/rusher2004/new-eden-agent-toolkit/commands"
	"github.com/rusher2004/new-eden-agent-toolkit/sso"
)

func TestCallbackHandlerCommand_RequiresExactlyOneArgument(t *testing.T) {
	root := commands.Root("dev", "test-client-id")

	err := root.Run(context.Background(), []string{"neat", "callback-handler"})
	if err == nil {
		t.Fatal("Run returned nil error, want an error for missing URL argument")
	}
}

func TestCallbackHandlerCommand_RejectsTooManyArguments(t *testing.T) {
	root := commands.Root("dev", "test-client-id")

	err := root.Run(context.Background(), []string{"neat", "callback-handler", "url1", "url2"})
	if err == nil {
		t.Fatal("Run returned nil error, want an error for too many arguments")
	}
}

func TestCallbackHandlerCommand_RejectsUnexpectedScheme(t *testing.T) {
	root := commands.Root("dev", "test-client-id")

	err := root.Run(context.Background(), []string{"neat", "callback-handler", "https://evil.example.com/callback"})
	if err == nil {
		t.Fatal("Run returned nil error, want an error for a URL with an unexpected scheme")
	}
}

func TestCallbackHandlerCommand_DeliversValidCallback(t *testing.T) {
	// /tmp doesn't exist at all on Windows (confirmed via real CI failure);
	// os.TempDir()'s default is short enough there and on Linux. On Darwin,
	// /tmp avoids overflowing the ~104-byte sockaddr_un limit that a long
	// default $TMPDIR can hit once ".neat/run/callback.sock" is appended —
	// see sso/socket_test.go's shortTempHome for the full rationale.
	base := ""
	if runtime.GOOS == "darwin" {
		base = "/tmp"
	}
	tmpHome, err := os.MkdirTemp(base, "neat-cmd-")
	if err != nil {
		t.Fatalf("create temp home: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpHome) })
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	lockPath := filepath.Join(tmpHome, ".neat", "run", "callback.lock")
	deliveryPath := filepath.Join(tmpHome, ".neat", "run", "callback.sock")
	listener, err := sso.ExportedListenForCallback(lockPath, deliveryPath)
	if err != nil {
		t.Fatalf("listenForCallback returned error: %v", err)
	}
	defer listener.Close()

	type result struct {
		url string
		err error
	}
	resultCh := make(chan result, 1)
	go func() {
		url, err := sso.ExportedReceiveCallback(context.Background(), listener.Delivery(), 5*time.Second)
		resultCh <- result{url, err}
	}()

	root := commands.Root("dev", "test-client-id")
	const wantURL = "eveauth-neat://callback?code=abc&state=xyz"
	if err := root.Run(context.Background(), []string{"neat", "callback-handler", wantURL}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	res := <-resultCh
	if res.err != nil {
		t.Fatalf("receiveCallback returned error: %v", res.err)
	}
	if res.url != wantURL {
		t.Errorf("received URL = %q, want %q", res.url, wantURL)
	}
}

func TestCallbackHandlerCommand_IsHidden(t *testing.T) {
	root := commands.Root("dev", "test-client-id")

	for _, cmd := range root.Commands {
		if cmd.Name == "callback-handler" {
			if !cmd.Hidden {
				t.Error("callback-handler command is not marked Hidden")
			}
			return
		}
	}
	t.Fatal("callback-handler command not found in Root()")
}
