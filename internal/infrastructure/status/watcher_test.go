package status_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/infrastructure/status"
)

// observed is the clock reading the watcher falls back to when the file carries no
// timestamp it can parse.
var observed = time.Date(2026, 8, 26, 9, 30, 0, 0, time.UTC)

// stamped is the moment written into the test status files, distinct from observed
// so a test can tell which of the two an event was dated by.
const stamped = "2026-08-26T11:15:00Z"

// writeStatus writes the status file the game would have written.
func writeStatus(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, "Status.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing %q: %v", path, err)
	}
}

// primedWatcher returns a watcher whose baseline is first, with the priming poll
// already taken and asserted silent.
func primedWatcher(t *testing.T, dir, first string) *status.Watcher {
	t.Helper()
	writeStatus(t, dir, first)
	watcher, err := status.NewWatcher(dir, func() time.Time { return observed })
	if err != nil {
		t.Fatalf("new watcher: %v", err)
	}
	events, err := watcher.Poll()
	if err != nil {
		t.Fatalf("priming poll: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("the priming poll emitted %d events; it must emit none", len(events))
	}
	return watcher
}

// pollAfter rewrites the status file and returns whatever the next poll reports.
func pollAfter(t *testing.T, watcher *status.Watcher, dir, body string) []event.Event {
	t.Helper()
	writeStatus(t, dir, body)
	events, err := watcher.Poll()
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	return events
}

// only asserts that exactly one event was emitted and returns it.
func only(t *testing.T, events []event.Event) event.Event {
	t.Helper()
	if len(events) != 1 {
		t.Fatalf("expected exactly one event, got %d: %v", len(events), names(events))
	}
	return events[0]
}

// names renders the event names for a failure message.
func names(events []event.Event) []string {
	out := make([]string, 0, len(events))
	for _, each := range events {
		out = append(out, each.Name())
	}
	return out
}

// flagsBody builds a status file carrying only the two flag words.
func flagsBody(flags, flags2 uint32) string {
	return fmt.Sprintf(
		`{"timestamp":%q,"Flags":%d,"Flags2":%d,"GuiFocus":0,"FireGroup":0,"Pips":[4,4,4]}`,
		stamped, flags, flags2,
	)
}

// atHelm is the bit saying the commander is in the main ship. Every fixture below
// carries it, because the game sets one of these wherever a commander can be and a
// reading with no flag set at all is it saying nobody is playing. A baseline of no
// flags would be the end of a session rather than a ship with nothing switched on.
var atHelm = status.FlagBit(status.FlagInMainShip)

func TestAWatcherRefusesADirectoryWithNoStatusFile(t *testing.T) {
	if _, err := status.NewWatcher(t.TempDir(), func() time.Time { return observed }); err == nil {
		t.Fatal("a directory with no Status.json was accepted")
	}
}

func TestAWatcherNamesItselfAndTheFileItWatches(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, flagsBody(0, 0))

	if watcher.Name() != "status" {
		t.Fatalf("name: got %q, want %q", watcher.Name(), "status")
	}
	want := filepath.Join(dir, "Status.json")
	if watcher.Path() != want {
		t.Fatalf("path: got %q, want %q", watcher.Path(), want)
	}
}

// The first poll is state, not news. Emitting on it would fire a cue for every flag
// that merely happened to be set when the application started.
func TestTheFirstPollPrimesTheBaselineAndSaysNothing(t *testing.T) {
	dir := t.TempDir()
	primedWatcher(t, dir, flagsBody(1, 0))
}

func TestARewriteThatChangesNothingIsNotNews(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, flagsBody(1, 0))

	if events := pollAfter(t, watcher, dir, flagsBody(1, 0)); len(events) != 0 {
		t.Fatalf("an identical rewrite emitted %v", names(events))
	}
}

