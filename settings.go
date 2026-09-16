// The settings surface of the facade: where the application reads from, plus changing
// it without restarting.
//
// It sits beside app.go for the reason cast.go and audition.go do. The facade is the
// one file everything hangs off, so it is first to reach the size cap; a pane's own
// questions are a slice that comes out whole.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/infrastructure/library"
	"github.com/oernster/bridge-talk/internal/infrastructure/setup"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ChooseLibraryRoot asks for the directory holding the recordings and takes it.
//
// The choice applies at once rather than at the next launch. A reader who has just
// pointed the application at their voices is answering the question "where are they",
// and a window that then says nothing until it is restarted has not answered it.
//
// The directory that was taken comes back, so the pane can say what happened. An empty
// answer means the dialog was cancelled: not an error and not a change, everything left
// exactly as it was. Returning nothing at all made those two outcomes identical on
// screen, which is what made a refusal read as the button doing nothing.
func (a *App) ChooseLibraryRoot() (string, error) {
	chosen, err := a.chooseDir("Where your recordings are", a.recordingsStart())
	if chosen == "" || err != nil {
		return "", err
	}

	// The scan names the directory itself (FR-237), so it is not named here again.
	found, _, scanErr := library.Scan(chosen, a.session.table)
	if scanErr != nil {
		return "", scanErr
	}
	if len(found) == 0 {
		return "", noVoicesIn(chosen)
	}

	a.libraryRoot = chosen
	a.adopt(found)

	if err := a.rememberLibraryRoot(chosen); err != nil {
		return "", err
	}
	a.emitState()
	return chosen, nil
}

// noVoicesIn refuses a directory that holds no voices, in words the reader can act on.
//
// The usual mistake is a parent of the right place, so the refusal names what a library
// directory looks like rather than only saying that this one is wrong.
//
// %s rather than %q, here and everywhere in this file a path is written: %q escapes
// what it quotes, so a Windows path reached the window with every separator doubled and
// the reader was shown a path that does not exist. It is a free function so that the
// wording can be read by a test; the dialog around it cannot run without a window.
func noVoicesIn(dir string) error {
	return fmt.Errorf(
		"no voices in %s: choose the folder that holds one directory per person, "+
			"each holding their recordings",
		dir,
	)
}

// ChooseJournalDir asks for the game's journal directory and takes it.
//
// Both sources are rebuilt over the new directory and swapped in together, because a
// journal read from one place and a status file from another would describe two
// different sessions. The journal reader starts at the end of the newest file, so a
// change mid-flight speaks about what happens next rather than replaying what already
// has.
//
// It answers the way ChooseLibraryRoot does: the directory that was taken; nothing at all
// where the dialog was cancelled.
func (a *App) ChooseJournalDir() (string, error) {
	chosen, err := a.chooseDir("Where the game writes its journal", a.journalDir)
	if chosen == "" || err != nil {
		return "", err
	}

	watched, err := watchJournal(chosen)
	if err != nil {
		return "", err
	}

	// Nothing is changed until both sources exist. A directory that yields one and
	// not the other would otherwise leave the application half moved; one that yields
	// both takes down whatever startup said was wrong (FR-238).
	a.watch(watched)

	if err := a.rememberJournalDir(chosen); err != nil {
		return "", err
	}
	a.emitState()
	return chosen, nil
}

// SetLaunchOnBoot adds or removes the entry that starts the application at sign-in.
//
// It writes the same per-user login entry the setup program offers, under the same
// name, so the toggle here and the box there are two views of one setting rather than
// two settings that can disagree. Nothing is remembered alongside it: the entry is the
// state; `State` reads it back each time it is asked.
//
// The path written is whichever program is running, so a copy started from anywhere
// registers itself rather than some other installation. The one refusal is a program
// running from the temporary directory, which is what a build run straight from source
// is: registering that would have Windows chase a file that is deleted by the morning.
func (a *App) SetLaunchOnBoot(enabled bool) error {
	running, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding the running program: %w", err)
	}
	if resolved, linkErr := filepath.EvalSymlinks(running); linkErr == nil {
		running = resolved
	}
	if enabled && underTemporaryDirectory(running) {
		return fmt.Errorf(
			"this copy is running from a temporary directory, so it would be gone by "+
				"the next sign-in: install it, then turn this on from the installed copy (%s)",
			running,
		)
	}
	if err := setup.SetLaunchOnBoot(running, enabled); err != nil {
		return fmt.Errorf("writing the login entry: %w", err)
	}
	a.emitState()
	return nil
}

