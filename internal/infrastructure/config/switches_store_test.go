package config

// The moments switched off on Chatter are kept in the settings file (FR-628, FR-629). Inside the package
// for the reason store_test.go is: a store is pointed at a path the test chooses.

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
)

// FR-629: the moments switched off are written beside every other setting and read back as written.
func TestTheSwitchesOutliveTheRunInTheSettingsFile(t *testing.T) {
	t.Parallel()
	store := &Settings{path: filepath.Join(t.TempDir(), settingsFile)}
	chosen := ports.Settings{LibraryRoot: "anywhere", Voice: "Grace", SwitchedOff: []string{"Bounty", "Docked"}}

	if err := store.Save(chosen); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := store.Load(); !reflect.DeepEqual(got, chosen) {
		t.Errorf("Load() = %+v, want %+v", got, chosen)
	}
}

// FR-628: a file an older build wrote names no switch, so every moment reads as on.
func TestASettingsFileFromAnOlderBuildHasEveryMomentOn(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), settingsFile)
	older := `{"libraryRoot": "anywhere", "journalDir": "", "voice": "Grace", "machineVoice": ""}`
	if err := os.WriteFile(path, []byte(older), filePerm); err != nil {
		t.Fatalf("writing the older file: %v", err)
	}

	got := (&Settings{path: path}).Load()
	if len(got.SwitchedOff) != 0 || got.Voice != "Grace" {
		t.Errorf("Load() = %+v, want Grace kept and nothing switched off", got)
	}
}
