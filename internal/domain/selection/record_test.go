package selection_test

// Asking is not firing. The gate and the window answer without recording anything, so a
// firing that is never heard holds nothing back; only a firing handed over to be spoken is
// recorded against the ones that follow it.

import (
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/selection"
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

// play picks a take and records it as played, for the same reason again.
func play(picker *selection.Picker, id cue.ID, clips []string) (string, bool) {
	clip, ok := picker.Pick(id, clips)
	if ok {
		picker.Played(id, clip)
	}
	return clip, ok
}

// Picking is not playing either: a take picked for a firing that is then let go was never
// heard, so it is not the take to avoid next time (FR-610).
func TestPickingRecordsNothing(t *testing.T) {
	picker := selection.NewPicker(&fixedChooser{values: []int{0}})
	clips := []string{"a.mp3", "b.mp3"}

	for range 2 {
		if got, _ := picker.Pick("Bounty", clips); got != "a.mp3" {
			t.Fatalf("pick = %q, want a.mp3: a take only picked was remembered as played", got)
		}
	}

	picker.Played("Bounty", "a.mp3")
	if got, _ := picker.Pick("Bounty", clips); got != "b.mp3" {
		t.Fatalf("pick = %q, want b.mp3: a take played did not hold back the next pick", got)
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
