//go:build !windows && !linux

package nativelibtest

import "errors"

// systemLibrary answers none here, where no native library is loaded.
func systemLibrary() (string, error) {
	return "", errors.New("native libraries are loaded on Windows and Linux only")
}
