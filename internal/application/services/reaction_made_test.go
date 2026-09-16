package services_test

// FR-514: a cue that fires with no line made for it waits for that line to be made, then speaks; a
// line still unwritten after 2 seconds lets the cue go.

import (
	"slices"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/domain/take"
)

// madeOnCall is FR-514's limit on how long a cue waits for its line.
const madeOnCall = 2 * time.Second

// fakeCueMaker records the cues it is asked to make lines for, answering each with answer.
type fakeCueMaker struct {
	asked  []cue.ID
	answer bool
}

func (f *fakeCueMaker) MakeNext(id cue.ID) bool {
	f.asked = append(f.asked, id)
	return f.answer
}

// waitingWiring assembles the decision path over a moving clock with a cue maker that answers yes,
// for one notice cue, Docked, that nothing has been made for.
func waitingWiring(t *testing.T) (
	*services.ReactionService, *fakeCatalogue, *fakePlayer, *collector, *movingClock, *fakeCueMaker,
) {
	t.Helper()
	player, log := newFakePlayer(), &collector{}
	moving := &movingClock{now: moment}
	catalogue := newFakeCatalogue()
	item := cueFor(t, cue.Definition{ID: "Docked", Source: "journal", Event: "Docked", Priority: "notice"})
	service := services.NewReactionService(
		cue.NewTable([]cue.Cue{item}), catalogue, services.NewScheduler(player, log, moving),
		firstChooser{}, log, moving,
	)
	maker := &fakeCueMaker{answer: true}
	service.SetCueMaker(maker)
	return service, catalogue, player, log, moving, maker
}

// docked is one firing of Docked at the clock's time.
func docked(clock *movingClock) []event.Event {
	return []event.Event{journalEvent("Docked", clock.now)}
}

func TestACueWithNothingMadeWaitsForItsLineThenSpeaks(t *testing.T) {
	service, catalogue, player, log, clock, maker := waitingWiring(t)

	service.HandleAll(docked(clock))
	service.Tick()

	if !slices.Equal(log.outcomes, []string{ports.OutcomeMaking}) || !slices.Equal(maker.asked, []cue.ID{"Docked"}) {
		t.Fatalf("outcomes %v, asked %v; want making recorded and Docked's line asked for", log.outcomes, maker.asked)
	}
	if len(player.played) != 0 {
		t.Fatalf("played %v before the line was written", player.played)
	}

	catalogue.hold("Docked", "docked.flac")
	clock.now = clock.now.Add(madeOnCall / 2)
	service.Tick()
	service.Tick()

	if want := []take.Take{take.Of("docked.flac")}; !slices.EqualFunc(player.played, want, slices.Equal) {
		t.Fatalf("played %v once the line was written, want %v once", player.played, want)
	}

	// The cue fired as the line was written, so a firing inside the repeat window from then is a
	// duplicate.
	clock.now = clock.now.Add(time.Millisecond)
	service.HandleAll(docked(clock))
	if last := log.outcomes[len(log.outcomes)-1]; last != ports.OutcomeDuplicate {
		t.Errorf("outcomes %v, want a firing just after the line was spoken recorded as duplicate", log.outcomes)
	}
}

func TestALineStillUnwrittenAfterTheLimitLetsTheCueGo(t *testing.T) {
	service, catalogue, player, log, clock, _ := waitingWiring(t)
	fired := clock.now

	service.HandleAll(docked(clock))
	clock.now = fired.Add(madeOnCall)
	service.Tick()
	if slices.Contains(log.outcomes, ports.OutcomeDropped) {
		t.Fatalf("outcomes %v, want the cue still waiting at the limit", log.outcomes)
	}
	clock.now = fired.Add(madeOnCall + time.Millisecond)
	service.Tick()
	catalogue.hold("Docked", "docked.flac")
	service.Tick()

	if want := []string{ports.OutcomeMaking, ports.OutcomeDropped}; !slices.Equal(log.outcomes, want) {
		t.Errorf("outcomes %v, want %v", log.outcomes, want)
	}
	if len(player.played) != 0 {
		t.Errorf("played %v for a cue let go", player.played)
	}
}

func TestAFurtherFiringWhileACueWaitsIsADuplicate(t *testing.T) {
	service, _, _, log, clock, maker := waitingWiring(t)

	service.HandleAll(docked(clock))
	clock.now = clock.now.Add(madeOnCall / 2)
	service.HandleAll(docked(clock))

	if want := []string{ports.OutcomeMaking, ports.OutcomeDuplicate}; !slices.Equal(log.outcomes, want) {
		t.Errorf("outcomes %v, want %v", log.outcomes, want)
	}
	if len(maker.asked) != 1 {
		t.Errorf("asked for lines %v, want once", maker.asked)
	}
}

// FR-611 with FR-514: muted, the cue is dropped at once while its line is still made; one muted while
// it waits is dropped when its line arrives.
func TestAMutedCueIsDroppedWhileItsLineIsStillMade(t *testing.T) {
	service, catalogue, player, log, clock, maker := waitingWiring(t)

	service.SetMuted(true)
	service.HandleAll(docked(clock))
	if want := []string{ports.OutcomeDropped}; !slices.Equal(log.outcomes, want) || len(maker.asked) != 1 {
		t.Fatalf("outcomes %v, asked %v; want dropped with the line still asked for", log.outcomes, maker.asked)
	}

	service.SetMuted(false)
	clock.now = clock.now.Add(time.Hour)
	service.HandleAll(docked(clock))
	service.SetMuted(true)
	catalogue.hold("Docked", "docked.flac")
	service.Tick()

	if want := []string{ports.OutcomeDropped, ports.OutcomeMaking, ports.OutcomeDropped}; !slices.Equal(log.outcomes, want) {
		t.Errorf("outcomes %v, want %v", log.outcomes, want)
	}
	if len(player.played) != 0 {
		t.Errorf("played %v while muted", player.played)
	}
}

// A cue no line can be made for is unbound, as it is for a voice with no maker at all.
func TestACueNoLineIsOnItsWayForIsUnbound(t *testing.T) {
	service, _, _, log, clock, maker := waitingWiring(t)
	maker.answer = false

	service.HandleAll(docked(clock))

	if want := []string{ports.OutcomeUnbound}; !slices.Equal(log.outcomes, want) {
		t.Errorf("outcomes %v, want %v", log.outcomes, want)
	}
}
