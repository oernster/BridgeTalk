// Package appdata finds the product's own folder for what it keeps on this machine. The default
// recordings directory and the made lines both sit inside it, so the rule for where it is has one
// home.
//
// It is local rather than roaming, because hours of audio do not belong in a roaming profile.
package appdata

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/oernster/bridge-talk/internal/product"
)

// windowsOS is runtime.GOOS on Windows, where the local data folder is named by an environment
// variable rather than by a convention under the home directory.
const windowsOS = "windows"

// errNoLocalAppData means Windows gave no local data folder to build on.
var errNoLocalAppData = errors.New("LOCALAPPDATA is not set")

// Dir answers with the product's local data folder, without touching the disk.
func Dir() (string, error) {
	return dir(runtime.GOOS, os.Getenv, os.UserHomeDir)
}

// dir works the folder out from the platform, the environment and the home directory, each a
// parameter so that every platform's rule is exercised on every platform.
func dir(goos string, getenv func(string) string, home func() (string, error)) (string, error) {
	if goos == windowsOS {
		base := getenv("LOCALAPPDATA")
		if base == "" {
			return "", errNoLocalAppData
		}
		return filepath.Join(base, product.Slug), nil
	}
	// Elsewhere the XDG base directory rule applies: XDG_DATA_HOME where it is set, otherwise
	// ~/.local/share.
	base := getenv("XDG_DATA_HOME")
	if base == "" {
		found, err := home()
		if err != nil {
			return "", fmt.Errorf("finding the home directory: %w", err)
		}
		base = filepath.Join(found, ".local", "share")
	}
	return filepath.Join(base, product.Slug), nil
}
