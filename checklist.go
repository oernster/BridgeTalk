// The record surface of the facade: which voice folders exist, what each still has no
// recording for and the way to the folder a take belongs in.
//
// It sits beside app.go for the reason folders.go does: the facade is first to reach the
// size cap; a pane's own questions are a slice that comes out whole.

package main

import (
	"fmt"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/infrastructure/library"
	"github.com/oernster/bridge-talk/internal/infrastructure/window"
)

// VoiceDirectories lists every voice folder under the recordings directory, including
// one that holds nothing yet: the folders the pane chooses among (FR-316). With no
// recordings directory there are none, which is an answer rather than a fault: the pane
// says where to make one (FR-317).
func (a *App) VoiceDirectories() ([]string, error) {
	if a.libraryRoot == "" {
		return []string{}, nil
	}
	return library.VoiceDirs(a.libraryRoot)
}

// Checklist answers with what the named voice folder has no recording for, with how many
// moments it does have one for and where the voice folder is (FR-311, FR-313).
//
// The folder ends in the platform's separator, so the page shows a moment's folder as the
// folder followed by the moment's id without knowing how this platform joins a path.
func (a *App) Checklist(voice string) (ChecklistDTO, error) {
	missing, recorded, err := library.Missing(a.libraryRoot, voice, a.session.table)
	if err != nil {
		return ChecklistDTO{Voice: voice}, err
	}
	return ChecklistDTO{
		Voice:    voice,
		Recorded: recorded,
		Total:    a.session.table.Len(),
		Missing:  cueLines(missing),
		Folder:   filepath.Join(a.libraryRoot, voice) + string(filepath.Separator),
	}, nil
}

// OpenMomentFolder opens the folder a take for one moment belongs in, making it where it
// is missing (FR-314). Nothing is opened for a moment that cannot be reached (FR-315).
func (a *App) OpenMomentFolder(voice, id string) error {
	dir, err := library.MomentFolder(a.libraryRoot, voice, id, a.session.table)
	if err != nil {
		return err
	}
	show := a.reveal
	if show == nil {
		show = window.Reveal
	}
	if err := show(dir); err != nil {
		return fmt.Errorf("opening %s: %w", dir, err)
	}
	return nil
}
