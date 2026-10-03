package config

// A settings file that will not parse is the user's choices with a fault in them, never nothing
// chosen. Every remembered choice is a load, one field changed and a save, so a damaged file read
// as empty was written over by the next choice, which then held that choice alone. It is now kept
// aside before anything is written, beside any copy kept aside earlier; the store says so.

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
)

// seededThenDamaged saves a full set of choices, then cuts the last two bytes off the file, as a
// disk error, a hand edit or a sync tool can. It answers the bytes left.
func seededThenDamaged(t *testing.T, store *Settings) []byte {
	t.Helper()
	full := ports.Settings{
		LibraryRoot: filepath.Join("D:", "rec"),
		JournalDir:  filepath.Join("D:", "journals"),
		Voice:       "Alice",
		SwitchedOff: []string{"DockingGranted", "FSDJump", "Scanned"},
	}
	if err := store.Save(full); err != nil {
		t.Fatalf("seeding: %v", err)
	}
	raw, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatalf("reading the seed: %v", err)
	}
	damaged := raw[:len(raw)-2]
	if err := os.WriteFile(store.path, damaged, filePerm); err != nil {
		t.Fatalf("damaging: %v", err)
	}
	return damaged
}

// remember is one remembered choice the way the facade makes it: load, change one field, save.
func remember(t *testing.T, store *Settings) {
	t.Helper()
	held := store.Load()
	held.SkippedUpdate = "9.9.9"
	if err := store.Save(held); err != nil {
		t.Fatalf("remembering: %v", err)
	}
}

func TestASettingsFileThatDoesNotParseIsKeptAsideRatherThanSavedOver(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	damaged := seededThenDamaged(t, store)

	remember(t, store)

	aside := store.path + damagedSuffix
	kept, err := os.ReadFile(aside)
	if err != nil {
		t.Fatalf("nothing was kept aside at %s: %v", aside, err)
	}
	if !reflect.DeepEqual(kept, damaged) {
		t.Errorf("kept aside %q, want the damaged file as it was: %q", kept, damaged)
	}
	if problem := store.Problem(); problem == nil || !strings.Contains(problem.Error(), aside) {
		t.Errorf("the store says %v, want it to name where the damaged file was kept", problem)
	}
	if got := store.Load().SkippedUpdate; got != "9.9.9" {
		t.Errorf("the choice made after it reads %q, want it kept", got)
	}
}

// A copy kept aside earlier is the only record of the choices it held, so a second damaged file
// goes beside it rather than over it.
func TestAnEarlierKeptAsideCopyIsNotWrittenOver(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	earlier := []byte("an earlier damaged file")
	seededThenDamaged(t, store)
	if err := os.WriteFile(store.path+damagedSuffix, earlier, filePerm); err != nil {
		t.Fatalf("planting the earlier copy: %v", err)
	}

	remember(t, store)

	if kept, err := os.ReadFile(store.path + damagedSuffix); err != nil || !reflect.DeepEqual(kept, earlier) {
		t.Errorf("the earlier copy reads %q (%v), want it untouched", kept, err)
	}
	if _, err := os.Stat(store.path + damagedSuffix + ".2"); err != nil {
		t.Errorf("the second damaged file was not kept beside the first: %v", err)
	}
}

// A file that reads and parses says nothing is wrong; nor does no file at all.
func TestASoundFileOrNoFileIsNoProblem(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	store.Load()
	if problem := store.Problem(); problem != nil {
		t.Errorf("no file: %v, want no problem", problem)
	}
	remember(t, store)
	store.Load()
	if problem := store.Problem(); problem != nil {
		t.Errorf("a sound file: %v, want no problem", problem)
	}
}
