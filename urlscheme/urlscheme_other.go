//go:build !darwin && !windows && !linux

package urlscheme

import "fmt"

// register is the fallback for platforms without a real eveauth-neat://
// registration implementation (darwin, windows, and linux each provide
// their own register() in a build-tag-separated file).
func register(_ string) error {
	return fmt.Errorf("URL scheme registration is not supported on this platform")
}
