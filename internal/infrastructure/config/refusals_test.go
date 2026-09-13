package config

// FR-237 in the settings store and the cue table loader: a refusal over a file or folder
// names it once, written as the reader would type it, with the reason in plain words and
// no system call.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/refusal"
)

func TestConfigRefusalsNameTheirPathOnce(t *testing.T) {
	base := t.TempDir()
	plainFile := filepath.Join(base, "plain.txt")
	if err := os.WriteFile(plainFile, []byte("x"), filePerm); err != nil {
		t.Fatalf("writing %s: %v", plainFile, err)
	}
	// A settings path under a plain file cannot be made; one that is itself a folder with
	// something inside it cannot be forgotten.
	blocked := &Settings{path: filepath.Join(plainFile, "Settings Folder", settingsFile)}
	held := &Settings{path: filepath.Join(base, "Held", settingsFile)}
	if err := os.MkdirAll(filepath.Join(held.path, "inside"), dirPerm); err != nil {
		t.Fatalf("making %s: %v", held.path, err)
	}
	missingTable := filepath.Join(base, "Missing cues.toml")
	_, loadErr := LoadCueTable(missingTable)

	cases := []struct {
		act     string
		refused error
		path    string
	}{
		{"saving under a plain file", blocked.Save(ports.Settings{}), filepath.Dir(blocked.path)},
		{"forgetting settings that are a folder", held.Forget(), held.path},
		{"loading a cue table that is not there", loadErr, missingTable},
	}
	for _, each := range cases {
		for _, problem := range refusal.Check(each.refused, each.path) {
			t.Errorf("%s %s", each.act, problem)
		}
	}
}
