package main

// The checklist through the facade. The folder rules are held in the library package;
// these hold what the window asks for and that nothing is shown for a moment that
// cannot be reached. No test here opens a real window: fixtureApp's reveal does nothing.

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/library"
)

// FR-316: a folder with nothing in it yet is among the folders the pane chooses from.
func TestEveryVoiceFolderIsOfferedRecordedOrNot(t *testing.T) {
	app, _, _ := fixtureApp(t)
	if err := os.MkdirAll(filepath.Join(app.libraryRoot, "Dora"), 0o755); err != nil {
		t.Fatalf("making Dora: %v", err)
	}

	got, err := app.VoiceDirectories()
	if err != nil || !reflect.DeepEqual(got, []string{"Alpha", "Beta", "Bystander", "Dora"}) {
		t.Fatalf("got %v, %v; want every folder under the root", got, err)
	}

	app.libraryRoot = ""
	if got, err := app.VoiceDirectories(); err != nil || len(got) != 0 {
		t.Fatalf("with no root got %v, %v; want none and no error", got, err)
	}
}

// FR-311 and FR-313: Alpha holds three of the fixture's four moments.
func TestTheChecklistCountsWhatIsRecorded(t *testing.T) {
	app, _, _ := fixtureApp(t)

	list, err := app.Checklist("Alpha")
	if err != nil {
		t.Fatalf("listing Alpha: %v", err)
	}
	if list.Recorded != 3 || list.Total != app.session.table.Len() {
		t.Fatalf("recorded %d of %d, want 3 of %d", list.Recorded, list.Total, app.session.table.Len())
	}
	if len(list.Missing) != 1 || list.Missing[0].ID != "zz.silent" {
		t.Fatalf("missing %v, want the one moment nobody records", list.Missing)
	}
	// FR-229: the page is sent the moment's folder name with its dot written as an underscore.
	if list.Missing[0].Folder != "zz_silent" {
		t.Fatalf("folder %q, want zz_silent", list.Missing[0].Folder)
	}
	// FR-311: the voice folder ends in the separator, so folder plus id is a moment's folder.
	wantFolder := filepath.Join(app.libraryRoot, "Alpha") + string(filepath.Separator)
	if list.Folder != wantFolder {
		t.Fatalf("folder %q, want %q", list.Folder, wantFolder)
	}
	if _, err := app.Checklist("Nobody"); err == nil {
		t.Fatal("a voice folder that is not there was listed")
	}
}

// FR-314: the folder is made where missing and that folder is the one shown.
func TestOpeningAMomentsFolderMakesItAndShowsIt(t *testing.T) {
	app, _, _ := fixtureApp(t)
	if err := os.MkdirAll(filepath.Join(app.libraryRoot, "Dora"), 0o755); err != nil {
		t.Fatalf("making Dora: %v", err)
	}
	var shown string
	app.reveal = func(dir string) error {
		shown = dir
		return nil
	}

	if err := app.OpenMomentFolder("Dora", "Docked"); err != nil {
		t.Fatalf("opening: %v", err)
	}
	want := filepath.Join(app.libraryRoot, "Dora", "Docked")
	if shown != want {
		t.Fatalf("showed %q, want %q", shown, want)
	}
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Fatalf("the folder shown does not exist: %v", err)
	}
}

// FR-315: nothing is shown for an unknown moment; a refusal to show is carried.
func TestAMomentFolderThatCannotBeOpenedIsReported(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.reveal = func(string) error {
		t.Fatal("a folder was shown for a moment that does not exist")
		return nil
	}
	if err := app.OpenMomentFolder("Alpha", "Nope"); !errors.Is(err, library.ErrUnknownCue) {
		t.Fatalf("got %v, want the moment refused", err)
	}

	refused := errors.New("the shell said no")
	app.reveal = func(string) error { return refused }
	if err := app.OpenMomentFolder("Alpha", "Docked"); !errors.Is(err, refused) {
		t.Fatalf("got %v, want the refusal carried", err)
	}
}
