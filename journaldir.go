// The journal directory: what startup watches and what Browse on the Settings pane puts in
// its place.
//
// Both reach the two sources through watchJournal, so the rule that a directory is watched
// only when both can be built is written once. Startup differs in one way: a directory that
// cannot be watched is carried to the window rather than returned, so the window opens and
// says what is wrong instead of the run ending before there is anywhere to say it (FR-238).

package main

import (
	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/infrastructure/journal"
	"github.com/oernster/bridge-talk/internal/infrastructure/status"
)

// journalWatch is what watching one journal directory amounts to: the directory, the
// status file inside it and the two sources reading them. Where those could not be
// built, it holds why not instead.
type journalWatch struct {
	directory  string
	statusPath string
	sources    []ports.EventSource
	problem    string
}

// watchJournal builds both sources over a directory.
//
// Each source names the directory itself (FR-237), so it is not named here again.
func watchJournal(directory string) (journalWatch, error) {
	journalSource, err := journal.NewSource(directory, systemClock{}.Now)
	if err != nil {
		return journalWatch{}, err
	}
	statusSource, err := status.NewWatcher(directory, systemClock{}.Now)
	if err != nil {
		return journalWatch{}, err
	}
	return journalWatch{
		directory:  directory,
		statusPath: statusSource.Path(),
		sources:    []ports.EventSource{journalSource, statusSource},
	}, nil
}

// openJournal decides what startup watches: the flag, then what Settings holds, then the
// game's usual directory.
//
// It has no error to return, which is the point of it. A directory that cannot be watched
// comes back holding the reason and no sources, so the poll loop has nothing to ask; the
// directory is kept so the panes can name where it looked.
func openJournal(flagged, stored string, standard func() (string, error)) journalWatch {
	directory := preferred(flagged, stored)
	if directory == "" {
		found, err := standard()
		if err != nil {
			return journalWatch{problem: err.Error()}
		}
		directory = found
	}
	watched, err := watchJournal(directory)
	if err != nil {
		return journalWatch{directory: directory, problem: err.Error()}
	}
	return watched
}

// watch puts a journal watch in place: the sources under the lock the poll loop reads them
// by, the rest beside them. A watch that works carries no problem, so taking one clears
// whatever startup said was wrong.
func (a *App) watch(watched journalWatch) {
	a.mu.Lock()
	a.sources = watched.sources
	a.mu.Unlock()

	a.journalDir = watched.directory
	a.statusPath = watched.statusPath
	a.journalProblem = watched.problem
}
