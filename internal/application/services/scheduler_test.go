package services_test

import (
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
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
	return services.Request{Cue: built, Clips: clips}
}

func TestAlertInterruptsWhatIsSpeaking(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, &collector{}, frozenClock{})

	scheduler.Submit(request(t, "ambient", "ambient", "quiet.mp3"))
	scheduler.Advance()
	if !player.playing {
		t.Fatal("the ambient cue should be playing")
	}

	scheduler.Submit(request(t, "alert", "alert", "danger.mp3"))
	if player.stops != 1 {
		t.Fatalf("stops = %d, an alert must interrupt", player.stops)
	}
	scheduler.Advance()
	last := player.played[len(player.played)-1]
	if last[0] != "danger.mp3" {
		t.Fatalf("played %v, the alert should have taken over", last)
	}
}

func TestAlertDoesNotInterruptAnotherAlert(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, &collector{}, frozenClock{})

	scheduler.Submit(request(t, "first", "alert", "one.mp3"))
	scheduler.Advance()
	scheduler.Submit(request(t, "second", "alert", "two.mp3"))

	if player.stops != 0 {
		t.Fatalf("stops = %d, one alert must not cut off another", player.stops)
	}
}

// A take cut short by an alert reports its end once the alert is already under way, as the
// player does. That report must not end the alert: a flavour line arriving then is let go
// (FR-612) rather than queued behind it. Reproduced on a real device before this was written.
func TestATakeCutShortLeavesTheAlertThatReplacedItSpeaking(t *testing.T) {
	player := newFakePlayer()
	log := &collector{}
	scheduler := services.NewScheduler(player, log, frozenClock{})

	scheduler.Submit(request(t, "ambient", "ambient", "quiet.mp3"))
	scheduler.Advance()
	scheduler.Submit(request(t, "alert", "alert", "danger.mp3"))
	scheduler.Advance()
	scheduler.Finished()

	if !scheduler.Speaking() {
		t.Fatal("the end of the cut take was taken for the end of the alert")
	}
	if scheduler.Submit(request(t, "flavour", "flavour", "idle.mp3")) {
		t.Fatalf("a flavour line was taken while the alert was speaking; pending = %d", scheduler.Pending())
	}
	if log.outcomes[len(log.outcomes)-1] != ports.OutcomeDropped {
		t.Errorf("outcomes = %v, want the flavour line recorded as dropped", log.outcomes)
	}
}

// Alerts waiting together are said in the order they happened, ahead of anything less
// urgent (Oliver, 2026-09-13).
func TestAlertsWaitingTogetherPlayInArrivalOrder(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, &collector{}, frozenClock{})

	scheduler.Submit(request(t, "notice", "notice", "notice.mp3"))
	scheduler.Submit(request(t, "first", "alert", "one.mp3"))
	scheduler.Submit(request(t, "second", "alert", "two.mp3"))
	scheduler.Advance()

	for index, want := range []string{"one.mp3", "two.mp3", "notice.mp3"} {
		if len(player.played) != index+1 || player.played[index][0] != want {
			t.Fatalf("played %v, want %s next", player.played, want)
		}
		player.finish()
		scheduler.Finished()
	}
}

// Submit says whether it took a request, which is what decides whether the firing counts
// against the cooldown.
func TestSubmitSaysWhetherItTookTheRequest(t *testing.T) {
	scheduler := services.NewScheduler(newFakePlayer(), &collector{}, frozenClock{})

	if !scheduler.Submit(request(t, "notice", "notice", "notice.mp3")) {
		t.Fatal("a notice was not taken")
	}
	if scheduler.Submit(request(t, "ambient", "ambient", "quiet.mp3")) {
		t.Fatal("an ambient cue let go behind a waiting notice was reported as taken")
	}
}

func TestFlavourIsDroppedWhileAnythingIsPending(t *testing.T) {
	log := &collector{}
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, log, frozenClock{})

	scheduler.Submit(request(t, "notice", "notice", "notice.mp3"))
	scheduler.Advance()
	scheduler.Submit(request(t, "flavour", "flavour", "idle.mp3"))

	for _, clips := range player.played {
		if clips[0] == "idle.mp3" {
			t.Fatal("a flavour cue played while something was speaking")
		}
	}
	if log.outcomes[len(log.outcomes)-1] != ports.OutcomeDropped {
		t.Fatalf("outcomes = %v, the flavour cue should be recorded as dropped", log.outcomes)
	}
}

func TestAmbientQueuesOnlyWhenNothingIsWaiting(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, &collector{}, frozenClock{})

	scheduler.Submit(request(t, "notice", "notice", "a.mp3"))
	scheduler.Submit(request(t, "ambient", "ambient", "b.mp3"))

	if scheduler.Pending() != 1 {
		t.Fatalf("pending = %d, the ambient cue should have been dropped", scheduler.Pending())
	}
}

func TestQueueIsOrderedByPriorityThenArrival(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, &collector{}, frozenClock{})

	scheduler.Submit(request(t, "n1", "notice", "n1.mp3"))
	scheduler.Submit(request(t, "n2", "notice", "n2.mp3"))
	scheduler.Submit(request(t, "a1", "alert", "a1.mp3"))

	scheduler.Advance()
	player.finish()
	scheduler.Finished()
	player.finish()
	scheduler.Finished()

	order := []string{}
	for _, clips := range player.played {
		order = append(order, clips[0])
	}
	want := []string{"a1.mp3", "n1.mp3", "n2.mp3"}
	for index := range want {
		if index >= len(order) || order[index] != want[index] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

func TestEveryCuePlaysExactlyOneFile(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, &collector{}, frozenClock{})

	scheduler.Submit(request(t, "take", "notice", "one.mp3", "two.mp3", "three.mp3"))
	scheduler.Advance()
	if len(player.played[0]) != 1 {
		t.Fatalf("a take played %d clips, want 1", len(player.played[0]))
	}
	if player.gaps[0] != 0 {
		t.Fatalf("a take used a gap of %v, want none", player.gaps[0])
	}

	player.finish()
	scheduler.Finished()

	// Several takes are alternatives, never the parts of one utterance: one cue plays
	// one file, so a request carrying three speaks the first and no more.
	scheduler.Submit(request(t, "alternatives", "notice", "1.mp3", "2.mp3", "3.mp3"))
	player.finish()
	scheduler.Advance()
	last := player.played[len(player.played)-1]
	if len(last) != 1 {
		t.Fatalf("a cue with three takes played %d clips, want 1", len(last))
	}
	if player.gaps[len(player.gaps)-1] != 0 {
		t.Fatal("a single take needs no gap")
	}
}

