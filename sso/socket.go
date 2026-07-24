package sso

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ErrAddCharacterInProgress is returned when add_character is called while
// another add-character flow is already waiting for its callback.
var ErrAddCharacterInProgress = errors.New("a character-add flow is already in progress")

// callbackListener bundles the two sockets one add-character flow holds: a
// lock socket (bound but never Accept()-ed — purely a liveness signal, so a
// second concurrent call's collision probe can never be mistaken for the
// real callback) and a delivery socket (Accept()-ed exactly once, by
// receiveCallback).
type callbackListener struct {
	lock         net.Listener
	delivery     net.Listener
	lockPath     string
	deliveryPath string
}

// Close releases both sockets and removes their files.
func (c *callbackListener) Close() error {
	var errs []error
	if err := c.delivery.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close delivery socket: %w", err))
	}
	if err := os.Remove(c.deliveryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, fmt.Errorf("remove delivery socket file: %w", err))
	}
	if err := c.lock.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close lock socket: %w", err))
	}
	if err := os.Remove(c.lockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, fmt.Errorf("remove lock socket file: %w", err))
	}
	return errors.Join(errs...)
}

// listenForCallback acquires the add-character lock at lockPath and opens
// the delivery socket at deliveryPath, failing with
// ErrAddCharacterInProgress if another flow already holds the lock. A lock
// or delivery socket file left behind by a crashed previous run (nothing
// listening) is treated as stale and removed.
//
// The lock and delivery sockets are deliberately separate: if they were the
// same socket, a second call's liveness probe (a dial-then-close) could
// itself be accepted by the first call's single receiveCallback Accept(),
// silently corrupting the first (legitimate) flow with an empty payload
// instead of the real callback.
func listenForCallback(lockPath, deliveryPath string) (*callbackListener, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return nil, fmt.Errorf("create run directory: %w", err)
	}

	lock, err := bindExclusive(lockPath)
	if err != nil {
		return nil, err
	}

	delivery, err := bindExclusive(deliveryPath)
	if err != nil {
		lock.Close()
		os.Remove(lockPath)
		return nil, fmt.Errorf("open delivery socket: %w", err)
	}

	return &callbackListener{lock: lock, delivery: delivery, lockPath: lockPath, deliveryPath: deliveryPath}, nil
}

// bindExclusive binds a Unix socket at path. An existing file with nothing
// listening behind it is treated as stale (removed and rebound); an
// existing file with something actively listening returns
// ErrAddCharacterInProgress.
//
// Binding is attempted directly first, before any liveness probe: a Unix
// socket bind() is atomic at the OS level, so a successful Listen here
// means no other process could have been bound at this path — no TOCTOU
// window. This closes the common-case race between two genuinely
// concurrent callers: only one of their Listen calls can win, and the
// other correctly falls through to the liveness probe below, which finds
// the winner already, genuinely bound. (A narrower race remains when two
// callers are both recovering the same stale, crashed-process file at the
// same instant — that fallback path isn't fully atomic, and would need a
// PID-tracked lockfile or equivalent to close completely.)
func bindExclusive(path string) (net.Listener, error) {
	if l, err := net.Listen("unix", path); err == nil {
		if err := os.Chmod(path, 0o600); err != nil {
			l.Close()
			return nil, fmt.Errorf("set socket permissions: %w", err)
		}
		return l, nil
	}

	if conn, err := net.DialTimeout("unix", path, 200*time.Millisecond); err == nil {
		conn.Close()
		return nil, ErrAddCharacterInProgress
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("remove stale socket %s: %w", path, err)
	}

	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		l.Close()
		return nil, fmt.Errorf("set socket permissions: %w", err)
	}
	return l, nil
}

// receiveCallback blocks until a callback-handler process delivers the
// redirect URL via DeliverCallback, or timeout elapses.
func receiveCallback(l net.Listener, timeout time.Duration) (string, error) {
	type result struct {
		url string
		err error
	}
	ch := make(chan result, 1)

	go func() {
		conn, err := l.Accept()
		if err != nil {
			ch <- result{err: fmt.Errorf("accept callback connection: %w", err)}
			return
		}
		defer conn.Close()

		data, err := io.ReadAll(conn)
		if err != nil {
			ch <- result{err: fmt.Errorf("read callback payload: %w", err)}
			return
		}
		ch <- result{url: string(data)}
	}()

	select {
	case res := <-ch:
		return res.url, res.err
	case <-time.After(timeout):
		return "", fmt.Errorf("timed out after %s waiting for browser callback", timeout)
	}
}

// DeliverCallback connects to the waiting add_character flow's delivery
// socket and hands it the full redirect URL. Called by the
// callback-handler subcommand, which the OS launches in a separate process
// on URL-scheme activation.
func DeliverCallback(rawURL string) error {
	path, err := callbackSocketPath()
	if err != nil {
		return err
	}
	conn, err := net.Dial("unix", path)
	if err != nil {
		return fmt.Errorf("connect to callback socket: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(rawURL)); err != nil {
		return fmt.Errorf("write callback payload: %w", err)
	}
	return nil
}

// ExportedCallbackListener exposes callbackListener for external tests.
type ExportedCallbackListener struct {
	inner *callbackListener
}

// Delivery returns the delivery listener, for passing to ExportedReceiveCallback.
func (e *ExportedCallbackListener) Delivery() net.Listener {
	return e.inner.delivery
}

// Close releases both sockets and removes their files.
func (e *ExportedCallbackListener) Close() error {
	return e.inner.Close()
}

// CloseWithoutRemovingFiles closes the underlying listeners without
// unlinking their socket files, for tests simulating a crashed process.
func (e *ExportedCallbackListener) CloseWithoutRemovingFiles() {
	e.inner.lock.Close()
	e.inner.delivery.Close()
}

// ExportedListenForCallback exposes listenForCallback for external tests.
func ExportedListenForCallback(lockPath, deliveryPath string) (*ExportedCallbackListener, error) {
	l, err := listenForCallback(lockPath, deliveryPath)
	if err != nil {
		return nil, err
	}
	return &ExportedCallbackListener{inner: l}, nil
}

// ExportedReceiveCallback exposes receiveCallback for external tests.
func ExportedReceiveCallback(l net.Listener, timeout time.Duration) (string, error) {
	return receiveCallback(l, timeout)
}
