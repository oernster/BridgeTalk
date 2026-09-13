// The folders surface of the facade: making a voice's folders, then looking again at
// what the recordings directory holds.
//
// It sits beside app.go for the reason cast.go and settings.go do: the facade is first
// to reach the size cap; a pane's own questions are a slice that comes out whole.

package main

import (
	"fmt"

	"github.com/oernster/bridge-talk/internal/infrastructure/library"
)

// MakeVoiceFolders makes a folder for the named voice holding one folder per moment.
//
// Where no recordings directory is chosen yet it asks where the folders should go and
// keeps that directory as the recordings directory. Nothing else could set one for a
// new user: ChooseLibraryRoot refuses a directory holding no voices, which is exactly
// what a new user has.
//
// The name is checked before anything is asked, so a name that would be refused never
// opens a dialog first. An empty path in the answer means the dialog was cancelled:
// nothing was made and nothing changed.
func (a *App) MakeVoiceFolders(name string) (VoiceFoldersDTO, error) {
	if err := library.CheckVoiceName(name); err != nil {
		return VoiceFoldersDTO{}, err
	}

	root := a.libraryRoot
	asked := root == ""
	if asked {
		chosen, err := a.chooseDir("Where your recordings will live", a.recordingsStart())
		if chosen == "" || err != nil {
			return VoiceFoldersDTO{}, err
		}
		root = chosen
	}

	dir, made, err := library.MakeVoiceFolders(root, name, a.session.table)
	if err != nil {
		return VoiceFoldersDTO{}, err
	}
	answer := VoiceFoldersDTO{Path: dir, Made: made}
	if !asked {
		return answer, nil
	}

	a.libraryRoot = root
	a.emitState()
	return answer, a.remember()
}

// recordingsStart is where a question about the recordings directory opens (FR-227).
//
// The directory already chosen where there is one; otherwise the product's own
// recordings directory. Given no folder at all, the system dialog chooses for itself
// and it opened in the game's folder, the one place a voice's recordings do not belong.
// A default that cannot be worked out or made leaves the choice to the system rather
// than refusing to ask at all.
func (a *App) recordingsStart() string {
	if a.libraryRoot != "" {
		return a.libraryRoot
	}
	start, err := library.DefaultRoot()
	if err != nil {
		return ""
	}
	return start
}

// Rescan reads the recordings directory again and takes what it finds, answering with
// how many voices that is (FR-214).
//
// A voice filled by hand appears only once its folder holds a recording, which happens
// outside the application, so without this the only way to see it was a restart.
// Finding nothing is an answer rather than an error: the voice that was cast stays cast
// rather than the window being left with no voice for a directory that is mid-copy.
func (a *App) Rescan() (int, error) {
	if a.libraryRoot == "" {
		return 0, fmt.Errorf("%w yet: choose one in Settings or make a voice's folders here", library.ErrNoRoot)
	}
	found, _, err := library.Scan(a.libraryRoot, a.session.table)
	if err != nil {
		return 0, fmt.Errorf("reading %s: %w", a.libraryRoot, err)
	}
	a.adopt(found)
	a.emitState()
	return len(found), nil
}

// adopt takes a freshly scanned list of voices as the ones available.
//
// The voice that was speaking may not survive the scan, so the voice is re-cast by name
// where it does and falls back to the first one where it does not. Leaving a catalogue
// pointing at a directory that has gone would fail silently at the next cue rather than
// here, where it can be said. An empty list changes nothing.
func (a *App) adopt(found []library.Voice) {
	if len(found) == 0 {
		return
	}
	a.session.available = found
	next, err := pick(found, a.session.active.Name)
	if err != nil {
		next = found[0]
	}
	a.session.useVoice(next)
}