func TestASetFlagRisesAndAClearedFlagFalls(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, flagsBody(atHelm, 0))
	docked := atHelm | status.FlagBit(status.FlagDocked)

	raised := only(t, pollAfter(t, watcher, dir, flagsBody(docked, 0)))
	if raised.Name() != status.FlagDocked || raised.Edge() != event.EdgeRising {
		t.Fatalf("rising: got %q %q", raised.Name(), raised.Edge())
	}
	if value, ok := raised.Field("value"); !ok || value != true {
		t.Fatalf("rising payload: got %v, %v", value, ok)
	}
	if raised.Source() != event.SourceStatus {
		t.Fatalf("source: got %q", raised.Source())
	}

	dropped := only(t, pollAfter(t, watcher, dir, flagsBody(atHelm, 0)))
	if dropped.Name() != status.FlagDocked || dropped.Edge() != event.EdgeFalling {
		t.Fatalf("falling: got %q %q", dropped.Name(), dropped.Edge())
	}
	if value, ok := dropped.Field("value"); !ok || value != false {
		t.Fatalf("falling payload: got %v, %v", value, ok)
	}
}

// Flags2 is the Odyssey word. It is a separate table and needs its own proof that
// the watcher reads it at all.
func TestTheOdysseyFlagWordIsWatchedToo(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, flagsBody(atHelm, 0))

	raised := only(t, pollAfter(t, watcher, dir, flagsBody(atHelm, 1)))
	if raised.Name() != status.FlagOnFoot || raised.Edge() != event.EdgeRising {
		t.Fatalf("got %q %q, want %q rising", raised.Name(), raised.Edge(), status.FlagOnFoot)
	}
}

// A burst of simultaneous changes has to arrive in a fixed order or the reaction log
// reads differently every run and no test of it can be stable.
func TestSimultaneousBitsAreReportedInAscendingBitOrder(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, flagsBody(atHelm, 0))

	// Docked, LandingGearDown and ShieldsUp, set together.
	together := atHelm | status.FlagBit(status.FlagDocked) |
		status.FlagBit(status.FlagLandingGearDown) | status.FlagBit(status.FlagShieldsUp)
	events := pollAfter(t, watcher, dir, flagsBody(together, 0))
	want := []string{status.FlagDocked, status.FlagLandingGearDown, status.FlagShieldsUp}
	got := names(events)
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// A bit the application does not react to is not in the table, so it must not
// produce an event even though the flag word changed.
func TestAnUnknownFlagBitIsIgnored(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, flagsBody(atHelm, 0))

	// Bit 2048 in Flags2 has no entry in the Odyssey table.
	if events := pollAfter(t, watcher, dir, flagsBody(atHelm, 2048)); len(events) != 0 {
		t.Fatalf("an unmapped bit emitted %v", names(events))
	}
}

func TestAnEventIsDatedByTheFilesOwnTimestamp(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, flagsBody(atHelm, 0))

	raised := only(t, pollAfter(t, watcher, dir,
		flagsBody(atHelm|status.FlagBit(status.FlagDocked), 0)))
	want, err := time.Parse(time.RFC3339, stamped)
	if err != nil {
		t.Fatalf("parsing the fixture timestamp: %v", err)
	}
	if !raised.At().Equal(want) {
		t.Fatalf("at: got %v, want %v", raised.At(), want)
	}
}

// The game is the writer. A timestamp this application cannot parse is not a reason
// to drop the change, so the clock stands in for it.
func TestATimestampThatWillNotParseFallsBackToTheClock(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, flagsBody(atHelm, 0))

	raised := only(t, pollAfter(t, watcher, dir, fmt.Sprintf(
		`{"timestamp":"not a time","Flags":%d,"Flags2":0,"GuiFocus":0,"FireGroup":0,"Pips":[4,4,4]}`,
		atHelm|status.FlagBit(status.FlagDocked))))
	if !raised.At().Equal(observed) {
		t.Fatalf("at: got %v, want the clock reading %v", raised.At(), observed)
	}
}

