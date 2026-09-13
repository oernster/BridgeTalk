package services_test

import (
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
)

// fakeCatalogue answers for whatever voice is pretended to be active.
type fakeCatalogue struct {
	clips  map[cue.ID][]string
	active string
	served int
	total  int
}

func (f *fakeCatalogue) Clips(id cue.ID) (ports.Performance, bool) {
	found, ok := f.clips[id]
	return ports.Performance{Clips: found}, ok
}

func (f *fakeCatalogue) ActiveVoice() string         { return f.active }
func (f *fakeCatalogue) Coverage() (int, int)        { return f.served, f.total }
func (f *fakeCatalogue) hold(id cue.ID, c ...string) { f.clips[id] = c }

func newFakeCatalogue() *fakeCatalogue {
	return &fakeCatalogue{
		clips:  make(map[cue.ID][]string),
		active: "Grace",
		served: 2,
		total:  3,
	}
}

// firstChooser always takes the first candidate, so a pick is deterministic.
type firstChooser struct{}

func (firstChooser) Intn(int) int { return 0 }

// movingClock is a clock a test can wind forward.
type movingClock struct{ now time.Time }

func (m *movingClock) Now() time.Time { return m.now }

var moment = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

func cueFor(t *testing.T, definition cue.Definition) cue.Cue {
	t.Helper()
	built, err := cue.New(definition)
	if err != nil {
		t.Fatalf("building cue %s: %v", definition.ID, err)
	}
	return built
}

// wiring assembles the whole decision path over fakes, which is how every test here
// starts: the service is only meaningful with a table, a catalogue and a scheduler.
func wiring(t *testing.T, cues ...cue.Cue) (
	*services.ReactionService, *fakeCatalogue, *fakePlayer, *collector,
) {
	t.Helper()
	player := newFakePlayer()
	log := &collector{}
	scheduler := services.NewScheduler(player, log, frozenClock{})
	catalogue := newFakeCatalogue()
	service := services.NewReactionService(
		cue.NewTable(cues), catalogue, scheduler, firstChooser{}, log, frozenClock{},
	)
	return service, catalogue, player, log
}

func journalEvent(name string, at time.Time) event.Event {
	return event.New(event.SourceJournal, name, event.EdgeNone, nil, at)
}

func TestAnEventNoCueClaimsIsIgnoredEntirely(t *testing.T) {
	service, _, player, log := wiring(t, cueFor(t, cue.Definition{
		ID: "Bounty", Source: "journal", Event: "Bounty",
	}))

	service.Handle(journalEvent("Shutdown", moment))

	if len(log.outcomes) != 0 {
		t.Errorf("outcomes = %v, want none: an unclaimed event is not a decision", log.outcomes)
	}
	if len(player.played) != 0 {
		t.Errorf("played %v for an event no cue claims", player.played)
	}
}

func TestAServedCueIsPlayedWithOneTake(t *testing.T) {
	service, catalogue, player, _ := wiring(t, cueFor(t, cue.Definition{
		ID: "Bounty", Source: "journal", Event: "Bounty",
	}))
	catalogue.hold("Bounty", "one.mp3", "two.mp3")

	service.HandleAll([]event.Event{journalEvent("Bounty", moment)})

	if len(player.played) != 1 || len(player.played[0]) != 1 {
		t.Fatalf("played %v, want exactly one take", player.played)
	}
	if player.played[0][0] != "one.mp3" {
		t.Errorf("played %q, want the chooser's pick", player.played[0][0])
	}
}

func TestARepeatInsideTheDedupeWindowIsCollapsed(t *testing.T) {
	service, catalogue, player, log := wiring(t, cueFor(t, cue.Definition{
		ID: "Bounty", Source: "journal", Event: "Bounty",
	}))
	catalogue.hold("Bounty", "one.mp3")

	service.HandleAll([]event.Event{
		journalEvent("Bounty", moment),
		journalEvent("Bounty", moment),
	})

	if len(player.played) != 1 {
		t.Fatalf("played %v, want the repeat collapsed", player.played)
	}
	if !contains(log.outcomes, ports.OutcomeDuplicate) {
		t.Errorf("outcomes = %v, want a duplicate recorded", log.outcomes)
	}
}

// Both the dedupe window and the cooldown are measured from the injected clock, so
// holding one firing back for the cooldown means winding the clock rather than
// spacing the events apart.
func TestACueInsideItsCooldownIsHeldAndRecorded(t *testing.T) {
	player := newFakePlayer()
	log := &collector{}
	moving := &movingClock{now: moment}
	scheduler := services.NewScheduler(player, log, moving)
	catalogue := newFakeCatalogue()
	item := cueFor(t, cue.Definition{
		ID: "Bounty", Source: "journal", Event: "Bounty", Cooldown: time.Minute,
	})
	service := services.NewReactionService(
		cue.NewTable([]cue.Cue{item}), catalogue, scheduler, firstChooser{}, log, moving,
	)
	catalogue.hold("Bounty", "one.mp3")

	service.HandleAll([]event.Event{journalEvent("Bounty", moving.now)})
	moving.now = moment.Add(2 * time.Second)
	service.HandleAll([]event.Event{journalEvent("Bounty", moving.now)})

	if len(player.played) != 1 {
		t.Fatalf("played %v, want the second firing held by the cooldown", player.played)
	}
	if !contains(log.outcomes, ports.OutcomeCooldown) {
		t.Errorf("outcomes = %v, want a cooldown recorded", log.outcomes)
	}
}

