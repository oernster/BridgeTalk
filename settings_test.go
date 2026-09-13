package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
	"github.com/oernster/bridge-talk/internal/infrastructure/journal"
	"github.com/oernster/bridge-talk/internal/infrastructure/status"
)

// answering makes the directory chooser return a fixed answer, standing in for the
// system dialog, which cannot run without a window.
func answering(app *App, chosen string, err error) {
	app.chooseDir = func(string, string) (string, error) { return chosen, err }
}

// journalDirFixture builds a directory holding a journal and a status file, which is
// what the game's saved-games directory looks like.
func journalDirFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeJournal(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "Status.json"),
		[]byte(`{"Flags":0,"Flags2":0,"GuiFocus":0,"FireGroup":0,"Pips":[4,4,4]}`),
		0o644); err != nil {
		t.Fatalf("writing the status file: %v", err)
	}
	return dir
}

// A reader who has just pointed the application at their voices is answering the
// question "where are they". A window that says nothing until it is restarted has not
// answered it, so the choice applies at once.
func TestChoosingALibraryRootTakesItAtOnce(t *testing.T) {
	app, _, log := fixtureApp(t)
	app.settings = &fakeSettings{}
	elsewhere := libraryRootFixture(t)
	answering(app, elsewhere, nil)

	taken, err := app.ChooseLibraryRoot()
	if err != nil {
		t.Fatalf("choosing: %v", err)
	}
	if taken != elsewhere {
		t.Fatalf("took %q, want %q", taken, elsewhere)
	}
	if app.libraryRoot != elsewhere {
		t.Fatalf("the facade still reads from %q", app.libraryRoot)
	}
	if !app.session.hasVoice() {
		t.Fatal("no voice was cast from the new root")
	}
	if log.countEmitted(stateEvent) == 0 {
		t.Fatal("the page was not told the root changed")
	}
}

// Cancelling is not an error and not a change. Returning nothing at all made those two
// outcomes identical on screen, which is what made a refusal read as a dead button.
func TestCancellingTheChooserChangesNothing(t *testing.T) {
	app, _, log := fixtureApp(t)
	before := app.libraryRoot
	answering(app, "", nil)

	taken, err := app.ChooseLibraryRoot()
	if err != nil {
		t.Fatalf("cancelling reported an error: %v", err)
	}
	if taken != "" {
		t.Fatalf("cancelling took %q", taken)
	}
	if app.libraryRoot != before {
		t.Fatalf("cancelling moved the root to %q", app.libraryRoot)
	}
	if log.countEmitted(stateEvent) != 0 {
		t.Fatal("cancelling announced a change")
	}
}

func TestAChooserThatFailsIsReported(t *testing.T) {
	app, _, _ := fixtureApp(t)
	answering(app, "", errors.New("the dialog would not open"))

	if _, err := app.ChooseLibraryRoot(); err == nil {
		t.Fatal("a failed chooser reported success")
	}
	answering(app, "", errors.New("the dialog would not open"))
	if _, err := app.ChooseJournalDir(); err == nil {
		t.Fatal("a failed chooser reported success")
	}
}

// The usual mistake is a parent of the right place, so the refusal names what a voice directory looks
// like rather than only saying that this one is wrong.
func TestADirectoryHoldingNoVoicesIsRefusedInWordsTheReaderCanActOn(t *testing.T) {
	app, _, _ := fixtureApp(t)
	empty := t.TempDir()
	answering(app, empty, nil)

	_, err := app.ChooseLibraryRoot()
	if err == nil {
		t.Fatal("a directory holding no voices was taken")
	}
	if !strings.Contains(err.Error(), empty) {
		t.Errorf("refusal = %q, want it to name the directory", err)
	}
	if app.libraryRoot == empty {
		t.Fatal("the refused directory was taken anyway")
	}
}

func TestADirectoryThatCannotBeReadIsReported(t *testing.T) {
	app, _, _ := fixtureApp(t)
	answering(app, filepath.Join(t.TempDir(), "nowhere"), nil)

	if _, err := app.ChooseLibraryRoot(); err == nil {
		t.Fatal("a directory that is not there was taken")
	}
}

