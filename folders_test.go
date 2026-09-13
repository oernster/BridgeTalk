package main

// Making a voice's folders and looking again, through the facade. The folder rules
// themselves are held in the library package; these hold what the window does around
// them: asking where, keeping the answer and taking what a second look finds.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/library"
)

// FR-223 through the facade: the folders land under the recordings directory already
// chosen, one per moment, with nothing asked.
func TestFoldersAreMadeUnderTheChosenRecordingsDirectory(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.libraryRoot = t.TempDir()
	app.chooseDir = func(string, string) (string, error) {
		t.Fatal("asked where, with a recordings directory already chosen")
		return "", nil
	}

	made, err := app.MakeVoiceFolders("Oliver")
	if err != nil {
		t.Fatalf("making folders: %v", err)
	}
	if made.Path != filepath.Join(app.libraryRoot, "Oliver") {
		t.Fatalf("made %q, want the voice under the chosen directory", made.Path)
	}
	if made.Made != app.session.table.Len() {
		t.Fatalf("made %d folders, want one per moment", made.Made)
	}
}

// FR-226: with no recordings directory there is no other way for a new user to name one,
// since choosing one that holds no voices is refused. So this asks; it keeps the answer.
func TestWithNoRecordingsDirectoryItAsksWhereAndKeepsTheAnswer(t *testing.T) {
	app, _, log := fixtureApp(t)
	store := &fakeSettings{}
	app.settings = store
	app.libraryRoot = ""
	where := t.TempDir()
	answering(app, where, nil)

	made, err := app.MakeVoiceFolders("Oliver")
	if err != nil {
		t.Fatalf("making folders: %v", err)
	}
	if _, err := os.Stat(made.Path); err != nil {
		t.Fatalf("no voice folder at %q: %v", made.Path, err)
	}
	if app.libraryRoot != where || store.held.LibraryRoot != where {
		t.Fatalf("root %q, stored %q; want both %q", app.libraryRoot, store.held.LibraryRoot, where)
	}
	if log.countEmitted(stateEvent) == 0 {
		t.Fatal("the page was not told there is now a recordings directory")
	}
}

// Cancelling the question makes nothing and changes nothing.
func TestCancellingWhereTheFoldersGoChangesNothing(t *testing.T) {
	app, _, log := fixtureApp(t)
	app.libraryRoot = ""
	answering(app, "", nil)

	made, err := app.MakeVoiceFolders("Oliver")
	if err != nil || made.Path != "" {
		t.Fatalf("cancelling answered %v, %v; want nothing at all", made, err)
	}
	if app.libraryRoot != "" || log.countEmitted(stateEvent) != 0 {
		t.Fatal("cancelling changed something")
	}
}

// FR-225: a name that would be refused is refused before any dialog opens.
func TestABadNameIsRefusedBeforeAnythingIsAsked(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.libraryRoot = ""
	app.chooseDir = func(string, string) (string, error) {
		t.Fatal("a dialog opened for a name that is refused anyway")
		return "", nil
	}

	if _, err := app.MakeVoiceFolders("a/b"); !errors.Is(err, library.ErrVoiceName) {
		t.Fatalf("got %v, want the name refused", err)
	}
}

// A failed dialog and a directory the folders cannot go in are both reported.
func TestWhereTheFoldersCannotGoIsReported(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.libraryRoot = ""
	answering(app, "", errors.New("the dialog would not open"))
	if _, err := app.MakeVoiceFolders("Oliver"); err == nil {
		t.Fatal("a failed dialog reported success")
	}

	app.libraryRoot = filepath.Join(t.TempDir(), "gone")
	if _, err := app.MakeVoiceFolders("Oliver"); err == nil {
		t.Fatal("folders were reported made under a directory that is not there")
	}
}

// FR-214: a voice filled since the start appears without a restart.
func TestLookingAgainFindsAVoiceFilledSinceTheStart(t *testing.T) {
	app, _, log := fixtureApp(t)
	before := len(app.Voices())
	writeClip(t, filepath.Join(app.libraryRoot, "Carol", "Docked", "take.wav"))

	found, err := app.Rescan()
	if err != nil {
		t.Fatalf("looking again: %v", err)
	}
	if found != before+1 || len(app.Voices()) != before+1 {
		t.Fatalf("found %d voices, want %d", found, before+1)
	}
	if !app.session.hasVoice() {
		t.Fatal("no voice is cast after looking again")
	}
	if log.countEmitted(stateEvent) == 0 {
		t.Fatal("the page was not told what was found")
	}
}

// Finding nothing leaves the cast voice speaking; no directory to look in says so.
func TestLookingAgainWithNothingToFindChangesNothing(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.libraryRoot = t.TempDir()
	available := len(app.session.available)

	if found, err := app.Rescan(); err != nil || found != 0 {
		t.Fatalf("got %d, %v; want no voices and no error", found, err)
	}
	if len(app.session.available) != available {
		t.Fatal("an empty look threw away the voices already known")
	}

	app.libraryRoot = filepath.Join(app.libraryRoot, "gone")
	if _, err := app.Rescan(); err == nil {
		t.Fatal("a directory that is not there was read")
	}
	app.libraryRoot = ""
	if _, err := app.Rescan(); !errors.Is(err, library.ErrNoRoot) {
		t.Fatalf("got %v, want no recordings directory named", err)
	}
}
