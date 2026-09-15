package main

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
)

// newTestApp assembles the smallest facade that can run its loop: a fake device and
// a scheduler over it. Nothing here touches the disk or Wails.
func newTestApp(t *testing.T, player *fakePlayer) (*App, *recorder) {
	t.Helper()
	current := &session{player: player}
	current.scheduler = services.NewScheduler(player, nil, systemClock{})
	fixtureMaking(t, current, offeredFiles(nil), makingtest.NewStore())
	app := newApp(current, fixtureWatch, "library-root", nil)
	log := newRecorder()
	app.emit = log.emit
	return app, log
}

// The player returns from Play as soon as the clip starts, so the only way the front
// end can know the sound has stopped is to be told. The pulse on an audition button
// is drawn from this event.
func TestTheEndOfASequenceIsAnnouncedToTheFrontEnd(t *testing.T) {
	player := newFakePlayer()
	app, log := newTestApp(t, player)

	go app.run()
	defer close(app.stop)

	player.finish()

	payload := log.await(t, playbackEvent)
	state, ok := payload.(PlaybackDTO)
	if !ok {
		t.Fatalf("playback payload was %T, want PlaybackDTO", payload)
	}
	if state.Playing {
		t.Error("the sequence ended with nothing else playing, yet playing was reported true")
	}
}

// A sequence that ends because a newer one replaced it must not tell the front end
// that everything has gone quiet: something is audibly playing.
func TestASupersededSequenceReportsThatSomethingIsStillPlaying(t *testing.T) {
	player := newFakePlayer()
	app, log := newTestApp(t, player)

	go app.run()
	defer close(app.stop)

	player.mu.Lock()
	player.playing = true
	player.mu.Unlock()
	player.finish()

	payload := log.await(t, playbackEvent)
	state, ok := payload.(PlaybackDTO)
	if !ok {
		t.Fatalf("playback payload was %T, want PlaybackDTO", payload)
	}
	if !state.Playing {
		t.Error("a replaced sequence reported silence while the replacement was playing")
	}
}

// Before Wails calls startup there is no context to emit into. An event raised then
// is dropped; the alternative is a nil-context panic on a path nothing can retry.
func TestEmittingBeforeStartupIsDropped(t *testing.T) {
	player := newFakePlayer()
	current := &session{player: player}
	current.scheduler = services.NewScheduler(player, nil, systemClock{})
	app := newApp(current, fixtureWatch, "library-root", nil)

	app.emit(playbackEvent, PlaybackDTO{Playing: false})
}

// The window has to hold the keyboard before the first key press. Without it the
// ring is never reached and the keyboard reads as dead.
func TestTheWindowIsRaisedOnceThePageExists(t *testing.T) {
	player := newFakePlayer()
	app, _ := newTestApp(t, player)
	raised := 0
	app.show = func() { raised++ }

	app.domReady(context.Background())

	if raised != 1 {
		t.Errorf("the window was raised %d times when the page became ready, want once", raised)
	}
}

// FR-704: a run started hidden, as the sign-in entry starts it, stays hidden when its page
// loads (Oliver, 2026-09-13).
func TestAWindowStartedHiddenIsNotRaisedWhenThePageLoads(t *testing.T) {
	player := newFakePlayer()
	app, _ := newTestApp(t, player)
	app.startedHidden = true
	raised := 0
	app.show = func() { raised++ }

	app.domReady(context.Background())

	if raised != 0 {
		t.Errorf("the window was raised %d times, want it left in the notification area", raised)
	}
}

// FR-704: the page of a run started hidden finds it has no keyboard and asks for it. The window
// stays put away until the tray brings it back; only then does the page's request raise it.
func TestAPageAskingForTheKeyboardDoesNotRaiseAWindowStartedHidden(t *testing.T) {
	player := newFakePlayer()
	app, _ := newTestApp(t, player)
	app.startedHidden = true
	raised := 0
	app.show = func() { raised++ }
	app.restore = func() {}

	app.TakeKeyboard()
	if raised != 0 {
		t.Fatalf("the window was raised %d times while put away, want none", raised)
	}
	app.handleTray(taskbar.Command{Kind: taskbar.CommandShow})
	app.TakeKeyboard()
	if raised != 1 {
		t.Errorf("the window was raised %d times once brought back, want once", raised)
	}
}

// The page asks for the keyboard when it finds every press going somewhere else.
func TestThePageCanAskForTheKeyboard(t *testing.T) {
	player := newFakePlayer()
	app, _ := newTestApp(t, player)
	raised := 0
	app.show = func() { raised++ }

	app.TakeKeyboard()

	if raised != 1 {
		t.Errorf("the window was raised %d times when the page asked, want once", raised)
	}
}