func TestACueTheActiveVoiceCannotServeIsRecordedAsUnbound(t *testing.T) {
	player := newFakePlayer()
	log := &collector{}
	moving := &movingClock{now: moment}
	scheduler := services.NewScheduler(player, log, moving)
	catalogue := newFakeCatalogue()
	item := cueFor(t, cue.Definition{ID: "Bounty", Source: "journal", Event: "Bounty"})
	service := services.NewReactionService(
		cue.NewTable([]cue.Cue{item}), catalogue, scheduler, firstChooser{}, log, moving,
	)

	service.Handle(journalEvent("Bounty", moving.now))

	if len(player.played) != 0 {
		t.Fatalf("played %v for a cue the voice does not serve", player.played)
	}
	if !contains(log.outcomes, ports.OutcomeUnbound) {
		t.Errorf("outcomes = %v, want unbound recorded", log.outcomes)
	}

	// A cue listed with no takes behind it is the same silence by a different
	// route, so it has to be reported the same way.
	catalogue.hold("Bounty")
	moving.now = moment.Add(time.Hour)
	service.Handle(journalEvent("Bounty", moving.now))

	if len(player.played) != 0 {
		t.Fatalf("played %v for a cue with no takes behind it", player.played)
	}
}

// Muting silences the sound, not the reasoning: the decision is still made and still
// recorded, so a silent cue can be diagnosed with the volume off.
func TestMutingRecordsWhatWouldHaveBeenSaidWithoutSayingIt(t *testing.T) {
	player := newFakePlayer()
	log := &collector{}
	moving := &movingClock{now: moment}
	scheduler := services.NewScheduler(player, log, moving)
	catalogue := newFakeCatalogue()
	item := cueFor(t, cue.Definition{ID: "Bounty", Source: "journal", Event: "Bounty"})
	service := services.NewReactionService(
		cue.NewTable([]cue.Cue{item}), catalogue, scheduler, firstChooser{}, log, moving,
	)
	catalogue.hold("Bounty", "one.mp3")

	if service.Muted() {
		t.Fatal("a new service started muted")
	}
	service.SetMuted(true)
	if !service.Muted() {
		t.Fatal("SetMuted(true) did not take")
	}

	service.HandleAll([]event.Event{journalEvent("Bounty", moving.now)})

	if len(player.played) != 0 {
		t.Fatalf("played %v while muted", player.played)
	}
	if !contains(log.outcomes, ports.OutcomeDropped) {
		t.Errorf("outcomes = %v, want the muted decision recorded as dropped", log.outcomes)
	}

	service.SetMuted(false)
	moving.now = moment.Add(time.Hour)
	service.HandleAll([]event.Event{journalEvent("Bounty", moving.now)})

	if len(player.played) != 1 {
		t.Errorf("played %v after unmuting, want the sound back", player.played)
	}
}

// A firing never heard holds nothing back: with no take behind it or muted, it starts
// neither the cooldown nor the repeat window, so the next firing that can be heard is heard
// at once (Oliver, 2026-09-13).
func TestAFiringNeverHeardHoldsNothingBack(t *testing.T) {
	player := newFakePlayer()
	log := &collector{}
	moving := &movingClock{now: moment}
	scheduler := services.NewScheduler(player, log, moving)
	catalogue := newFakeCatalogue()
	item := cueFor(t, cue.Definition{
		ID: "Bounty", Source: "journal", Event: "Bounty", Cooldown: time.Minute,
	})
	service := services.NewReactionService(
		cue.NewTable([]cue.Cue{item}), catalogue, scheduler, firstChooser{}, log, moving,
	)
	firing := []event.Event{journalEvent("Bounty", moving.now)}

	service.HandleAll(firing)
	catalogue.hold("Bounty", "one.mp3")
	service.SetMuted(true)
	service.HandleAll(firing)
	service.SetMuted(false)
	service.HandleAll(firing)

	if len(player.played) != 1 {
		t.Fatalf("played %v, want the one firing that could be heard", player.played)
	}
	if contains(log.outcomes, ports.OutcomeCooldown) || contains(log.outcomes, ports.OutcomeDuplicate) {
		t.Errorf("outcomes = %v, want nothing held back by a firing never heard", log.outcomes)
	}
}

func TestDescribeNamesTheVoiceAndItsCoverage(t *testing.T) {
	service, catalogue, _, _ := wiring(t)
	catalogue.active = "Iris"
	catalogue.served = 93
	catalogue.total = 118

	if got := service.Describe(); got != "Iris serves 93 of 118 cues" {
		t.Errorf("Describe() = %q", got)
	}
}

// The reporter is optional: the command-line modes build the path without one, so a
// missing log must not turn every silent decision into a crash.
func TestADecisionIsMadeWithNoReporterWiredUp(t *testing.T) {
	player := newFakePlayer()
	scheduler := services.NewScheduler(player, nil, frozenClock{})
	catalogue := newFakeCatalogue()
	item := cueFor(t, cue.Definition{ID: "Bounty", Source: "journal", Event: "Bounty"})
	service := services.NewReactionService(
		cue.NewTable([]cue.Cue{item}), catalogue, scheduler, firstChooser{}, nil, frozenClock{},
	)

	service.Handle(journalEvent("Bounty", moment))
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// A cue with several takes speaks once: they are alternatives, not parts to play in
// turn.
func TestACueWithSeveralTakesSpeaksOnce(t *testing.T) {
	service, catalogue, player, _ := wiring(t, cueFor(t, cue.Definition{
		ID: "Undocked", Source: "journal", Event: "Undocked",
	}))
	catalogue.hold("Undocked", "a.mp3", "b.mp3", "c.mp3", "d.mp3", "e.mp3")

	service.HandleAll([]event.Event{journalEvent("Undocked", moment)})

	if len(player.played) != 1 {
		t.Fatalf("played %v, want a single utterance", player.played)
	}
	if len(player.played[0]) != 1 {
		t.Fatalf("spoke %d clips for one cue, want 1", len(player.played[0]))
	}
}
