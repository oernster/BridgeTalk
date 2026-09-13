package main

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/services"
)

// fakePlayer stands in for the output device. Every method records what it was
// asked to do, so a test can assert on the facade's behaviour without a sound card.
type fakePlayer struct {
	mu       sync.Mutex
	played   [][]string
	stops    int
	playing  bool
	silent   bool
	volume   float64
	failWith error
	finished chan struct{}

	stalls     int
	worstStall time.Duration
}

func newFakePlayer() *fakePlayer {
	return &fakePlayer{volume: 1, finished: make(chan struct{}, 1)}
}

func (f *fakePlayer) Play(clips []string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failWith != nil {
		return f.failWith
	}
	f.played = append(f.played, clips)
	f.playing = true
	return nil
}

func (f *fakePlayer) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops++
	f.playing = false
}

func (f *fakePlayer) Playing() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.playing
}

func (f *fakePlayer) Done() <-chan struct{} { return f.finished }

func (f *fakePlayer) Silent() bool { return f.silent }

func (f *fakePlayer) Stalls() (int, time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stalls, f.worstStall
}

func (f *fakePlayer) Volume() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.volume
}

func (f *fakePlayer) SetVolume(level float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.volume = level
}

func (f *fakePlayer) Close() error { return nil }

// finish signals one completed sequence, as the real player does from its own
// goroutine when a clip runs out.
func (f *fakePlayer) finish() { f.finished <- struct{}{} }

// recorder collects what the facade announced to the front end.
type recorder struct {
	mu     sync.Mutex
	events []recorded
	waits  map[string]chan any
}

type recorded struct {
	name    string
	payload any
}

func newRecorder() *recorder {
	return &recorder{waits: make(map[string]chan any)}
}

func (r *recorder) emit(name string, payload any) {
	r.mu.Lock()
	r.events = append(r.events, recorded{name: name, payload: payload})
	waiter := r.waits[name]
	r.mu.Unlock()
	if waiter != nil {
		select {
		case waiter <- payload:
		default:
		}
	}
}

// await returns the next payload emitted under a name, failing the test rather than
// hanging when the facade stays silent.
func (r *recorder) await(t *testing.T, name string) any {
	t.Helper()
	r.mu.Lock()
	waiter, ok := r.waits[name]
	if !ok {
		waiter = make(chan any, 1)
		r.waits[name] = waiter
	}
	r.mu.Unlock()

	select {
	case payload := <-waiter:
		return payload
	case <-time.After(2 * time.Second):
		t.Fatalf("no %q event was emitted", name)
		return nil
	}
}

// newTestApp assembles the smallest facade that can run its loop: a fake device and
// a scheduler over it. Nothing here touches the disk or Wails.
func newTestApp(t *testing.T, player *fakePlayer) (*App, *recorder) {
	t.Helper()
	current := &session{player: player}
	current.scheduler = services.NewScheduler(player, nil, systemClock{})
	app := newApp(current, nil, "journal-dir", "status-file", "library-root", nil)
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
	app := newApp(current, nil, "journal-dir", "status-file", "library-root", nil)

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
	app := newApp(current, nil, "journal-dir", "status-file", "library-root", nil)

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
	app := newApp(current, nil, "journal-dir", "status-file", "library-root", nil)

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
	if !strings.Contains(about.Attribution, "ships no audio") {
		t.Errorf("attribution = %q, want it to state that no audio ships", about.Attribution)
	}
	if len(about.Credits) == 0 {
		t.Error("about carried no open source credits")
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
	app := newApp(current, nil, "journal-dir", "status-file", "library-root", nil)
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
