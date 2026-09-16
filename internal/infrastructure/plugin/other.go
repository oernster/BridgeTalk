//go:build !windows && !linux

package plugin

import "errors"

// OpenLibrary loads nothing anywhere but Windows and Linux.
//
// The stub is here so the package builds and vets on every platform, as the setup package's
// portable half does.
func OpenLibrary(path string) (Library, error) {
	return nil, errors.New("plugins are loaded on Windows and Linux only")
}
