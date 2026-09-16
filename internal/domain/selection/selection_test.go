package selection_test

import (
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/selection"
)

// fixedChooser returns a scripted sequence, so take selection is deterministic.
type fixedChooser struct {
	values []int
	next   int
}

// Intn returns the next scripted value, wrapped into range.
func (f *fixedChooser) Intn(n int) int {
	if len(f.values) == 0 {
		return 0
	}
	value := f.values[f.next%len(f.values)]
	f.next++
	return value % n
}

var start = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

func mustCue(t *testing.T, id string, cooldown time.Duration) cue.Cue {
	t.Helper()
	built, err := cue.New(cue.Definition{
		ID: id, Source: "journal", Event: "Bounty", Cooldown: cooldown,
	})
	if err != nil {
		t.Fatalf("building cue: %v", err)
	}
	return built
}

func TestPickerNeverRepeatsTheImmediatelyPreviousClip(t *testing.T) {
	clips := takesOf("a.mp3", "b.mp3", "c.mp3")
	// A chooser that always asks for index 0 would repeat forever without the guard.
	picker := selection.NewPicker(&fixedChooser{values: []int{0}})

	first, ok := play(picker, "x", clips)
	if !ok {
		t.Fatal("expected a pick")
	}
	for range 5 {
		next, ok := play(picker, "x", clips)
		if !ok {
			t.Fatal("expected a pick")
		}
		if next == first {
			t.Fatalf("picked %q twice in a row", next)
		}
		first = next
	}
}

func TestPickerHandlesSmallFolders(t *testing.T) {
	picker := selection.NewPicker(&fixedChooser{values: []int{0}})

	if _, ok := play(picker, "empty", nil); ok {
		t.Fatal("an empty folder should yield no pick")
	}
	// A single-clip cue must keep speaking, so the no-repeat rule yields to it.
	for range 3 {
		clip, ok := play(picker, "one", takesOf("only.mp3"))
		if !ok || clip != "only.mp3" {
			t.Fatalf("single-clip pick = %q, %v", clip, ok)
		}
	}
}

func TestPickerDrawsAcrossTheWholeFolder(t *testing.T) {
	clips := takesOf("a.mp3", "b.mp3", "c.mp3", "d.mp3")
	picker := selection.NewPicker(&fixedChooser{values: []int{0, 1, 2, 0, 1, 2}})
	seen := make(map[string]bool)
	for range 12 {
		clip, _ := play(picker, "x", clips)
		seen[clip] = true
	}
	if len(seen) < 3 {
		t.Fatalf("only saw %d distinct clips, the picker is not spreading", len(seen))
	}
}

func TestCooldownBlocksASecondFiringInsideTheWindow(t *testing.T) {
	gate := selection.NewCooldownGate()
	item := mustCue(t, "Bounty", 20*time.Second)

	if !fire(gate, item, start) {
		t.Fatal("first firing should be allowed")
	}
	if fire(gate, item, start.Add(19*time.Second)) {
		t.Fatal("second firing inside the cooldown should be blocked")
	}
	if !fire(gate, item, start.Add(21*time.Second)) {
		t.Fatal("firing after the cooldown should be allowed")
	}
}

func TestCooldownOfZeroAlwaysAllows(t *testing.T) {
	gate := selection.NewCooldownGate()
	item := mustCue(t, "Died", 0)
	for offset := range 5 {
		if !fire(gate, item, start.Add(time.Duration(offset)*time.Millisecond)) {
			t.Fatalf("a cue with no cooldown was blocked at offset %d", offset)
		}
	}
}

func TestCooldownIsPerCue(t *testing.T) {
	gate := selection.NewCooldownGate()
	first := mustCue(t, "one", time.Minute)
	second := mustCue(t, "two", time.Minute)

	fire(gate, first, start)
	if !fire(gate, second, start) {
		t.Fatal("one cue's cooldown blocked another cue")
	}
}

func TestDedupeCollapsesRapidRepeats(t *testing.T) {
	window := selection.NewDedupeWindow(900 * time.Millisecond)

	if !seen(window, "a", start) {
		t.Fatal("first sighting should be fresh")
	}
	if seen(window, "a", start.Add(500*time.Millisecond)) {
		t.Fatal("a repeat inside the window should be collapsed")
	}
	if !seen(window, "a", start.Add(time.Second)) {
		t.Fatal("a repeat after the window should be fresh")
	}
	if !seen(window, "b", start) {
		t.Fatal("a different cue should not be collapsed")
	}
}

// A list of takes can hold the same clip more than once, so every candidate can be the clip
// that played last. There is nothing else to choose; silence would be worse than a
// repeat, so the repeat is played.
func TestPickerRepeatsWhenEveryClipIsTheOneItJustPlayed(t *testing.T) {
	picker := selection.NewPicker(&fixedChooser{values: []int{0}})

	first, ok := play(picker, "Bounty", takesOf("a.mp3", "a.mp3"))
	if !ok || first != "a.mp3" {
		t.Fatalf("first pick = %q, %v", first, ok)
	}

	second, ok := play(picker, "Bounty", takesOf("a.mp3", "a.mp3"))
	if !ok || second != "a.mp3" {
		t.Fatalf("second pick = %q, %v, want the repeat rather than silence", second, ok)
	}
}

func TestResetClearsEveryRecordedFiring(t *testing.T) {
	gate := selection.NewCooldownGate()
	item := mustCue(t, "Bounty", 30*time.Second)

	if !fire(gate, item, start) {
		t.Fatal("the first firing was blocked")
	}
	if fire(gate, item, start.Add(time.Second)) {
		t.Fatal("a firing inside the cooldown was allowed")
	}

	gate.Reset()

	if !fire(gate, item, start.Add(time.Second)) {
		t.Error("after Reset the cue was still held by the cleared cooldown")
	}
}
