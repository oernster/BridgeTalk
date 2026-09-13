// The Wails facade: the surface the front end calls.
//
// It is a client of the application use cases only. It maps their results into DTOs
// the front end can render and holds no logic of its own; every decision it reports
// was made below it. Together with main.go it forms the composition root, which is
// why the structural test whitelists exactly these two files.
package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/infrastructure/setup"
	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// reactionHistory is how many decisions the home pane keeps. Enough to diagnose a
// cue that never fires without holding a session's worth of chatter in memory.
const reactionHistory = 200

// reactionEvent is the Wails event name the front end subscribes to.
const reactionEvent = "reaction"

// playbackEvent is emitted whenever a sequence ends, carrying whether anything is
// still playing. Play returns as soon as the clip starts, so without this the front
// end has no way to know when the sound stopped and can only guess.
const playbackEvent = "playback"

// windowShownEvent is emitted when the window is summoned back from the notification
// area, so the page can open where a summoned window should open rather than wherever
// it happened to be left. Hiding a window does not reload the page, so without this it
// comes back on whichever pane was last used, which is usually the one being fiddled
// with rather than the one worth seeing.
const windowShownEvent = "window-shown"

// stateEvent is emitted whenever the voice or mute state changes, so the front end
// re-reads rather than tracking state the Go side already owns.
const stateEvent = "state"

// App is the bound facade.
type App struct {
	ctx     context.Context
	session *session
	sources []ports.EventSource

	journalDir  string
	statusPath  string
	libraryRoot string

	// settings keeps the two directory choices between runs. The facade owns the
	// writing because it owns the act that changes them.
	settings ports.SettingsStore

	mu      sync.Mutex
	history []ReactionDTO
	stop    chan struct{}

	// emit sends one event to the front end. It is a field rather than a direct
	// call so a test can read what the facade announced; production wiring points
	// it at Wails in newApp.
	emit func(name string, payload any)

	// show brings the window to the front and gives it the keyboard. A field for the
	// same reason emit is one.
	show func()

	// quit ends the application. A field for the same reason the other two are.
	quit func()

	// hide puts the window away without ending the run, for the close dialog's
	// Minimise. A field for the same reason the other three are.
	hide func()

	// restore brings the window back from the notification area, centred. A field
	// for the same reason the other four are.
	restore func()

	// chooseDir opens the system's directory chooser. A field for the same reason
	// the other five are: the dialog cannot run without a window, so what the two
	// choose-a-directory methods do with an answer is otherwise unreachable.
	chooseDir func(title, start string) (string, error)

	// quitting records that a quit has already been decided, so the close dialog is
	// not raised over the top of the quit it was just asked to perform.
	quitting atomic.Bool
}

// newApp builds the facade over an assembled session.
func newApp(
	current *session,
	sources []ports.EventSource,
	journalDir, statusPath, libraryRoot string,
	settings ports.SettingsStore,
) *App {
	built := &App{
		session:     current,
		sources:     sources,
		journalDir:  journalDir,
		statusPath:  statusPath,
		libraryRoot: libraryRoot,
		settings:    settings,
		stop:        make(chan struct{}),
	}
	built.emit = built.emitToWails
	built.show = built.showInWails
	built.quit = built.quitWails
	built.hide = built.hideInWails
	built.restore = built.restoreInWails
	built.chooseDir = built.pickDirectory
	return built
}

// emitToWails is the production emitter. Before startup there is no context to emit
// into, so an event raised then is dropped rather than panicking.
func (a *App) emitToWails(name string, payload any) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, payload)
}

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
		Event:   reaction.Event,
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

// startup begins the poll loop once Wails has a context.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.run()
}

// TakeKeyboard is called by the page when it finds it has no keyboard.
//
// The page is the only thing that can tell: from Go the window looks focused either
// way. document.hasFocus() being false there means every key is going somewhere else,
// which reads as a keyboard that does nothing at all.
func (a *App) TakeKeyboard() {
	a.show()
}