// A flavour line is the lowest priority there is: it exists only for a quiet moment,
// so anything at all already speaking or waiting discards it.
func TestAFlavourLineIsDiscardedWhileSomethingIsSpeaking(t *testing.T) {
	player := newFakePlayer()
	log := &collector{}
	scheduler := services.NewScheduler(player, log, frozenClock{})

	scheduler.Submit(request(t, "notice", "notice", "one.mp3"))
	scheduler.Advance()

	scheduler.Submit(request(t, "flavour", "flavour", "two.mp3"))

	if len(player.played) != 1 {
		t.Fatalf("played %v, want the flavour line discarded", player.played)
	}
	if log.outcomes[len(log.outcomes)-1] != ports.OutcomeDropped {
		t.Errorf("outcomes = %v, want the flavour line recorded as dropped", log.outcomes)
	}
}

// A device that refuses the clip is silence by another name, so it is recorded the
// same way and nothing is left marked as speaking; otherwise the scheduler would
// wait forever for a sequence that never started.
func TestAPlaybackFailureIsRecordedAndLeavesNothingSpeaking(t *testing.T) {
	player := newFakePlayer()
	player.fail = true
	log := &collector{}
	scheduler := services.NewScheduler(player, log, frozenClock{})

	scheduler.Submit(request(t, "notice", "notice", "one.mp3"))
	scheduler.Advance()

	if scheduler.Speaking() {
		t.Error("the scheduler is speaking after the device refused the clip")
	}
	if log.outcomes[len(log.outcomes)-1] != ports.OutcomeUnbound {
		t.Errorf("outcomes = %v, want the refusal recorded", log.outcomes)
	}
}

func TestSpeakingFollowsWhatIsActuallyPlaying(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, &collector{}, frozenClock{})

	if scheduler.Speaking() {
		t.Fatal("a new scheduler reported that it is speaking")
	}

	scheduler.Submit(request(t, "notice", "notice", "one.mp3"))
	scheduler.Advance()

	if !scheduler.Speaking() {
		t.Error("nothing was reported as speaking while a clip was playing")
	}

	player.finish()
	scheduler.Finished()

	if scheduler.Speaking() {
		t.Error("the scheduler still reported speaking after the clip finished")
	}
}

// The command-line modes build the path without a reaction log, so every decision
// has to survive there being nobody to tell.
func TestDecisionsAreMadeWithNoReporterWiredUp(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, nil, frozenClock{})

	scheduler.Submit(request(t, "notice", "notice", "one.mp3"))
	scheduler.Advance()
	scheduler.Submit(request(t, "flavour", "flavour", "two.mp3"))

	if len(player.played) != 1 {
		t.Errorf("played %v, want the one playable request", player.played)
	}
}

// Ambient chatter is the voice talking to itself. It waits only when nothing else is,
// so anything already queued discards it rather than lengthening the wait.
func TestAmbientChatterIsDiscardedWhenSomethingIsAlreadyQueued(t *testing.T) {
	player := newFakePlayer()
	log := &collector{}
	scheduler := services.NewScheduler(player, log, frozenClock{})

	scheduler.Submit(request(t, "notice", "notice", "one.mp3"))
	scheduler.Submit(request(t, "ambient", "ambient", "two.mp3"))

	if scheduler.Pending() != 1 {
		t.Fatalf("pending = %d, want the ambient line discarded rather than queued", scheduler.Pending())
	}
	if log.outcomes[len(log.outcomes)-1] != ports.OutcomeDropped {
		t.Errorf("outcomes = %v, want the ambient line recorded as dropped", log.outcomes)
	}
}

// The quiet moment a flavour line exists for: nothing speaking and nothing waiting,
// so it is taken rather than discarded.
func TestAFlavourLineIsTakenWhenNothingElseIsPending(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, &collector{}, frozenClock{})

	scheduler.Submit(request(t, "flavour", "flavour", "one.mp3"))
	scheduler.Advance()

	if len(player.played) != 1 || player.played[0][0] != "one.mp3" {
		t.Errorf("played %v, want the flavour line taken in the quiet", player.played)
	}
}

// A request speaks once, however many clips it carries. The number of clips
// available is not a reason to speak for longer.
func TestAnUnorderedRequestSpeaksOnceHoweverManyClipsItCarries(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, &collector{}, frozenClock{})

	folder := request(t, "Undocked", "notice", "0.mp3",
		"1.mp3", "2.mp3", "3.mp3", "4.mp3", "5.mp3", "6.mp3", "7.mp3", "8.mp3", "9.mp3", "10.mp3")

	scheduler.Submit(folder)
	scheduler.Advance()

	if len(player.played) != 1 {
		t.Fatalf("played %v, want a single utterance", player.played)
	}
	if len(player.played[0]) != 1 {
		t.Fatalf("spoke %d clips for one event, want 1: a folder is not a performance",
			len(player.played[0]))
	}
}
