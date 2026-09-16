package main

import (
	"sync"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
)

// fakeTray stands in for the notification area icon, recording what it was last told to show.
type fakeTray struct {
	voice taskbar.Voice
	shown string
	muted bool
}

func (f *fakeTray) Commands() <-chan taskbar.Command { return nil }

func (f *fakeTray) SetMuted(muted bool) { f.muted = muted }

func (f *fakeTray) SetActiveVoice(cast taskbar.Voice, label string) {
	f.voice, f.shown = cast, label
}

func (f *fakeTray) Stop() {}

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

// PlayIfIdle starts only when the fake is not already playing, as the real player does.
func (f *fakePlayer) PlayIfIdle(clips []string, _ time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failWith != nil {
		return false, f.failWith
	}
	if f.playing {
		return false, nil
	}
	f.played = append(f.played, clips)
	f.playing = true
	return true, nil
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
