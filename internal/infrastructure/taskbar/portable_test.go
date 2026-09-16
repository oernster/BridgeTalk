package taskbar

// The tray's rules that hold on every platform: the wait for the desktop's watcher (FR-814), plus
// the menu entries and hover text both trays show (FR-710).

import (
	"testing"
	"time"
)

// The watcher is asked until it answers, sleeping between asks; with none it gives up once the grace
// period has passed, having asked at its start and at every step up to its end.
func TestTheWatcherIsAskedUntilItAnswersOrTheGracePeriodEnds(t *testing.T) {
	every, grace := time.Second, 15*time.Second
	asks, slept := 0, time.Duration(0)
	sleep := func(step time.Duration) { slept += step }

	if awaitWatcher(func() bool { asks++; return false }, every, grace, sleep) {
		t.Fatal("a watcher that never answered was taken as there")
	}
	if asks != 16 || slept != grace {
		t.Errorf("asked %d times over %v, want 16 asks over %v", asks, slept, grace)
	}

	asks, slept = 0, 0
	if !awaitWatcher(func() bool { asks++; return asks == 3 }, every, grace, sleep) {
		t.Fatal("a watcher that answered on the third ask was not taken")
	}
	if asks != 3 || slept != 2*every {
		t.Errorf("asked %d times over %v, want 3 asks over %v", asks, slept, 2*every)
	}
}

// Both trays read the menu and the hover text from one place: the cast voice checked by every field
// that identifies it, a separator before each new kind and the state said in the hover text.
func TestTheMenuAndHoverTextReadTheSameOnEveryTray(t *testing.T) {
	voices := []Choice{
		{Voice: Voice{Kind: Recorded, Name: "Alice"}, Label: "Alice Hart"},
		{Voice: Voice{Kind: Machine, Name: "Alice"}, Label: "Alice (British, female)"},
		{Voice: Voice{Kind: Plugin, Plugin: "Crew", Name: "one"}, Label: "The Officer"},
	}
	entries := voiceEntries(voices, Voice{Kind: Machine, Name: "Alice"})
	want := []voiceEntry{
		{label: "Alice Hart"},
		{label: "Alice (British, female)", checked: true, separated: true},
		{label: "The Officer", separated: true},
	}
	for index := range want {
		if entries[index] != want[index] {
			t.Errorf("entry %d = %+v, want %+v", index, entries[index], want[index])
		}
	}
	if got := hoverText("Title", Voice{}, "", true); got != "Title (muted)" {
		t.Errorf("hover text = %q", got)
	}
	if got := hoverText("Title", voices[0].Voice, labelOf(voices, voices[0].Voice), false); got != "Title: Alice Hart" {
		t.Errorf("hover text = %q", got)
	}
	if got := labelOf(voices, Voice{Kind: Plugin, Plugin: "Other", Name: "one"}); got != "one" {
		t.Errorf("a voice the menu does not hold is shown as %q, want its name", got)
	}
}