// The voice that was speaking may not exist under the new root. Leaving a catalogue
// pointing into the old directory would fail silently at the next cue rather than
// here, where it can be said.
func TestTheSpeakingVoiceIsRecastUnderANewRoot(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.settings = &fakeSettings{}
	if err := app.SelectVoice("Beta"); err != nil {
		t.Fatalf("casting Beta: %v", err)
	}

	// A root holding a voice by the same name keeps the voice.
	answering(app, libraryRootFixture(t), nil)
	if _, err := app.ChooseLibraryRoot(); err != nil {
		t.Fatalf("choosing a root holding the same voice: %v", err)
	}
	if app.session.active.Name != "Beta" {
		t.Fatalf("active voice is %q, want Beta kept by name", app.session.active.Name)
	}

	// A root holding nothing by that name falls back to the first voice there is.
	other := t.TempDir()
	audiotest.WriteTake(t, filepath.Join(other, "Zeta", "ShieldState_ShieldsUp_false", "z.mp3"))
	answering(app, other, nil)
	if _, err := app.ChooseLibraryRoot(); err != nil {
		t.Fatalf("choosing a root holding a different voice: %v", err)
	}
	if app.session.active.Name != "Zeta" {
		t.Fatalf("active voice is %q, want the fallback Zeta", app.session.active.Name)
	}
}

// Both sources are rebuilt over the new directory and swapped in together: a journal
// read from one place and a status file from another would describe two sessions.
func TestChoosingAJournalDirectoryRebuildsBothSourcesTogether(t *testing.T) {
	app, _, log := fixtureApp(t)
	app.settings = &fakeSettings{}
	dir := journalDirFixture(t)
	answering(app, dir, nil)

	taken, err := app.ChooseJournalDir()
	if err != nil {
		t.Fatalf("choosing: %v", err)
	}
	if taken != dir {
		t.Fatalf("took %q, want %q", taken, dir)
	}
	if app.journalDir != dir {
		t.Fatalf("the facade still reads from %q", app.journalDir)
	}
	if app.statusPath != filepath.Join(dir, "Status.json") {
		t.Fatalf("status path = %q, want the file in the new directory", app.statusPath)
	}

	app.mu.Lock()
	sources := app.sources
	app.mu.Unlock()
	if len(sources) != 2 {
		t.Fatalf("got %d sources, want the journal and the status file", len(sources))
	}
	if log.countEmitted(stateEvent) == 0 {
		t.Fatal("the page was not told the directory changed")
	}
}

// Nothing is changed until both sources exist. A directory that yields one and not the
// other would otherwise leave the application half moved.
func TestADirectoryYieldingOnlyOneSourceChangesNothing(t *testing.T) {
	t.Run("no journal", func(t *testing.T) {
		app, _, _ := fixtureApp(t)
		bare := t.TempDir()
		answering(app, bare, nil)

		if _, err := app.ChooseJournalDir(); err == nil {
			t.Fatal("a directory with no journal was taken")
		}
		if app.journalDir != "journal-dir" {
			t.Fatalf("the directory moved to %q", app.journalDir)
		}
	})

	t.Run("a journal but no status file", func(t *testing.T) {
		app, _, _ := fixtureApp(t)
		half := t.TempDir()
		writeJournal(t, half)
		answering(app, half, nil)

		if _, err := app.ChooseJournalDir(); err == nil {
			t.Fatal("a directory with no status file was taken")
		}
		if app.journalDir != "journal-dir" {
			t.Fatalf("the directory moved to %q", app.journalDir)
		}
		app.mu.Lock()
		defer app.mu.Unlock()
		if app.sources != nil {
			t.Fatal("the sources were swapped for a directory that was refused")
		}
	})
}

// The two real sources satisfy the port they are swapped in as. Building them here is
// what proves the facade's own wiring, rather than a fake standing in for both.
func TestTheRebuiltSourcesAreTheRealOnes(t *testing.T) {
	dir := journalDirFixture(t)

	source, err := journal.NewSource(dir, systemClock{}.Now)
	if err != nil {
		t.Fatalf("building the journal source: %v", err)
	}
	watcher, err := status.NewWatcher(dir, systemClock{}.Now)
	if err != nil {
		t.Fatalf("building the status watcher: %v", err)
	}
	for _, each := range []interface {
		Name() string
		Poll() ([]event.Event, error)
	}{source, watcher} {
		if each.Name() == "" {
			t.Fatal("a source with no name reached the facade")
		}
		if _, err := each.Poll(); err != nil {
			t.Fatalf("%s: %v", each.Name(), err)
		}
	}
}

// The temporary directory is normally resolvable. Where it is not, the raw path stands
// in rather than the check silently reporting that nothing is temporary.
func TestAnUnresolvableTemporaryDirectoryFallsBackToItsRawPath(t *testing.T) {
	nowhere := filepath.Join(t.TempDir(), "not", "a", "real", "place")
	t.Setenv("TMP", nowhere)
	t.Setenv("TEMP", nowhere)
	t.Setenv("TMPDIR", nowhere)

	if !underTemporaryDirectory(filepath.Join(nowhere, "app.exe")) {
		t.Fatal("a child of the unresolvable temporary directory was not recognised")
	}
}

func TestTheClockReportsTheTimeItIsAsked(t *testing.T) {
	before := time.Now()
	got := systemClock{}.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Fatalf("the clock read %v, outside the window %v to %v", got, before, after)
	}
}