// domReady takes the keyboard once the page exists.
//
// The window is already on screen by now. The tray is built before it and runs a
// window and a message pump of its own, so the application can come up with the
// foreground somewhere else. Asking for it here is what makes the webview hold the
// keyboard: Wails focuses it from the main window's own focus event, which a window
// that never took focus never raises. Without this the first Tab goes to whatever
// does hold the keyboard and the ring is never reached, which reads as a dead
// keyboard rather than as a focus that landed elsewhere.
func (a *App) domReady(context.Context) {
	a.show()
}

// shutdown stops the poll loop and releases the audio device.
func (a *App) shutdown(context.Context) {
	close(a.stop)
	a.session.player.Stop()
	if a.session.tray != nil {
		a.session.tray.Stop()
	}
}

// run polls both sources and services the tray until shutdown.
func (a *App) run() {
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
			a.session.scheduler.Finished()
			a.emit(playbackEvent, PlaybackDTO{Playing: a.session.player.Playing()})
		case <-ticker.C:
			a.poll()
		}
	}
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
		a.restore()
		a.emit(windowShownEvent, nil)
	case taskbar.CommandToggleMute:
		a.SetMuted(!a.session.muted)
	case taskbar.CommandSelectVoice:
		_ = a.SelectVoice(command.Voice)
	}
}

// State returns everything the header and home pane need.
func (a *App) State() StateDTO {
	// With no voice cast there is no catalogue to ask. The honest answer is that
	// nothing is covered rather than that nothing exists to cover.
	bound, total := 0, a.session.table.Len()
	if a.session.catalogue != nil {
		bound, total = a.session.catalogue.Coverage()
	}
	stalls, worst := a.session.player.Stalls()
	return StateDTO{
		Voice:       a.session.active.Name,
		Bound:       bound,
		Total:       total,
		Muted:       a.session.muted,
		Silent:      a.session.player.Silent(),
		JournalDir:  a.journalDir,
		StatusPath:  a.statusPath,
		LibraryRoot: a.libraryRoot,
		Version:     version,
		// Read at the moment it is asked for rather than held. The entry is one the
		// setup program writes too, so a copy kept here would go stale the first time
		// it was changed from there.
		LaunchOnBoot: setup.IsLaunchOnBoot(),
		Stalls:       stalls,
		WorstStall:   int(worst.Milliseconds()),
	}
}

// Muted reports whether playback is silenced.
func (a *App) Muted() bool { return a.session.muted }

// SetMuted silences or unsilences playback.
func (a *App) SetMuted(muted bool) {
	a.session.muted = muted
	if a.session.hasVoice() {
		a.session.reactions.SetMuted(muted)
	}
	if a.session.tray != nil {
		a.session.tray.SetMuted(muted)
	}
	if muted {
		a.session.player.Stop()
	}
	a.emitState()
}

// Volume reports the playback level, where zero is silence and one is the clip as
// recorded.
func (a *App) Volume() float64 {
	if a.session.player == nil {
		return 0
	}
	return a.session.player.Volume()
}

// SetVolume sets the playback level. The front end owns the stored preference and
// pushes it in on load, so nothing here has to persist it.
func (a *App) SetVolume(level float64) {
	if a.session.player != nil {
		a.session.player.SetVolume(level)
	}
}

// Reactions returns the decision history, newest last.
func (a *App) Reactions() []ReactionDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]ReactionDTO, len(a.history))
	copy(out, a.history)
	return out
}

// About returns the identity and dependency credits for the About dialog.
func (a *App) About() AboutDTO {
	return AboutDTO{
		Name:        appTitle,
		Tagline:     appTagline,
		Version:     version,
		Author:      appAuthor,
		Copyright:   appCopyright,
		Authorship:  appAuthorship,
		Attribution: appAttribution,
		Licence:     appLicence,
		Credits:     credits(),
	}
}

// Licence returns the full terms for the dialog under Help: the LICENSE file itself.
func (a *App) Licence() string { return licenceText }

// emitState tells the front end to re-read the state it does not own.
func (a *App) emitState() {
	a.emit(stateEvent, a.State())
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
