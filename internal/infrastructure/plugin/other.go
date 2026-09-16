//go:build !windows

package plugin

import "errors"

// OpenLibrary loads nothing anywhere but Windows, which is the only platform Bridge Talk
// ships on today.
//
// The stub is here so the package builds and vets on every platform, as the setup package's
// portable half does. Linux is in scope later and a plugin there is a shared object built
// from the same repository, so this is where that goes.
func OpenLibrary(path string) (Library, error) {
	return nil, errors.New("plugins are loaded on Windows only")
}
