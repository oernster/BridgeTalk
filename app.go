// The Wails facade: the surface the front end calls.
//
// It is a client of the application use cases only. It maps their results into DTOs
// the front end can render and holds no logic of its own; every decision it reports
// was made below it. Together with main.go it forms the composition root, which is
// why the structural test whitelists exactly these two files.
package main

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/infrastructure/setup"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// playbackEvent is emitted whenever a sequence starts or ends, carrying whether anything
// is playing. Play returns as soon as the clip starts, so without this the front end
// has no way to know when the sound stopped and can only guess.
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

	// journalProblem says why the journal directory is not being watched; empty while it
	// is. The window opens either way, so the panes are where it is said (FR-238).
	journalProblem string

	// settings keeps the two directory choices between runs. The facade owns the
	// writing because it owns the act that changes them.
	settings ports.SettingsStore

	mu      sync.Mutex
	history []ReactionDTO
	stop    chan struct{}

	// stoppedReacting says why the loop watching the game ended; empty while it runs. The loop's
	// own goroutine writes it and the window's reads it, so it is held under the lock beside the
	// sources for the same reason they are (FR-742).
	stoppedReacting string

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

	// reveal opens a folder in the file manager; a field so a test opens no window.
	reveal func(dir string) error

	// browse hands an address to the desktop's browser (FR-718); a field so a test opens none.
	browse func(address string) error

	// quitting records that a quit has already been decided, so the close dialog is
	// not raised over the top of the quit it was just asked to perform.
	quitting atomic.Bool

	// startedHidden records that the run began put away in the notification area.
	startedHidden bool

	// broughtBack records that the tray has brought the window back. The tray's goroutine sets it;
	// the page's request for the keyboard reads it from the window's.
	broughtBack atomic.Bool
}

// newApp builds the facade over an assembled session.
func newApp(
	current *session,
	watched journalWatch,
	libraryRoot string,
	settings ports.SettingsStore,
) *App {
	built := &App{
		session:     current,
		libraryRoot: libraryRoot,
		settings:    settings,
		stop:        make(chan struct{}),
	}
	built.watch(watched)
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
//
// A run started hidden is not raised by it until the tray has brought the window back (FR-704).
// Its page loads while the window is put away, where finding no keyboard is to be expected; asking
// then would raise the window the sign-in entry put away.
func (a *App) TakeKeyboard() {
	if a.startedHidden && !a.broughtBack.Load() {
		return
	}
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
//
// A run started hidden is left hidden (FR-704): the tray brings it up with the keyboard
// when it is wanted.
func (a *App) domReady(context.Context) {
	if a.startedHidden {
		return
	}
	a.show()
}

// shutdown stops the poll loop and releases the audio device.
func (a *App) shutdown(context.Context) {
	close(a.stop)
	a.session.player.Stop()
	if a.session.tray != nil {
		a.session.tray.Stop()
	}
	a.session.release()
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
		Voice:        a.session.active.Name,
		VoiceDisplay: a.session.active.Display,
		Bound:        bound,
		Total:        total,
		Muted:        a.session.muted,
		Silent:       a.session.player.Silent(),
		JournalDir:   a.journalDir,
		StatusPath:   a.statusPath,
		LibraryRoot:  a.libraryRoot,
		Version:      version,
		// Read at the moment it is asked for rather than held. The entry is one the
		// setup program writes too, so a copy kept here would go stale the first time
		// it was changed from there.
		LaunchOnBoot:    setup.IsLaunchOnBoot(),
		Stalls:          stalls,
		WorstStall:      int(worst.Milliseconds()),
		JournalProblem:  a.journalProblem,
		StoppedReacting: a.reactingStopped(),
		MachineVoice:    a.session.active.Machine,
		Plugin:          a.session.active.Plugin,
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
func (a *App) Volume() float64 { return a.session.player.Volume() }

// SetVolume sets the playback level. The front end owns the stored preference and
// pushes it in on load, so nothing here has to persist it.
func (a *App) SetVolume(level float64) { a.session.player.SetVolume(level) }

// emitState tells the front end to re-read the state it does not own.
func (a *App) emitState() {
	a.emit(stateEvent, a.State())
}
