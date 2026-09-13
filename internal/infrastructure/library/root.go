package library

// The recordings directory offered before any has been chosen (FR-227).
//
// It belongs to the product rather than to the game. Given no folder, the system's own
// dialog chooses for itself; it opened in the game's folder, the one place a voice's
// recordings do not belong.
//
// Of the folders the product owns it is the one nothing removes. Uninstall deletes the
// install directory; forgetting settings deletes the window's state folder and the
// settings file. Recordings kept in any of those would be lost to an ordinary
// uninstall. It is local rather than roaming, because hours of audio do not belong in a
// roaming profile.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/oernster/bridge-talk/internal/product"
)

// recordingsFolder names the default recordings directory inside the product's own
// data folder.
const recordingsFolder = "Recordings"

// windowsOS is runtime.GOOS on Windows, where the local data folder is named by an
// environment variable rather than by a convention under the home directory.
const windowsOS = "windows"

// errNoLocalAppData means Windows gave no local data folder to put the default in.
var errNoLocalAppData = errors.New("LOCALAPPDATA is not set")

// DefaultRoot answers with the default recordings directory, making it where it is
// missing so a folder dialog can open inside it.
func DefaultRoot() (string, error) {
	dir, err := defaultRoot(runtime.GOOS, os.Getenv, os.UserHomeDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, folderPerm); err != nil {
		return "", fmt.Errorf("making %s: %w", dir, err)
	}
	return dir, nil
}

// defaultRoot works the directory out without touching the disk.
//
// The platform, the environment and the home directory are parameters so that each
// platform's rule is exercised on every platform, rather than only on the one a test
// happens to run on.
func defaultRoot(goos string, getenv func(string) string, home func() (string, error)) (string, error) {
	if goos == windowsOS {
		base := getenv("LOCALAPPDATA")
		if base == "" {
			return "", errNoLocalAppData
		}
		return filepath.Join(base, product.Slug, recordingsFolder), nil
	}
	// Elsewhere the XDG base directory rule applies: XDG_DATA_HOME where it is set,
	// otherwise ~/.local/share.
	base := getenv("XDG_DATA_HOME")
	if base == "" {
		dir, err := home()
		if err != nil {
			return "", fmt.Errorf("finding the home directory: %w", err)
		}
		base = filepath.Join(dir, ".local", "share")
	}
	return filepath.Join(base, product.Slug, recordingsFolder), nil
}
