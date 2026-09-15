package config

// Forgetting is what an uninstall asks for when the reader ticks the box to forget their
// settings. These tests point a store at a temporary tree, as the other store tests do,
// so the real user configuration directory is never touched.

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
)

func TestForgettingRemovesTheChoicesAndTheirDirectory(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	if err := store.Save(ports.Settings{LibraryRoot: "somewhere", Voice: "Hugo"}); err != nil {
		t.Fatalf("saving: %v", err)
	}
	// An interrupted save leaves its working file behind; forgetting takes that too.
	if err := os.WriteFile(store.path+writingSuffix, []byte("{"), filePerm); err != nil {
		t.Fatalf("planting the working file: %v", err)
	}

	if err := store.Forget(); err != nil {
		t.Fatalf("forgetting: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(store.path)); !os.IsNotExist(err) {
		t.Errorf("the settings directory survived: %v", err)
	}
	if held := store.Load(); !reflect.DeepEqual(held, ports.Settings{}) {
		t.Errorf("after forgetting the store loaded %+v, want nothing", held)
	}
}

// The directory is the application's, yet something else may have been put in it. That
// is not the store's to delete, so it stays and so does the directory holding it.
func TestForgettingLeavesWhatIsNotTheStoresOwn(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	if err := store.Save(ports.Settings{Voice: "Hugo"}); err != nil {
		t.Fatalf("saving: %v", err)
	}
	occupant := filepath.Join(filepath.Dir(store.path), "not ours.txt")
	if err := os.WriteFile(occupant, []byte("x"), filePerm); err != nil {
		t.Fatalf("planting the occupant: %v", err)
	}

	if err := store.Forget(); err != nil {
		t.Fatalf("forgetting: %v", err)
	}
	if _, err := os.Stat(store.path); !os.IsNotExist(err) {
		t.Errorf("the settings file survived: %v", err)
	}
	if _, err := os.Stat(occupant); err != nil {
		t.Errorf("something that was not the store's was removed: %v", err)
	}
}

// A copy that never saved a choice has nothing to forget; nor does a machine with
// nowhere to keep one. Neither is a failure.
func TestForgettingWhatWasNeverChosenIsNotAFailure(t *testing.T) {
	t.Parallel()
	if err := storeIn(t).Forget(); err != nil {
		t.Errorf("forgetting with nothing saved: %v", err)
	}
	if err := (&Settings{}).Forget(); err != nil {
		t.Errorf("forgetting with nowhere to keep settings: %v", err)
	}
}

func TestForgettingThatCannotRemoveTheFileReports(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := &Settings{path: filepath.Join(dir, settingsFile)}
	// A non-empty directory standing where the settings file belongs cannot be removed
	// as a file on any platform this application ships to.
	if err := os.Mkdir(store.path, dirPerm); err != nil {
		t.Fatalf("planting the blocker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.path, "occupant"), []byte("x"), filePerm); err != nil {
		t.Fatalf("filling the blocker: %v", err)
	}

	if err := store.Forget(); err == nil {
		t.Fatal("a settings file that could not be removed was reported as forgotten")
	}
}
