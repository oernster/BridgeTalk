package selection_test

// Asking is not firing. The gate and the window answer without recording anything, so a
// firing that is never heard holds nothing back; only a firing handed over to be spoken is
// recorded against the ones that follow it.

import (
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/selection"
	"github.com/oernster/bridge-talk/internal/domain/take"
)

// fire asks the gate and records the firing when it passes, which is what the reaction
// service does with a firing the scheduler takes.
func fire(gate *selection.CooldownGate, item cue.Cue, now time.Time) bool {
	if !gate.Open(item, now) {
		return false
	}
	gate.Record(item.ID(), now)
	return true
}

// seen asks the window and marks the cue when it is fresh, for the same reason.
func seen(window *selection.DedupeWindow, id cue.ID, now time.Time) bool {
	if !window.Fresh(id, now) {
		return false
	}
	window.Mark(id, now)
	return true
}

// takesOf builds one take per path, which is what every voice that records one file per
// take answers with. The tests that care about parts build their takes by hand.
func takesOf(paths ...string) []take.Take {
	out := make([]take.Take, 0, len(paths))
	for _, path := range paths {
		out = append(out, take.Of(path))
	}
	return out
}

// play picks a take and records it as played, for the same reason again. It answers the
// take's key, since that is what the caller compares.
func play(picker *selection.Picker, id cue.ID, takes []take.Take) (string, bool) {
	chosen, ok := picker.Pick(id, takes)
	if ok {
		picker.Played(id, chosen)
	}
	return chosen.Key(), ok
}

// FR-591: takes that are spans of one shared file are different takes, so the take played last is
// avoided among them as it is among files of their own.
func TestSpansOfOneFileAreAvoidedOneByOne(t *testing.T) {
	takes := []take.Take{
		{take.SpanOf("many.bin", "mp3", 0, 4000)},
		{take.SpanOf("many.bin", "mp3", 4000, 4000)},
		{take.SpanOf("many.bin", "mp3", 8000, 4000)},
	}
	// A chooser that always asks for index 0 repeats the first take unless the picker avoids it.
	picker := selection.NewPicker(&fixedChooser{values: []int{0}})

	first, _ := play(picker, "DockingGranted", takes)
	second, ok := play(picker, "DockingGranted", takes)

	if !ok || second == first {
		t.Errorf("the second firing played the same span as the first, want a different one")
	}
}

// Picking is not playing either: a take picked for a firing that is then let go was never
// heard, so it is not the take to avoid next time (FR-610).
func TestPickingRecordsNothing(t *testing.T) {
	picker := selection.NewPicker(&fixedChooser{values: []int{0}})
	takes := []take.Take{take.Of("a.mp3"), take.Of("b.mp3")}

	for range 2 {
		if got, _ := picker.Pick("Bounty", takes); got.Key() != "a.mp3" {
			t.Fatalf("pick = %v, want a.mp3: a take only picked was remembered as played", got)
		}
	}

	picker.Played("Bounty", take.Of("a.mp3"))
	if got, _ := picker.Pick("Bounty", takes); got.Key() != "b.mp3" {
		t.Fatalf("pick = %v, want b.mp3: a take played did not hold back the next pick", got)
	}
}

func TestAskingRecordsNothing(t *testing.T) {
	item, err := cue.New(cue.Definition{ID: "Bounty", Source: "journal", Event: "Bounty", Cooldown: time.Minute})
	if err != nil {
		t.Fatalf("building the cue: %v", err)
	}
	start := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	gate := selection.NewCooldownGate()
	window := selection.NewDedupeWindow(time.Second)

	for range 2 {
		if !gate.Open(item, start) {
			t.Fatal("the gate closed on a cue that was only asked about")
		}
		if !window.Fresh(item.ID(), start) {
			t.Fatal("the window counted a cue that was only asked about")
		}
	}

	gate.Record(item.ID(), start)
	window.Mark(item.ID(), start)
	if gate.Open(item, start) || window.Fresh(item.ID(), start) {
		t.Fatal("a recorded firing did not hold back the next")
	}
}
