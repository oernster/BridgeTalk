// The loop that watches the game: the one goroutine the facade owns.
//
// It is here rather than in app.go because it is a concern of its own: a ticker, the tray's
// commands and the player's completions on one select, with the fault handling that keeps a
// failure in any arm from taking the window with it (FR-742). app.go keeps the bound surface.
package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
)

// run polls both sources and services the tray until shutdown.
//
// A fault raised anywhere in the loop ends the loop and nothing else (FR-742). It ran without a
// guard until 2026-09-16, when a nil pointer in one of its arms ended the whole run: the window
// went; the only record of why reached the log nobody had been asked to open.
func (a *App) run() {
	defer a.stopReactingOnFault()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var trayCommands <-chan taskbar.Command
	if a.session.tray != nil {
		trayCommands = a.session.tray.Commands()
	}

	for {
		select {
		case <-a.stop:
			return
		case command, ok := <-trayCommands:
			if !ok {
				trayCommands = nil
				continue
			}
			a.handleTray(command)
		case <-a.session.player.Done():
			// An audition plays with no voice cast (FR-216), when there is no scheduler to tell.
			if a.session.scheduler != nil {
				a.session.scheduler.Finished()
			}
			a.emit(playbackEvent, PlaybackDTO{Playing: a.session.player.Playing()})
		case <-ticker.C:
			a.pollAndAnnounce()
		}
	}
}

// stopReactingOnFault ends the loop with the fault said rather than with the application gone.
//
// Three things happen, in the order a reader needs them. The fault and its stack reach the run
// log, which is where the author looks and the only place a stack is any use. The window is told,
// so the panes say the application has stopped reacting and what to do about it: a recovered
// fault that reaches no surface leaves a window that looks alive and answers nothing, which is
// worse than the application ending. Then the loop stays ended, because running it into the same
// fault four times a second would fill the log with one line repeated and change nothing.
//
// What the window can still do afterwards is the whole point: every pane reads, every dialog
// opens and Quit works, since those are called from the page rather than from here. The tray menu
// is the one thing that goes quiet with the loop, which is why the words on screen say to start
// the application again rather than implying everything is well.
func (a *App) stopReactingOnFault() {
	fault := recover()
	if fault == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "the loop watching the game stopped: %v\n%s\n", fault, debug.Stack())

	a.mu.Lock()
	a.stoppedReacting = fmt.Sprintf("%v", fault)
	a.mu.Unlock()
	a.emitState()
}

// reactingStopped answers why the loop ended, empty while it runs. It reads under the lock
// because the loop's own goroutine is what wrote it.
func (a *App) reactingStopped() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stoppedReacting
}

// poll asks each source for what is new and hands it to the reaction service.
//
// The sources are taken under the lock and polled outside it. Choosing a journal
// directory replaces the whole slice from the window's own goroutine, so reading it
// directly here would be a race with a poll already under way; polling inside the lock
// would instead hold it across every file read.
func (a *App) poll() {
	a.mu.Lock()
	sources := a.sources
	a.mu.Unlock()

	for _, source := range sources {
		events, err := source.Poll()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", source.Name(), err)
			continue
		}
		if len(events) > 0 && a.session.hasVoice() {
			a.session.reactions.HandleAll(events)
		}
	}
	if a.session.scheduler != nil {
		a.session.scheduler.Advance()
	}
}

// handleTray acts on one tray-menu choice.
func (a *App) handleTray(command taskbar.Command) {
	switch command.Kind {
	case taskbar.CommandQuit:
		// Through the same door as every other quit, so the intent is recorded and
		// the close dialog does not appear over a quit chosen from the tray.
		a.Quit()
	case taskbar.CommandShow:
		a.broughtBack.Store(true)
		a.restore()
		a.emit(windowShownEvent, nil)
	case taskbar.CommandToggleMute:
		a.SetMuted(!a.session.muted)
	case taskbar.CommandSelectVoice:
		a.castFromTray(command.Chosen)
	case taskbar.CommandNoTray:
		// The desktop draws no tray after all, so the cross closes rather than hiding into
		// nothing; a window started hidden for a tray that never came is shown (FR-814).
		a.trayGone.Store(true)
		if a.startedHidden {
			a.broughtBack.Store(true)
			a.restore()
			a.emit(windowShownEvent, nil)
		}
	}
}

// castFromTray casts what the tray menu chose, by the kind the choice carries.
//
// A refusal goes no further than here. The tray has nowhere to say one: it has an icon, a tooltip
// and a menu that has already closed, so a refusal put there would be a message with no surface.
// The window is where a cast is refused in words; every refusal reachable from the menu is also
// reachable there, where it is said (FR-519, FR-570).
func (a *App) castFromTray(chosen taskbar.Voice) {
	switch chosen.Kind {
	case taskbar.Machine:
		_ = a.CastMachineVoice(chosen.Name)
	case taskbar.Plugin:
		_ = a.CastPluginVoice(chosen.Plugin, chosen.Name)
	case taskbar.Recorded:
		_ = a.SelectVoice(chosen.Name)
	}
}
