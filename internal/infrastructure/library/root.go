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
// uninstall. Where the product's data folder is on each platform is appdata's rule.

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/infrastructure/appdata"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// recordingsFolder names the default recordings directory inside the product's own
// data folder.
const recordingsFolder = "Recordings"

// DefaultRoot answers with the default recordings directory, making it where it is
// missing so a folder dialog can open inside it.
func DefaultRoot() (string, error) {
	base, err := appdata.Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, recordingsFolder)
	if err := os.MkdirAll(dir, folderPerm); err != nil {
		return "", fmt.Errorf("making %s: %w", dir, refusal.Reason(err))
	}
	return dir, nil
}
