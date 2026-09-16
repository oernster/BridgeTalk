package config

// These tests are inside the package so they can point a store at a temporary file.
// The alternative was a second constructor taking a path, which would widen the API
// for the benefit of the tests alone.

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
)

// storeIn builds a store writing inside a temporary directory that does not exist
// yet, so every test also exercises the directory being created.
func storeIn(t *testing.T) *Settings {
	t.Helper()
	return &Settings{path: filepath.Join(t.TempDir(), "nested", settingsFile)}
}

// TestNothingChosenYetLoadsAsNothing covers the first run, which is the common case.
func TestNothingChosenYetLoadsAsNothing(t *testing.T) {
	t.Parallel()
	if held := storeIn(t).Load(); !reflect.DeepEqual(held, ports.Settings{}) {
		t.Errorf("a store with no file loaded %+v, want the zero value", held)
	}
}

// TestChoicesSurviveASave is the whole point of the store.
func TestChoicesSurviveASave(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	want := ports.Settings{
		LibraryRoot: filepath.Join("D:", "Recordings"),
		JournalDir:  filepath.Join("E:", "Journals"),
		Voice:       "Hugo",
		// FR-540 and FR-569: every kind of voice survives a save, though casting keeps only one of
		// them. A plugin voice takes two fields, since an id is unique only within its plugin.
		MachineVoice: "bf_emma",
		Plugin:       "Bridge Crew",
		PluginVoice:  "one",
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("saving: %v", err)
	}
	if got := store.Load(); !reflect.DeepEqual(got, want) {
		t.Errorf("loaded %+v, want %+v", got, want)
	}
}

// TestAFileThatDoesNotParseLoadsAsNothing keeps a damaged file from stopping a start.
//
// A truncated or hand-edited file is not different to a reader from never having
// chosen anything: in both cases there is nothing to honour, so the application
// detects as usual rather than refusing to run over its own settings file.
func TestAFileThatDoesNotParseLoadsAsNothing(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	if err := os.MkdirAll(filepath.Dir(store.path), dirPerm); err != nil {
		t.Fatalf("preparing: %v", err)
	}
	if err := os.WriteFile(store.path, []byte("{ this is not json"), filePerm); err != nil {
		t.Fatalf("planting: %v", err)
	}
	if held := store.Load(); !reflect.DeepEqual(held, ports.Settings{}) {
		t.Errorf("a damaged file loaded %+v, want the zero value", held)
	}
}

// TestASaveLeavesNoWorkingFileBehind checks the rename actually happened.
//
// The write goes to a temporary file beside the target and is renamed over it, so an
// interrupted save leaves the previous settings rather than a truncated file that
// would then load as nothing at all.
func TestASaveLeavesNoWorkingFileBehind(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	if err := store.Save(ports.Settings{LibraryRoot: "somewhere"}); err != nil {
		t.Fatalf("saving: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(store.path))
	if err != nil {
		t.Fatalf("reading the directory: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != settingsFile {
			t.Errorf("a save left %q behind", entry.Name())
		}
	}
}

// TestASecondSaveReplacesTheFirst proves the rename overwrites rather than failing on
// an existing file, which is what a second choice depends on.
func TestASecondSaveReplacesTheFirst(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	if err := store.Save(ports.Settings{LibraryRoot: "first"}); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := store.Save(ports.Settings{LibraryRoot: "second"}); err != nil {
		t.Fatalf("second save: %v", err)
	}
	if got := store.Load().LibraryRoot; got != "second" {
		t.Errorf("library root is %q, want the second choice", got)
	}
}

// TestAMachineWithNoConfigDirectoryStillStarts covers the store built without a path.
//
// It loads nothing rather than failing, so the application runs on detected paths; a
// save says plainly that there is nowhere to put it rather than reporting success.
func TestAMachineWithNoConfigDirectoryStillStarts(t *testing.T) {
	t.Parallel()
	store := &Settings{}
	if held := store.Load(); !reflect.DeepEqual(held, ports.Settings{}) {
		t.Errorf("loaded %+v, want the zero value", held)
	}
	if err := store.Save(ports.Settings{LibraryRoot: "anywhere"}); err == nil {
		t.Error("saving with nowhere to write reported success")
	}
	if store.Path() != "" {
		t.Errorf("path is %q, want empty", store.Path())
	}
}
