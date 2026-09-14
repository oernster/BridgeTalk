// The reaction log's surface of the facade: every decision the application makes, kept for
// the Status pane and announced to the page as it is made.
//
// It sits beside app.go for the reason cast.go does: the facade is the one file everything else
// hangs off, so it is first to reach the size cap; the log's own record and question are a
// slice that comes out whole.

package main

import "github.com/oernster/bridge-talk/internal/application/ports"

// reactionHistory is how many decisions the home pane keeps. Enough to diagnose a
// cue that never fires without holding a session's worth of chatter in memory.
const reactionHistory = 200

// reactionEvent is the Wails event name the front end subscribes to.
const reactionEvent = "reaction"

// reporter carries the facade's Reporter port without putting it on the wire.
//
// Wails binds every exported method of the object it is given, so an exported method
// is a public one whether or not it was meant to be. Report exists for the application
// layer to hand its decisions back; leaving it on the facade offered the page a way to
// write entries into the reaction log, which is a record of what the application did.
// Holding it on a type of its own means the port is satisfied and nothing is offered.
// The history lives on the facade because it exists purely to be displayed; nothing
// below this line has any use for it.
type reporter struct{ app *App }

// Report records one decision and tells the front end about it.
func (r reporter) Report(reaction ports.Reaction) { r.app.record(reaction) }

// record is the body of that report, kept on the facade because it owns the history
// and the channel to the page.
func (a *App) record(reaction ports.Reaction) {
	line := ReactionDTO{
		At:      reaction.At.Format("15:04:05"),
		Cue:     string(reaction.Cue),
		Title:   a.session.cueEntry(reaction.Cue).Title,
		Clip:    baseName(reaction.Clip),
		Outcome: reaction.Outcome,
	}

	a.mu.Lock()
	a.history = append(a.history, line)
	if len(a.history) > reactionHistory {
		a.history = a.history[len(a.history)-reactionHistory:]
	}
	a.mu.Unlock()

	a.emit(reactionEvent, line)
}

// Reactions returns the decision history, newest last.
func (a *App) Reactions() []ReactionDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]ReactionDTO, len(a.history))
	copy(out, a.history)
	return out
}

// baseName trims a clip path down to its file name for display.
func baseName(path string) string {
	for index := len(path) - 1; index >= 0; index-- {
		if path[index] == '/' || path[index] == '\\' {
			return path[index+1:]
		}
	}
	return path
}
