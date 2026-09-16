package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// The plugins folder is the user's, inside a directory that is otherwise setup's own. Setup makes
// it so nobody has to create a folder by name in the right place (FR-576). It keeps it on the
// way out wherever something is in it, unless told otherwise (FR-578).

// folderReader reads the entries of one directory. KeepablePlugins passes os.ReadDir; a test
// hands in one that refuses, since a folder that exists and cannot be read is not something a
// Windows disk will produce on request.
type folderReader func(dir string) ([]os.DirEntry, error)

// PluginsDir answers where the plugins folder is inside the install directory dir.
func PluginsDir(dir string) string { return filepath.Join(dir, product.PluginsFolder) }

// MakePluginsFolder makes the plugins folder inside dir where there is none, leaving one that is
// already there exactly as it is (FR-576).
func MakePluginsFolder(dir string) error {
	folder := PluginsDir(dir)
	if err := os.MkdirAll(folder, dirPerm); err != nil {
		return fmt.Errorf("making %s: %w", folder, refusal.Reason(err))
	}
	return nil
}

// KeepablePlugins answers the plugins folder inside dir where uninstall has something to offer to
// keep; empty where it has nothing (FR-578).
//
// An absent folder and an empty one hold nothing a user would lose. Anything at all in it counts,
// a subfolder as much as a file: a plugin may keep what it needs beside itself; a folder the
// user put there would be lost just as silently as a file. A folder that is there and cannot be
// read is offered too, since what cannot be seen cannot be shown to be empty and the cost of the
// wrong answer is the user's plugins gone without a word.
func KeepablePlugins(dir string) string { return keepablePlugins(dir, os.ReadDir) }

// KeptOnUninstall answers the folder inside dir the uninstall delete leaves standing: the plugins
// folder where it holds anything and removing it was not asked for; empty otherwise, which removes
// the lot (FR-578).
//
// The folder is read again here rather than taken from what the page was shown, so a plugin put
// there while the Uninstall screen stood open is still kept. An empty folder is never left
// behind in an install directory that has otherwise gone.
func KeptOnUninstall(dir string, removePlugins bool) string {
	if removePlugins || KeepablePlugins(dir) == "" {
		return ""
	}
	return product.PluginsFolder
}

// keepablePlugins is KeepablePlugins with the directory read handed in.
func keepablePlugins(dir string, read folderReader) string {
	folder := PluginsDir(dir)
	entries, err := read(folder)
	switch {
	case os.IsNotExist(err):
		return ""
	case err != nil:
		return folder
	case len(entries) == 0:
		return ""
	}
	return folder
}
