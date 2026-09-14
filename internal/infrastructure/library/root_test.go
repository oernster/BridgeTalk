package library

// The default recordings directory (FR-227). Where the product's data folder is on each platform
// is appdata's rule, tested there; the directory itself is made only under a temporary data
// folder, so no test writes to the machine it runs on.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
	"github.com/oernster/bridge-talk/internal/product"
)

// It is made where missing, so a folder dialog has somewhere to open; asking again
// finds it rather than failing.
func TestTheDefaultRecordingsDirectoryIsMadeWhereMissing(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	t.Setenv("XDG_DATA_HOME", base)

	for range 2 {
		dir, err := DefaultRoot()
		if err != nil {
			t.Fatalf("making the default: %v", err)
		}
		if dir != filepath.Join(base, product.Slug, recordingsFolder) {
			t.Fatalf("made %q, want it under the data folder", dir)
		}
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("no directory at %q: %v", dir, err)
		}
	}
}

// A default that cannot be worked out or cannot be made is reported, not guessed at.
func TestADefaultThatCannotBeMadeIsReported(t *testing.T) {
	file := filepath.Join(t.TempDir(), "plain")
	audiotest.WriteFile(t, file, audiotest.NotARecording)
	t.Setenv("LOCALAPPDATA", file)
	t.Setenv("XDG_DATA_HOME", file)
	if _, err := DefaultRoot(); err == nil {
		t.Error("a default was made inside a file")
	}

	for _, key := range []string{"LOCALAPPDATA", "XDG_DATA_HOME", "HOME", "USERPROFILE"} {
		t.Setenv(key, "")
	}
	if _, err := DefaultRoot(); err == nil {
		t.Error("a default was invented with nowhere to put it")
	}
}
