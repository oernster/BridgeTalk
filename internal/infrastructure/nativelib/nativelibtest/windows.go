//go:build windows

package nativelibtest

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

// kernel32 is a library present on every Windows machine.
const kernel32 = "kernel32.dll"

// systemLibrary answers kernel32 in the system directory.
func systemLibrary() (string, error) {
	system, err := windows.GetSystemDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(system, kernel32), nil
}
