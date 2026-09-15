package services_test

// A moment switched off on Chatter (FR-622 to FR-627).

import (
	"slices"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
)

// board is a switchboard a test can change between firings.
type board struct{ off map[cue.ID]bool }

func (b *board) Off(id cue.ID) bool { return b.off[id] }

// switchedOff answers a board with the ids named switched off.
func switchedOff(ids ...cue.ID) *board {
	held := &board{off: map[cue.ID]bool{}}
	for _, id := range ids {
		held.off[id] = true
	}
	return held
}

// bounty is a cue for Bounty with no cooldown.
func bounty(t *testing.T) cue.Cue {
	t.Helper()
	return cueFor(t, cue.Definition{ID: "Bounty", Source: "journal", Event: "Bounty"})
}

// FR-622's acceptance: a moment switched off plays nothing and is recorded as off.
func TestAMomentSwitchedOffPlaysNothingAndIsRecordedOff(t *testing.T) {
	service, catalogue, player, log := wiring(t, bounty(t))
	catalogue.hold("Bounty", "one.mp3")
	service.SetSwitchboard(switchedOff("Bounty"))

	service.HandleAll([]event.Event{journalEvent("Bounty", moment)})

	if len(player.played) != 0 {
		t.Errorf("played %v for a moment switched off", player.played)
	}
	if !slices.Equal(log.outcomes, []string{ports.OutcomeOff}) {
		t.Errorf("outcomes = %v, want off alone", log.outcomes)
	}
}

// FR-623's acceptance: a firing switched off starts no cooldown, so switching the moment back on lets the
// next firing play inside what would have been its cooldown.
func TestAFiringSwitchedOffStartsNoCooldown(t *testing.T) {
	player, log := newFakePlayer(), &collector{}
	clock := &movingClock{now: moment}
	catalogue := newFakeCatalogue()
	item := cueFor(t, cue.Definition{ID: "Docked", Source: "journal", Event: "Docked", Cooldown: 30 * time.Second})
	service := services.NewReactionService(
		cue.NewTable([]cue.Cue{item}), catalogue, services.NewScheduler(player, log, clock), firstChooser{}, log, clock,
	)
	catalogue.hold("Docked", "docked.mp3")
	switches := switchedOff("Docked")
	service.SetSwitchboard(switches)

	service.HandleAll([]event.Event{journalEvent("Docked", clock.now)})
	switches.off["Docked"] = false
	clock.now = clock.now.Add(10 * time.Second)
	service.HandleAll([]event.Event{journalEvent("Docked", clock.now)})

	if len(player.played) != 1 {
		t.Errorf("played %v, want the second firing heard", player.played)
	}
}

// FR-623: a firing switched off opens no repeat window, so the same moment firing again at once, switched
// on, is heard rather than taken for a repeat.
func TestAFiringSwitchedOffOpensNoDuplicateWindow(t *testing.T) {
	service, catalogue, player, log := wiring(t, bounty(t))
	catalogue.hold("Bounty", "one.mp3")
	switches := switchedOff("Bounty")
	service.SetSwitchboard(switches)

	service.Handle(journalEvent("Bounty", moment))
	switches.off["Bounty"] = false
	service.HandleAll([]event.Event{journalEvent("Bounty", moment)})

	if len(player.played) != 1 || slices.Contains(log.outcomes, ports.OutcomeDuplicate) {
		t.Errorf("played %v with outcomes %v, want it heard and no duplicate", player.played, log.outcomes)
	}
}

// FR-623's acceptance: a machine voice makes no line for a moment switched off.
func TestAFiringSwitchedOffMakesNoLine(t *testing.T) {
	service, _, _, log, clock, maker := waitingWiring(t)
	service.SetSwitchboard(switchedOff("Docked"))

	service.HandleAll(docked(clock))

	if len(maker.asked) != 0 {
		t.Errorf("asked to make lines for %v", maker.asked)
	}
	if slices.Contains(log.outcomes, ports.OutcomeMaking) {
		t.Errorf("outcomes = %v, want no making", log.outcomes)
	}
}

// FR-624's acceptance: while muted, a moment switched off is recorded as off rather than dropped.
func TestAMomentSwitchedOffWhileMutedIsRecordedOff(t *testing.T) {
	service, catalogue, _, log := wiring(t, bounty(t))
	catalogue.hold("Bounty", "one.mp3")
	service.SetMuted(true)
	service.SetSwitchboard(switchedOff("Bounty"))

	service.HandleAll([]event.Event{journalEvent("Bounty", moment)})

	if !slices.Equal(log.outcomes, []string{ports.OutcomeOff}) {
		t.Errorf("outcomes = %v, want off alone", log.outcomes)
	}
}

