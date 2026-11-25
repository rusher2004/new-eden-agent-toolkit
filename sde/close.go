package sde

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// closeInto closes c and joins a failure into *err, for use as
// `defer closeInto(&err, rows, "rows")` in a function with a named error
// result, so a cleanup error is reported rather than dropped.
func closeInto(err *error, c io.Closer, what string) {
	if cerr := c.Close(); cerr != nil {
		*err = errors.Join(*err, fmt.Errorf("close %s: %w", what, cerr))
	}
}

// removeTemp deletes a temp file left by a failed write, returning a
// descriptive error to join into the failure that made it unwanted.
func removeTemp(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove temp file %s: %w", path, err)
	}
	return nil
}

// discardTemp closes f and deletes its file, joining both failures.
func discardTemp(f *os.File, path string) error {
	var cerr error
	if err := f.Close(); err != nil {
		cerr = fmt.Errorf("close temp file %s: %w", path, err)
	}
	return errors.Join(cerr, removeTemp(path))
}
