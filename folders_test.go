package main

// Making a voice's folders and looking again, through the facade. The folder rules
// themselves are held in the library package; these hold what the window does around
// them: using the product's own recordings directory where none is chosen, keeping it
// and taking what a second look finds.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/library"
)

// dataHome points the product's data folder at a temporary directory for one test and
// answers with the default recordings directory that results. Any test that can reach
// the default calls it, so the default is made there rather than on the machine running
// the test.
func dataHome(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	t.Setenv("XDG_DATA_HOME", base)
	start, err := library.DefaultRoot()
	if err != nil {
		t.Fatalf("working out the default recordings directory: %v", err)
	}
	return start
}

// neverAsked fails the test if a directory dialog opens, since making folders asks
// nothing.
func neverAsked(t *testing.T, app *App) {
	t.Helper()
	app.chooseDir = func(string, string) (string, error) {
		t.Fatal("a dialog opened for Make folders")
		return "", nil
	}
}

// FR-223 through the facade: the folders land under the recordings directory already
// chosen, one per moment, with nothing asked.
func TestFoldersAreMadeUnderTheChosenRecordingsDirectory(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.libraryRoot = t.TempDir()
	neverAsked(t, app)

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

// FR-228: with no recordings directory the folders go in the product's own, which is then
// kept. No dialog opens.
func TestWithNoRecordingsDirectoryTheFoldersGoInTheDefaultOne(t *testing.T) {
	app, _, log := fixtureApp(t)
	store := &fakeSettings{}
	app.settings = store
	app.libraryRoot = ""
	want := dataHome(t)
	neverAsked(t, app)

	made, err := app.MakeVoiceFolders("Oliver")
	if err != nil {
		t.Fatalf("making folders: %v", err)
	}
	if made.Path != filepath.Join(want, "Oliver") || made.Made != app.session.table.Len() {
		t.Fatalf("made %d under %q, want one per moment under %q", made.Made, made.Path, want)
	}
	if info, err := os.Stat(made.Path); err != nil || !info.IsDir() {
		t.Fatalf("no voice folder at %q: %v", made.Path, err)
	}
	if app.libraryRoot != want || store.held.LibraryRoot != want {
		t.Fatalf("root %q, stored %q; want both %q", app.libraryRoot, store.held.LibraryRoot, want)
	}
	if log.countEmitted(stateEvent) == 0 {
		t.Fatal("the page was not told there is now a recordings directory")
	}
}

// FR-228: a default that cannot be made is reported, makes nothing and changes nothing.
func TestADefaultThatCannotBeMadeIsReported(t *testing.T) {
	app, _, log := fixtureApp(t)
	app.libraryRoot = ""
	file := filepath.Join(t.TempDir(), "plain")
	writeClip(t, file)
	t.Setenv("LOCALAPPDATA", file)
	t.Setenv("XDG_DATA_HOME", file)
	neverAsked(t, app)

	made, err := app.MakeVoiceFolders("Oliver")
	if err == nil || made.Path != "" {
		t.Fatalf("got %v, %v; want the reason and nothing made", made, err)
	}
	if app.libraryRoot != "" || log.countEmitted(stateEvent) != 0 {
		t.Fatal("a default that could not be made changed something")
	}
}

// FR-227: with nothing chosen, the question of where the recordings are opens in the
// product's own recordings folder, which exists by the time it does, rather than
// wherever the system dialog was last pointed.
func TestTheRecordingsQuestionOpensInTheProductsOwnFolder(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.libraryRoot = ""
	want := dataHome(t)
	var starts []string
	app.chooseDir = func(_ string, start string) (string, error) {
		starts = append(starts, start)
		return "", nil
	}

	if _, err := app.ChooseLibraryRoot(); err != nil {
		t.Fatalf("choosing: %v", err)
	}
	if len(starts) != 1 || starts[0] != want {
		t.Fatalf("the question opened at %q, want %q", starts, want)
	}
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Fatalf("the folder the question opened in does not exist: %v", err)
	}
}

// Once a recordings directory is chosen, the question opens where the recordings are.
func TestTheRecordingsQuestionOpensWhereTheRecordingsAre(t *testing.T) {
	app, _, _ := fixtureApp(t)
	var start string
	app.chooseDir = func(_ string, given string) (string, error) {
		start = given
		return "", nil
	}

	if _, err := app.ChooseLibraryRoot(); err != nil {
		t.Fatalf("choosing: %v", err)
	}
	if start != app.libraryRoot {
		t.Fatalf("the question opened at %q, want the chosen %q", start, app.libraryRoot)
	}
}

// A default that cannot be made leaves the question's opening folder to the system rather
// than refusing to ask.
func TestADefaultThatCannotBeMadeStillAsks(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.libraryRoot = ""
	file := filepath.Join(t.TempDir(), "plain")
	writeClip(t, file)
	t.Setenv("LOCALAPPDATA", file)
	t.Setenv("XDG_DATA_HOME", file)
	asked, start := false, "unset"
	app.chooseDir = func(_ string, given string) (string, error) {
		asked, start = true, given
		return "", nil
	}

	if _, err := app.ChooseLibraryRoot(); err != nil {
		t.Fatalf("choosing: %v", err)
	}
	if !asked || start != "" {
		t.Fatalf("asked %v at %q; want the question asked with no folder given", asked, start)
	}
}

// FR-225: a name that would be refused is refused before anything is made.
func TestABadNameIsRefusedBeforeAnythingIsMade(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.libraryRoot = ""
	want := dataHome(t)
	neverAsked(t, app)

	if _, err := app.MakeVoiceFolders("a/b"); !errors.Is(err, library.ErrVoiceName) {
		t.Fatalf("got %v, want the name refused", err)
	}
	if entries, err := os.ReadDir(want); err != nil || len(entries) != 0 {
		t.Fatalf("the recordings directory holds %v, %v; want nothing made", entries, err)
	}
	if app.libraryRoot != "" {
		t.Fatal("a refused name set a recordings directory")
	}
}

// A recordings directory the folders cannot go in is reported.
func TestWhereTheFoldersCannotGoIsReported(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.libraryRoot = filepath.Join(t.TempDir(), "gone")
	neverAsked(t, app)

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