// The game rewrites this file far more often than its contents change, so catching
// it mid-write is expected. A half-written file is silence, never an error.
func TestAFileCaughtMidWriteIsSilentRatherThanAnError(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, flagsBody(1, 0))

	if events := pollAfter(t, watcher, dir, `{"Flags":1,`); len(events) != 0 {
		t.Fatalf("unparseable JSON emitted %v", names(events))
	}
	if events := pollAfter(t, watcher, dir, ""); len(events) != 0 {
		t.Fatalf("an empty file emitted %v", names(events))
	}
}

// A file that is not there is silence rather than an error, the same as a file caught
// mid-write. This is not what the game does on exit, whatever the comment here used to
// claim: it rewrites the file with a zeroed Flags rather than removing it, which is
// covered above. This covers the directory being moved or the file being deleted under
// a running application.
func TestAMissingFileIsSilentRatherThanAnError(t *testing.T) {
	dir := t.TempDir()
	watcher := primedWatcher(t, dir, flagsBody(1, 0))

	if err := os.Remove(filepath.Join(dir, "Status.json")); err != nil {
		t.Fatalf("removing the status file: %v", err)
	}
	events, err := watcher.Poll()
	if err != nil {
		t.Fatalf("poll over a removed file: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("a removed file emitted %v", names(events))
	}
}

// Leaving the game is not the ship doing anything.
//
// On exit the game rewrites the file as one line carrying nothing but a zeroed Flags:
// `{ "timestamp":"...", "event":"Status", "Flags":0 }`, with no Flags2, GuiFocus,
// FireGroup or Pips at all. Diffed against the state the commander left, every bit
// that was set reads as falling, so quitting from a landed ship announced the landing
// gear being raised, the shields dropping and the ship undocking, none of which
// happened.
func TestLeavingTheGameIsNotTheShipDoingAnything(t *testing.T) {
	dir := t.TempDir()
	// Docked in the main ship with the gear down and the shields up, which is where
	// anybody quitting from a station is sitting.
	landed := status.FlagBit(status.FlagDocked) | status.FlagBit(status.FlagLandingGearDown) |
		status.FlagBit(status.FlagShieldsUp) | status.FlagBit(status.FlagInMainShip)
	watcher := primedWatcher(t, dir, flagsBody(landed, 0))

	exited := fmt.Sprintf(`{ "timestamp":%q, "event":"Status", "Flags":0 }`, stamped)
	if events := pollAfter(t, watcher, dir, exited); len(events) != 0 {
		t.Fatalf("leaving the game emitted %v", names(events))
	}
}

// Coming back is a fresh baseline, not a burst of news.
//
// The session that follows may be a different ship in a different place, so the first
// reading of it is state rather than change, exactly as the first reading of the run
// is. What moves after that is news again.
func TestReturningToTheGamePrimesAfreshRatherThanAnnouncingTheNewSession(t *testing.T) {
	dir := t.TempDir()
	inShip := status.FlagBit(status.FlagInMainShip)
	watcher := primedWatcher(t, dir, flagsBody(inShip|status.FlagBit(status.FlagSupercruise), 0))

	exited := fmt.Sprintf(`{ "timestamp":%q, "event":"Status", "Flags":0 }`, stamped)
	if events := pollAfter(t, watcher, dir, exited); len(events) != 0 {
		t.Fatalf("leaving the game emitted %v", names(events))
	}

	// A new session, landed this time. Nothing is announced for it.
	landed := inShip | status.FlagBit(status.FlagLanded) | status.FlagBit(status.FlagLandingGearDown)
	if events := pollAfter(t, watcher, dir, flagsBody(landed, 0)); len(events) != 0 {
		t.Fatalf("the first reading of a new session emitted %v", names(events))
	}

	// A flag changing within that session is news again.
	raised := only(t, pollAfter(t, watcher, dir, flagsBody(inShip|status.FlagBit(status.FlagLanded), 0)))
	if raised.Name() != status.FlagLandingGearDown || raised.Edge() != event.EdgeFalling {
		t.Fatalf("got %s %s, want %s falling", raised.Name(), raised.Edge(), status.FlagLandingGearDown)
	}
}
