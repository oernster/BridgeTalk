//go:build !windows

package window

import "errors"

// Reveal is not built for this platform yet, so it says so rather than doing nothing.
func Reveal(string) error {
	return errors.New("opening a folder is not built for this platform yet")
}
