package voicefiles_test

// FR-539: the application reads the files from the folder models beside its own executable.

import (
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

func TestTheFilesAreReadFromTheFolderBesideTheApplication(t *testing.T) {
	t.Parallel()
	installed := t.TempDir()
	application := filepath.Join(installed, "application.exe")

	got := voicefiles.Beside(application)

	if want := filepath.Join(installed, voicefiles.Folder); got != want || voicefiles.Folder != "models" {
		t.Errorf("Beside(%q) = %q, want %q named models", application, got, want)
	}
}
