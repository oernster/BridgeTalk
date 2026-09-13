package main

// FR-238: the window opens whatever the journal directory holds. run hands the window to
// Wails, which no test can open, so these hold what run decides before it does: what
// openJournal watches and what the facade then tells the panes.

import (
	"errors"
	"path/filepath"
	"testing"
)

// standardAt stands in for the game's usual directory, so no test reads the real profile.
func standardAt(dir string, err error) func() (string, error) {
	return func() (string, error) { return dir, err }
}

// unconsulted stands in for the usual directory where a chosen one should win; asking it
// at all fails the test.
func unconsulted(t *testing.T) func() (string, error) {
	t.Helper()
	return func() (string, error) {
		t.Error("the game's usual directory was consulted over a chosen one")
		return "", nil
	}
}

// Every folder a reader can wrongly hold is carried to the window with its reason rather
// than ending the run; the reason names the folder once.
func TestAJournalDirectoryThatCannotBeWatchedIsCarriedToTheWindow(t *testing.T) {
	empty, missing, journalOnly, plainFile := wrongFolders(t)
	for _, dir := range []string{empty, missing, journalOnly, plainFile} {
		name := filepath.Base(dir)
		watched := openJournal(dir, "", unconsulted(t))

		if watched.problem == "" {
			t.Errorf("%s: no reason was carried", name)
			continue
		}
		namedOnce(t, "startup given "+name, errors.New(watched.problem), dir)
		if watched.directory != dir {
			t.Errorf("%s: carried %q, want the directory it looked in", name, watched.directory)
		}
		if watched.sources != nil {
			t.Errorf("%s: sources were built over a directory that cannot be watched", name)
		}
	}
}

func TestAJournalDirectoryThatCanBeWatchedCarriesNoProblem(t *testing.T) {
	dir := journalDirFixture(t)

	watched := openJournal(dir, "", unconsulted(t))

	if watched.problem != "" {
		t.Fatalf("a directory that can be watched carried %q", watched.problem)
	}
	if len(watched.sources) != 2 {
		t.Fatalf("got %d sources, want the journal and the status file", len(watched.sources))
	}
	if watched.statusPath != filepath.Join(dir, "Status.json") {
		t.Fatalf("status path = %q, want the file in %s", watched.statusPath, dir)
	}
}

// The flag outranks what Settings holds, which outranks the game's usual directory.
func TestStartupWatchesTheFlagThenSettingsThenTheUsualDirectory(t *testing.T) {
	flagged, stored, usual := journalDirFixture(t), journalDirFixture(t), journalDirFixture(t)

	if got := openJournal(flagged, stored, unconsulted(t)).directory; got != flagged {
		t.Errorf("with a flag, watched %q, want the flag's %q", got, flagged)
	}
	if got := openJournal("", stored, unconsulted(t)).directory; got != stored {
		t.Errorf("with no flag, watched %q, want the stored %q", got, stored)
	}
	if got := openJournal("", "", standardAt(usual, nil)).directory; got != usual {
		t.Errorf("with nothing chosen, watched %q, want the usual %q", got, usual)
	}
}

// A machine where the game has never run: nothing chosen and no usual directory on disk.
// The window still opens and names where it looked.
func TestAMissingUsualDirectoryIsCarriedToTheWindow(t *testing.T) {
	usual := filepath.Join(t.TempDir(), "Elite Dangerous")

	watched := openJournal("", "", standardAt(usual, nil))

	if watched.problem == "" {
		t.Fatal("no reason was carried for a usual directory that is not there")
	}
	namedOnce(t, "startup with nothing chosen", errors.New(watched.problem), usual)
	if watched.directory != usual {
		t.Errorf("carried %q, want the usual directory %q", watched.directory, usual)
	}
}

// With no home directory there is no usual place to name at all, so the reason alone is
// carried.
func TestNoUsualDirectoryAtAllIsCarriedToTheWindow(t *testing.T) {
	reason := errors.New("resolving home directory: no home")

	watched := openJournal("", "", standardAt("", reason))

	if watched.problem != reason.Error() {
		t.Errorf("carried %q, want %q", watched.problem, reason)
	}
	if watched.directory != "" || watched.sources != nil {
		t.Errorf("carried a directory %q or sources with no usual place to watch", watched.directory)
	}
}

// What the panes read: the reason reaches the state, survives a press that is refused and
// goes once Browse takes a directory that can be watched.
func TestBrowseOverAStartupProblemClearsIt(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.settings = &fakeSettings{}
	missing := filepath.Join(t.TempDir(), "Missing Folder")
	app.watch(openJournal(missing, "", unconsulted(t)))

	state := app.State()
	if state.JournalProblem == "" {
		t.Fatal("the state carries no reason for a directory that cannot be watched")
	}
	if state.JournalDir != missing {
		t.Fatalf("the state names %q, want the directory startup looked in", state.JournalDir)
	}

	answering(app, t.TempDir(), nil)
	if _, err := app.ChooseJournalDir(); err == nil {
		t.Fatal("an empty folder was taken")
	}
	if app.State().JournalProblem == "" {
		t.Fatal("a refused press took the startup reason down, though nothing changed")
	}

	answering(app, journalDirFixture(t), nil)
	if _, err := app.ChooseJournalDir(); err != nil {
		t.Fatalf("choosing: %v", err)
	}
	if got := app.State().JournalProblem; got != "" {
		t.Fatalf("the reason %q outlived a directory that can be watched", got)
	}
}