// underTemporaryDirectory reports whether a path sits inside the user's temp tree.
//
// Compared case insensitively because Windows paths are. The prefix carries a
// separator on the end so that a sibling directory whose name merely starts with the
// same letters is not mistaken for a child of it.
func underTemporaryDirectory(path string) bool {
	temporary, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		temporary = os.TempDir()
	}
	prefix := strings.ToLower(filepath.Clean(temporary)) + string(filepath.Separator)
	return strings.HasPrefix(strings.ToLower(filepath.Clean(path)), prefix)
}

// pickDirectory opens the system's own directory chooser.
//
// A chooser rather than a text box: a path typed by hand is wrong more often than it
// is right. The two usual mistakes, a stray quote or a trailing separator, both read
// as a directory that does not exist.
func (a *App) pickDirectory(title, start string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("the window is not ready")
	}
	chosen, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            title,
		DefaultDirectory: start,
	})
	if err != nil {
		return "", fmt.Errorf("choosing a directory: %w", err)
	}
	return chosen, nil
}

// rememberLibraryRoot writes the recordings directory down, so the choice outlives the run.
//
// Each directory has a writer of its own, called only once that directory has been taken.
// A single writer for both kept whatever the other one held at the time: a journal
// directory detected, given on the command line for one run or missing altogether was
// frozen into Settings by choosing the recordings directory (FR-711). Recording a guess keeps
// a profile that later moves pointing at where it used to be.
func (a *App) rememberLibraryRoot(dir string) error {
	return a.keep(func(held *ports.Settings) { held.LibraryRoot = dir })
}

// rememberJournalDir writes the journal directory down, for the reason
// rememberLibraryRoot writes the recordings directory: it alone was chosen.
func (a *App) rememberJournalDir(dir string) error {
	return a.keep(func(held *ports.Settings) { held.JournalDir = dir })
}

// rememberVoice writes the cast voice's name down, so the voice outlives the run.
//
// It is separate from remember for the same reason the detected paths are not written
// at all: only a deliberate choice is recorded. Casting is that choice. The re-cast
// that follows a new library root is not, because it falls back to whatever voice is
// there when the stored name is not, so writing the voice from remember would freeze
// a voice nobody picked and keep honouring it every run afterwards.
//
// Each kind of voice is kept apart from the others, so casting one forgets whichever was kept
// before (FR-540, FR-569).
func (a *App) rememberVoice(name string) error {
	return a.keep(castKept(name, "", "", ""))
}

// rememberMachineVoice writes the cast machine voice's id down, forgetting every other kind, for
// the reason rememberVoice writes a recorded one (FR-540).
func (a *App) rememberMachineVoice(id string) error {
	return a.keep(castKept("", id, "", ""))
}

// rememberPluginVoice writes the cast plugin voice down by the plugin that offered it and the
// voice's id within that plugin, forgetting every other kind (FR-569).
func (a *App) rememberPluginVoice(from, id string) error {
	return a.keep(castKept("", "", from, id))
}

// castKept writes the voice cast, of whichever kind, then forgets the rest.
//
// The four fields are written together in one place because at most one of them ever holds a
// voice: a start casts a kept plugin voice ahead of a kept machine voice ahead of a recorded one,
// so a kind left behind from an earlier cast would speak in place of the one just chosen.
func castKept(voice, machine, from, pluginVoice string) func(*ports.Settings) {
	return func(held *ports.Settings) {
		held.Voice, held.MachineVoice = voice, machine
		held.Plugin, held.PluginVoice = from, pluginVoice
	}
}

// keep reads what is stored, applies a change to it and writes it back.
//
// Read before write rather than write alone: each caller owns part of the file, so a
// whole-file write built from one of them would blank the other's fields. A facade
// with no store is not an error; it is a machine with nowhere to keep settings, where
// everything works and nothing is remembered.
func (a *App) keep(change func(*ports.Settings)) error {
	if a.settings == nil {
		return nil
	}
	held := a.settings.Load()
	change(&held)
	if err := a.settings.Save(held); err != nil {
		return fmt.Errorf("remembering the choice: %w", err)
	}
	return nil
}
