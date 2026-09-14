package modelfiles_test

// The folder the files are kept in sits at the repository root, beside go.mod.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
)

// From anywhere in the repository the folder is models beside go.mod.
func TestTheFolderSitsBesideGoMod(t *testing.T) {
	t.Parallel()
	working, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}

	dir, err := modelfiles.Dir(working)

	if err != nil || filepath.Base(dir) != modelfiles.Folder {
		t.Fatalf("Dir = %q, %v; want a folder named %s", dir, err, modelfiles.Folder)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "go.mod")); err != nil {
		t.Errorf("no go.mod beside %s: %v", dir, err)
	}
}
