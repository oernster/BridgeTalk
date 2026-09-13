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
