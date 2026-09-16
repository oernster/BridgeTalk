package services_test

// What the service tests pretend with: the player, the clock and the reaction log they run
// against, plus the two builders every file here uses to state takes. They live in one file
// so a fake has one home rather than one per suite.

import (
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/take"
)

// fakePlayer records what it was asked to do. It is hand-written rather than
// generated, so the test states exactly the behaviour it depends on.
type fakePlayer struct {
	played  [][]string
	gaps    []time.Duration
	stops   int
	playing bool
	fail    bool
	done    chan struct{}
}

func newFakePlayer() *fakePlayer {
	return &fakePlayer{done: make(chan struct{}, 1)}
}

func (f *fakePlayer) Play(clips []string, gap time.Duration) error {
	if f.fail {
		return ports.ErrPlaybackFailed
	}
	f.played = append(f.played, clips)
	f.gaps = append(f.gaps, gap)
	f.playing = true
	return nil
}

func (f *fakePlayer) Stop()                 { f.stops++; f.playing = false }
func (f *fakePlayer) Playing() bool         { return f.playing }
func (f *fakePlayer) Done() <-chan struct{} { return f.done }
func (f *fakePlayer) Close() error          { return nil }
func (f *fakePlayer) finish()               { f.playing = false }

// frozenClock never moves, so nothing in these tests depends on real time.
type frozenClock struct{}

func (frozenClock) Now() time.Time { return time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC) }

// collector captures the reaction log.
type collector struct{ outcomes []string }

func (c *collector) Report(reaction ports.Reaction) {
	c.outcomes = append(c.outcomes, reaction.Outcome)
}

func request(t *testing.T, id string, priority string, clips ...string) services.Request {
	t.Helper()
	built, err := cue.New(cue.Definition{
		ID: id, Source: "journal", Event: "X", Priority: priority,
	})
	if err != nil {
		t.Fatalf("building cue: %v", err)
	}
	return services.Request{Cue: built, Take: take.Of(clips...)}
}

// holds reports whether any take among them is the single part named, which is how a test
// asks whether one made line answers a cue without caring which take carries it.
func holds(takes []take.Take, path string) bool {
	for _, each := range takes {
		if each.Key() == path {
			return true
		}
	}
	return false
}

// takesOf builds one take per path, the shape a voice recording one file per take answers.
func takesOf(paths ...string) []take.Take {
	out := make([]take.Take, 0, len(paths))
	for _, path := range paths {
		out = append(out, take.Of(path))
	}
	return out
}
