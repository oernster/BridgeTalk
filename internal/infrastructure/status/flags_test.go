package status_test

import (
	"fmt"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/infrastructure/status"
)

// valuesBody builds a status file carrying the non-bit values, with both flag words
// held still so nothing but the value under test can produce an event.
// The flag words carry atHelm and nothing else, so the bit path stays quiet across
// every reading here while the values move. They cannot both be zero: that is the
// game saying nobody is playing; a reading of it ends the session rather than
// reporting a commander whose pips happen to have shifted.
func valuesBody(gui, fire int, pips string) string {
	return fmt.Sprintf(
		`{"timestamp":%q,"Flags":%d,"Flags2":0,"GuiFocus":%d,"FireGroup":%d,"Pips":%s}`,
		stamped, atHelm, gui, fire, pips,
	)
}

// A value event has no falling edge. The value simply became something else, so it
// is reported as rising with the new reading as its payload.
func assertValue(t *testing.T, raised event.Event, name, value string) {
	t.Helper()
	if raised.Name() != name {
		t.Fatalf("name: got %q, want %q", raised.Name(), name)
	}
	if raised.Edge() != event.EdgeRising {
		t.Fatalf("edge: got %q, want rising", raised.Edge())
	}
	if got, ok := raised.Field("value"); !ok || got != value {
		t.Fatalf("payload: got %v (present %v), want %q", got, ok, value)
	}
}

func TestChangingTheFocusedPanelIsReportedByName(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, valuesBody(0, 0, "[4,4,4]"))

	raised := only(t, pollAfter(t, watcher, dir, valuesBody(6, 0, "[4,4,4]")))
	assertValue(t, raised, status.ValueGuiFocus, status.GuiGalaxyMap)
}

// A game update adding a panel must not crash the watcher or emit a bare number. An
// unknown focus reads as no focus, which is the safe interpretation.
func TestAFocusValueThisApplicationDoesNotKnowReadsAsNoFocus(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, valuesBody(0, 0, "[4,4,4]"))

	raised := only(t, pollAfter(t, watcher, dir, valuesBody(99, 0, "[4,4,4]")))
	assertValue(t, raised, status.ValueGuiFocus, status.GuiNoFocus)
}

func TestChangingTheFireGroupIsReportedAsItsNumber(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, valuesBody(0, 0, "[4,4,4]"))

	raised := only(t, pollAfter(t, watcher, dir, valuesBody(0, 2, "[4,4,4]")))
	assertValue(t, raised, status.ValueFireGroup, "2")
}

// The game reports pips in half-pip units and rewrites them constantly. What the cue
// table reacts to is which capacitor leads, so only a change of leader is news.
func TestPipsAreReportedAsTheLeadingCapacitor(t *testing.T) {
	cases := []struct {
		name  string
		pips  string
		leads string
	}{
		{"systems", "[8,2,2]", status.PipsSystems},
		{"engines", "[2,8,2]", status.PipsEngines},
		{"weapons", "[2,2,8]", status.PipsWeapons},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			dir := t.TempDir()
			watcher := primedWatcher(t, dir, valuesBody(0, 0, "[4,4,4]"))

			raised := only(t, pollAfter(t, watcher, dir, valuesBody(0, 0, each.pips)))
			assertValue(t, raised, status.ValuePips, each.leads)
		})
	}
}

func TestAnEvenSpreadOfPipsIsBalanced(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, valuesBody(0, 0, "[8,2,2]"))

	raised := only(t, pollAfter(t, watcher, dir, valuesBody(0, 0, "[4,4,4]")))
	assertValue(t, raised, status.ValuePips, status.PipsBalanced)
}

// Two capacitors tied at the top is nobody leading, so it reads as balanced rather
// than picking whichever the game happened to list first.
func TestTwoCapacitorsTiedAtTheTopReadAsBalanced(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, valuesBody(0, 0, "[8,2,2]"))

	raised := only(t, pollAfter(t, watcher, dir, valuesBody(0, 0, "[6,6,0]")))
	assertValue(t, raised, status.ValuePips, status.PipsBalanced)
}

// A truncated pip array cannot name a leader, so it is balanced. The array also
// changed length, which is itself the change the watcher notices.
func TestAPipArrayShorterThanThreeReadsAsBalanced(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, valuesBody(0, 0, "[8,2,2]"))

	raised := only(t, pollAfter(t, watcher, dir, valuesBody(0, 0, "[4,4]")))
	assertValue(t, raised, status.ValuePips, status.PipsBalanced)
}

// Moving pips around without changing which capacitor leads is not news. The
// underlying numbers differ, so this proves the reduction happens before the
// comparison rather than after it.
func TestMovingPipsWithoutChangingTheLeaderIsNotNews(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, valuesBody(0, 0, "[8,2,2]"))

	if events := pollAfter(t, watcher, dir, valuesBody(0, 0, "[6,4,2]")); len(events) != 0 {
		t.Fatalf("a change of leader was reported where there was none: %v", names(events))
	}
}

// Every value moving at once produces one event each; the flag words being
// untouched proves the bit path stays quiet when nothing bit-shaped changed.
func TestEveryChangedValueGetsItsOwnEvent(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, valuesBody(0, 0, "[4,4,4]"))

	events := pollAfter(t, watcher, dir, valuesBody(7, 3, "[2,2,8]"))
	if len(events) != 3 {
		t.Fatalf("got %d events %v, want one each for focus, fire group and pips",
			len(events), names(events))
	}
	assertValue(t, events[0], status.ValueGuiFocus, status.GuiSystemMap)
	assertValue(t, events[1], status.ValueFireGroup, "3")
	assertValue(t, events[2], status.ValuePips, status.PipsWeapons)
}
