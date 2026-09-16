//go:build windows || linux

package plugin

// Loading a library file and finding its functions. No plugin exists to load, nor can one be built
// here, so these use a library the operating system itself ships: one certainly real and certainly not
// a plugin. Calling into a library is nativelib's, tested there.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/nativelib/nativelibtest"
)

func TestARealLibraryThatIsNoPluginIsRefusedByTheFunctionItLacks(t *testing.T) {
	t.Parallel()

	library, err := OpenLibrary(nativelibtest.SystemLibrary(t))

	if library != nil {
		t.Fatalf("a system library loaded as a plugin")
	}
	if err == nil || !strings.Contains(err.Error(), versionFunction) {
		t.Errorf("refusal = %v, want the missing function named", err)
	}
}

func TestSomethingThatIsNoLibraryAtAllIsRefused(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "not-a-library")
	if err := os.WriteFile(path, []byte("this is text"), 0o600); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	library, err := OpenLibrary(path)

	if library != nil || err == nil {
		t.Fatalf("library = %v, err = %v; want a refusal", library, err)
	}
	if !strings.Contains(err.Error(), "could not be loaded") {
		t.Errorf("refusal = %v, want it to say the file would not load", err)
	}
}

func TestAFileThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := OpenLibrary(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("a library that is not there loaded")
	}
}
