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
	"github.com/oernster/bridge-talk/internal/refusal"
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

// PluginChecklist answers what the cast plugin voice has no take for, with how many moments it
// does have one for and no folder at all (FR-571).
//
// It is read off the catalogue rather than off a directory, because a plugin voice has no
// directory this application owns: where its audio lives is the plugin's business and may not be
// written to (FR-572). The empty folder is what says so on the wire: the pane offers no way to
// open or create one for a list that carries none, rather than keeping a rule of its own about
// which kinds of voice have folders.
//
// Only the cast voice has a catalogue, so only the cast plugin voice has a list here. Any other
// state answers an empty list rather than an error: the pane behind it is a thing to read, so it
// has nothing to report.
func (a *App) PluginChecklist() ChecklistDTO {
	if a.session.active.Plugin == "" || a.session.catalogue == nil {
		return ChecklistDTO{Missing: []CueDTO{}}
	}
	recorded, total := a.session.catalogue.Coverage()
	return ChecklistDTO{
		Voice:    a.session.active.Display,
		Recorded: recorded,
		Total:    total,
		Missing:  cueLines(a.session.catalogue.Unbound()),
	}
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
		return fmt.Errorf("opening %s: %w", dir, refusal.Reason(err))
	}
	return nil
}