// Before Wails calls startup there is no context to raise into. The request is
// dropped, because the alternative is a nil-context panic on a path nothing retries.
func TestRaisingBeforeStartupIsDropped(t *testing.T) {
	player := newFakePlayer()
	current := &session{player: player}
	current.scheduler = services.NewScheduler(player, nil, systemClock{})
	app := newApp(current, fixtureWatch, "library-root", nil)

	app.show()
}

// Quit is the File menu's only item, so it has to actually end the application
// rather than closing a window that is the application.
func TestQuittingEndsTheApplication(t *testing.T) {
	player := newFakePlayer()
	app, _ := newTestApp(t, player)
	quits := 0
	app.quit = func() { quits++ }

	app.Quit()

	if quits != 1 {
		t.Errorf("quitting asked to end the application %d times, want once", quits)
	}
}

// Before Wails calls startup there is no run loop to end.
func TestQuittingBeforeStartupIsDropped(t *testing.T) {
	player := newFakePlayer()
	current := &session{player: player}
	current.scheduler = services.NewScheduler(player, nil, systemClock{})
	app := newApp(current, fixtureWatch, "library-root", nil)

	app.quit()
}

// The About dialog is the only place the application says whose work it is and whose
// voices it speaks with, so both statements have to reach it.
func TestAboutCarriesAuthorshipAndAttribution(t *testing.T) {
	player := newFakePlayer()
	app, _ := newTestApp(t, player)

	about := app.About()

	if !strings.Contains(about.Authorship, "Oliver Ernster") {
		t.Errorf("authorship = %q, want it to name the author", about.Authorship)
	}
	if !strings.Contains(about.Attribution, "ships no recordings") || !strings.Contains(about.Attribution, "Kokoro-82M") {
		t.Errorf("attribution = %q, want it to state that no recordings ship and name the model the machine voices speak with", about.Attribution)
	}
	// The machine voices ship the model and the runtime that runs it, so both are credited
	// with their licences (FR-712).
	for _, shipped := range [][2]string{{"ONNX Runtime", "MIT"}, {"Kokoro-82M", "Apache-2.0"}} {
		if !slices.ContainsFunc(about.Credits, func(credit string) bool {
			return strings.Contains(credit, shipped[0]) && strings.Contains(credit, shipped[1])
		}) {
			t.Errorf("credits = %q, want one naming %s under %s", about.Credits, shipped[0], shipped[1])
		}
	}
}

// A user who owns no voices at all still gets a window. Everything the page asks for on
// the way up has to answer without a voice behind it, so this makes every one of those
// first calls over a session that never had one cast.
//
// Proved by removing any one of the guards it covers and watching it panic. Before
// them the application did not reach this state at all: it printed to a stream nobody
// sees and exited, so the reader saw no window and no reason for its absence.
func TestTheFacadeAnswersWithNoVoiceCast(t *testing.T) {
	player := newFakePlayer()
	current := &session{player: player}
	app := newApp(current, fixtureWatch, "library-root", nil)
	app.emit = newRecorder().emit

	state := app.State()
	if state.Voice != "" {
		t.Errorf("Voice = %q, want empty with nothing cast", state.Voice)
	}
	if state.Bound != 0 {
		t.Errorf("Bound = %d, want 0 with nothing cast", state.Bound)
	}
	if state.LibraryRoot != "library-root" {
		t.Errorf("LibraryRoot = %q, want the directory that was searched", state.LibraryRoot)
	}

	if app.Muted() {
		t.Error("Muted() = true, want false before anything is muted")
	}
	app.SetMuted(true)
	if !app.Muted() {
		t.Error("Muted() = false after SetMuted(true): the answer has to survive having no voice")
	}

	if voices := app.Voices(); len(voices) != 0 {
		t.Errorf("Voices() returned %d, want none", len(voices))
	}
	app.poll()
	app.StopAudition()
}

// TestARefusalNamesThePathAsItIsWritten guards the wording the window shows.
//
// It was %q once, which escapes what it quotes: the refusal reached the pane with
// every separator doubled, so the reader was shown a path that does not exist. Nothing
// else catches that, since the string is only ever read by a human.
func TestARefusalNamesThePathAsItIsWritten(t *testing.T) {
	chosen := `C:\Users\Someone\Music\Recordings`
	said := noVoicesIn(chosen).Error()

	if !strings.Contains(said, chosen) {
		t.Errorf("refusal = %s, want the path exactly as it was chosen", said)
	}
	if strings.Contains(said, `\\`) {
		t.Errorf("refusal = %s, want single separators: a doubled path is not a place", said)
	}
	if !strings.Contains(said, "one directory per person") {
		t.Errorf("refusal = %s, want it to name what a voice directory looks like", said)
	}
}
