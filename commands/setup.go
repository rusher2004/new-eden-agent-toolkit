package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/rusher2004/new-eden-agent-toolkit/config"
	"github.com/rusher2004/new-eden-agent-toolkit/creds"
)

// verifyKeychain is the keychain round-trip setup runs. Overridable in
// internal tests, which need a check that blocks the way a locked keychain
// waiting on an unlock prompt does.
var verifyKeychain = creds.VerifyAccess

// keychainWaitNotice is how long the keychain check may run before setup
// says it's waiting on the user. A locked keychain doesn't fail: the OS
// shows an unlock prompt and the check blocks until it's answered, which
// looks like a hang when the prompt opens somewhere the user isn't looking.
var keychainWaitNotice = 5 * time.Second

func setupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "prepare neat's local config directory and verify keychain access",
		Action: func(_ context.Context, cmd *cli.Command) error {
			store, err := config.NewStore()
			if err != nil {
				return fmt.Errorf("resolve config store: %w", err)
			}
			return runSetup(cmd.Writer, store)
		},
	}
}

func runSetup(w io.Writer, store *config.Store) error {
	if _, err := os.Stat(store.Path()); errors.Is(err, os.ErrNotExist) {
		// Update, not Save of an empty Config: the YAML files may already
		// hold characters and profiles, which an empty save would delete.
		if _, err := store.Update(func(*config.Config) error { return nil }); err != nil {
			return fmt.Errorf("initialize config file: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("check config file: %w", err)
	}
	if _, err := fmt.Fprintf(w, "Config directory ready at %s\n", store.Path()); err != nil {
		return fmt.Errorf("print config directory status: %w", err)
	}

	if err := checkKeychain(w); err != nil {
		return fmt.Errorf("keychain check: %w%s", err, keychainHint(runtime.GOOS))
	}
	if _, err := fmt.Fprintln(w, "Keychain access verified."); err != nil {
		return fmt.Errorf("print keychain status: %w", err)
	}

	return nil
}

// checkKeychain runs verifyKeychain, and if it hasn't returned after
// keychainWaitNotice, tells the user it's probably waiting on an unlock
// prompt before waiting on.
func checkKeychain(w io.Writer) error {
	done := make(chan error, 1)
	go func() { done <- verifyKeychain() }()

	timer := time.NewTimer(keychainWaitNotice)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
	}
	if _, err := fmt.Fprintln(w, keychainWaitMessage(runtime.GOOS)); err != nil {
		// The notice is only a courtesy: still wait for the check, so its
		// own result is what setup reports.
		slog.Default().Warn("setup: failed to print the keychain wait notice", "error", err)
	}
	return <-done
}

// keychainWaitMessage is printed while the keychain check is still blocked,
// most likely on an unlock prompt.
func keychainWaitMessage(goos string) string {
	msg := "Waiting for your keychain. If your computer asks you to unlock it or allow access, enter your password there."
	if goos == "linux" {
		msg += " No prompt? It may have opened on another display, such as a remote desktop session. Press Ctrl+C and see the README's Troubleshooting section."
	}
	return msg
}

// keychainHint is appended to a failed keychain check: what to do about it
// on goos, which is where nearly every failure comes from.
func keychainHint(goos string) string {
	switch goos {
	case "linux":
		return "\n\nneat stores logins in your desktop's Secret Service keyring (GNOME Keyring or KWallet). " +
			"Make sure one is running and has a default keyring you can unlock: an error naming /org/freedesktop/secrets/collection/login means the default keyring is missing or wasn't loaded. " +
			"Open Passwords and Keys (seahorse), add a Password keyring with + if none is listed, set it as the default, then run neat setup again. " +
			"The README's Troubleshooting section has more."
	case "darwin":
		return "\n\nAllow neat when macOS asks for keychain access, then run neat setup again."
	default:
		return ""
	}
}