// FR-625's acceptance: a moment switched off while it waits in the queue is never played and is recorded
// as off when its turn comes; the one behind it is played instead.
func TestAMomentSwitchedOffInTheQueueIsLetGo(t *testing.T) {
	player, log := newFakePlayer(), &collector{}
	scheduler := services.NewScheduler(player, log, frozenClock{})
	switches := switchedOff()
	scheduler.SetSwitchboard(switches)

	scheduler.Submit(request(t, "HullDamage", "alert", "hull.mp3"))
	scheduler.Advance()
	scheduler.Submit(request(t, "Docked", "notice", "docked.mp3"))
	scheduler.Submit(request(t, "Undocked", "notice", "undocked.mp3"))
	switches.off["Docked"] = true
	player.finish()
	scheduler.Finished()

	want := [][]string{{"hull.mp3"}, {"undocked.mp3"}}
	if !slices.EqualFunc(player.played, want, slices.Equal) {
		t.Errorf("played %v, want %v", player.played, want)
	}
	if !slices.Contains(log.outcomes, ports.OutcomeOff) || scheduler.Pending() != 0 {
		t.Errorf("outcomes = %v with %d pending, want off recorded and nothing waiting", log.outcomes, scheduler.Pending())
	}
}

// FR-625: a moment switched off while it waits for its line is let go when the line is written.
func TestAMomentSwitchedOffWhileItsLineIsMadeIsLetGo(t *testing.T) {
	service, catalogue, player, log, clock, _ := waitingWiring(t)
	switches := switchedOff()
	service.SetSwitchboard(switches)

	service.HandleAll(docked(clock))
	switches.off["Docked"] = true
	catalogue.hold("Docked", "docked.flac")
	service.Tick()

	if len(player.played) != 0 {
		t.Errorf("played %v for a moment switched off while its line was made", player.played)
	}
	if log.outcomes[len(log.outcomes)-1] != ports.OutcomeOff {
		t.Errorf("outcomes = %v, want off last", log.outcomes)
	}
}

// FR-626's acceptance: switching a moment off leaves the take already playing to end on its own.
func TestSwitchingOffAMomentLeavesItsTakePlaying(t *testing.T) {
	service, catalogue, player, _ := wiring(t, bounty(t))
	catalogue.hold("Bounty", "one.mp3")
	switches := switchedOff()
	service.SetSwitchboard(switches)

	service.HandleAll([]event.Event{journalEvent("Bounty", moment)})
	switches.off["Bounty"] = true
	service.Tick()

	if !player.playing || player.stops != 0 {
		t.Errorf("playing = %t after %d stops, want the take still playing", player.playing, player.stops)
	}
}

// FR-636's acceptance: with station traffic switched off and other npc messages switched on, a
// station's docking answer plays nothing and is recorded as off while a military message is answered.
func TestStationTrafficSwitchedOffLeavesOtherMessagesSpoken(t *testing.T) {
	npc := cueFor(t, cue.Definition{
		ID: "ReceiveText.Channel.npc", Source: "journal", Event: "ReceiveText",
		Match: map[string]string{"Channel": "npc"}, Priority: "ambient",
	})
	traffic := cueFor(t, cue.Definition{
		ID: "ReceiveText.StationTraffic", Source: "journal", Event: "ReceiveText",
		Begins:   map[string][]string{"Message": {"STATION_", "DockingChatter_", "DockingFailed_"}},
		Priority: "ambient",
	})
	service, catalogue, player, log := wiring(t, npc, traffic)
	catalogue.hold("ReceiveText.Channel.npc", "npc.mp3")
	catalogue.hold("ReceiveText.StationTraffic", "traffic.mp3")
	service.SetSwitchboard(switchedOff("ReceiveText.StationTraffic"))

	message := func(key string) event.Event {
		return event.New(event.SourceJournal, "ReceiveText", event.EdgeNone,
			map[string]any{"Channel": "npc", "Message": key}, moment)
	}
	service.HandleAll([]event.Event{message("$STATION_docking_granted;")})
	if len(player.played) != 0 || !slices.Equal(log.outcomes, []string{ports.OutcomeOff}) {
		t.Fatalf("played %v with outcomes %v, want nothing played and off", player.played, log.outcomes)
	}

	service.HandleAll([]event.Event{message("$Military_Patrol01;")})
	if want := [][]string{{"npc.mp3"}}; !slices.EqualFunc(player.played, want, slices.Equal) {
		t.Errorf("played %v, want %v", player.played, want)
	}
}

// FR-627's acceptance: a switch applies from the next firing, with nothing rebuilt.
func TestASwitchAppliesToTheNextFiring(t *testing.T) {
	service, catalogue, player, log := wiring(t, bounty(t))
	catalogue.hold("Bounty", "one.mp3")
	switches := switchedOff()
	service.SetSwitchboard(switches)

	service.HandleAll([]event.Event{journalEvent("Bounty", moment)})
	switches.off["Bounty"] = true
	service.HandleAll([]event.Event{journalEvent("Bounty", moment)})

	if len(player.played) != 1 || log.outcomes[len(log.outcomes)-1] != ports.OutcomeOff {
		t.Errorf("played %v with outcomes %v, want one take then off", player.played, log.outcomes)
	}
}
