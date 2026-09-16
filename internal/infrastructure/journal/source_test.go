package journal_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/infrastructure/journal"
)

// observed is the clock reading a line falls back to when it carries no timestamp
// the source can parse.
var observed = time.Date(2026, 8, 26, 9, 30, 0, 0, time.UTC)

// clock is the injected time reading; the source never reads the wall clock itself.
func clock() time.Time { return observed }

// journalName builds a journal file name for a given sortable stamp. The game embeds
// a timestamp in the name, which is why lexical order is time order.
func journalName(stamp string) string {
	return "Journal." + stamp + ".01.log"
}

// line renders one journal line.
func line(name, stamp string) string {
	return `{"timestamp":"` + stamp + `","event":"` + name + `"}` + "\n"
}

// sourceOver builds a source over a directory holding one journal with the given body.
func sourceOver(t *testing.T, dir, name, body string) *journal.Source {
	t.Helper()
	writeFile(t, dir, name, body)
	source, err := journal.NewSource(dir, clock)
	if err != nil {
		t.Fatalf("new source: %v", err)
	}
	return source
}

// poll takes a poll and fails the test if it errors.
func poll(t *testing.T, source *journal.Source) []event.Event {
	t.Helper()
	events, err := source.Poll()
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	return events
}

func TestASourceRefusesADirectoryThatIsNotThere(t *testing.T) {
	_, err := journal.NewSource(filepath.Join(t.TempDir(), "nowhere"), clock)
	if err == nil {
		t.Fatal("a directory that does not exist was accepted")
	}
}

func TestASourceRefusesADirectoryHoldingNoJournals(t *testing.T) {
	if _, err := journal.NewSource(t.TempDir(), clock); err == nil {
		t.Fatal("a directory with no journal files was accepted")
	}
}

// A directory name containing a glob metacharacter makes the search pattern itself
// invalid. That is a real path a user can pick, so it must surface as an error rather
// than as an empty result that looks like a quiet game.
func TestASourceReportsADirectoryNameThatCannotBeSearched(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bracket[")
	if err := os.Mkdir(bad, 0o755); err != nil {
		t.Skipf("this filesystem will not hold a directory named %q: %v", bad, err)
	}
	if _, err := journal.NewSource(bad, clock); err == nil {
		t.Fatal("a directory whose name breaks the search pattern was accepted")
	}
}

func TestASourceNamesItselfAndTheFileItFollows(t *testing.T) {
	dir := t.TempDir()
	source := sourceOver(t, dir, journalName("2026-08-26T090000"), "")

	if source.Name() != "journal" {
		t.Fatalf("name: got %q, want %q", source.Name(), "journal")
	}
	want := filepath.Join(dir, journalName("2026-08-26T090000"))
	if source.Path() != want {
		t.Fatalf("path: got %q, want %q", source.Path(), want)
	}
}

// Launching the application mid-session must not fire hundreds of cues at once, so
// construction seeks to the end of what is already there.
func TestASourceStartsAtTheEndAndNeverReplaysHistory(t *testing.T) {
	dir := t.TempDir()
	source := sourceOver(t, dir, journalName("2026-08-26T090000"),
		line("Shutdown", "2026-08-26T09:00:00Z")+line("LoadGame", "2026-08-26T09:01:00Z"))

	if events := poll(t, source); len(events) != 0 {
		t.Fatalf("history was replayed: %d events", len(events))
	}
}

func TestAppendedLinesArriveAsEvents(t *testing.T) {
	dir := t.TempDir()
	name := journalName("2026-08-26T090000")
	source := sourceOver(t, dir, name, "")
	poll(t, source)

	appendTo(t, filepath.Join(dir, name), line("FSDJump", "2026-08-26T09:05:00Z"))
	events := poll(t, source)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Name() != "FSDJump" {
		t.Fatalf("name: got %q, want %q", events[0].Name(), "FSDJump")
	}
	if events[0].Source() != event.SourceJournal {
		t.Fatalf("source: got %q", events[0].Source())
	}
	if events[0].Edge() != event.EdgeNone {
		t.Fatalf("edge: got %q, want none; a journal line has no direction", events[0].Edge())
	}
	want, err := time.Parse(time.RFC3339, "2026-08-26T09:05:00Z")
	if err != nil {
		t.Fatalf("parsing the fixture timestamp: %v", err)
	}
	if !events[0].At().Equal(want) {
		t.Fatalf("at: got %v, want %v", events[0].At(), want)
	}
}

// Every field of the line reaches the cue engine, which is how a cue can react to a
// jump distance or a bounty amount rather than only to the event name.
func TestALinesPayloadReachesTheEvent(t *testing.T) {
	dir := t.TempDir()
	name := journalName("2026-08-26T090000")
	source := sourceOver(t, dir, name, "")
	poll(t, source)

	appendTo(t, filepath.Join(dir, name),
		`{"timestamp":"2026-08-26T09:05:00Z","event":"Bounty","Reward":4200}`+"\n")
	events := poll(t, source)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if value, ok := events[0].Field("Reward"); !ok || value != 4200.0 {
		t.Fatalf("Reward: got %v (present %v), want 4200", value, ok)
	}
}

// The game is the writer and it occasionally emits something this application has no
// schema for. One unreadable line is not an error worth surfacing; it is dropped and
// the readable lines around it still arrive.
func TestAMalformedLineIsDroppedRatherThanFailingThePoll(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"not JSON at all", "this is not JSON\n"},
		{"no event field", `{"timestamp":"2026-08-26T09:05:00Z"}` + "\n"},
		{"an event that is not a string", `{"event":42}` + "\n"},
		{"an empty event name", `{"event":""}` + "\n"},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			dir := t.TempDir()
			name := journalName("2026-08-26T090000")
			source := sourceOver(t, dir, name, "")
			poll(t, source)

			appendTo(t, filepath.Join(dir, name),
				each.body+line("FSDJump", "2026-08-26T09:06:00Z"))
			events := poll(t, source)
			if len(events) != 1 || events[0].Name() != "FSDJump" {
				t.Fatalf("got %d events, want only the readable one", len(events))
			}
		})
	}
}

// A line the game wrote without a timestamp it can parse is still an event. The
// injected clock stands in, which is why the domain never reads the wall clock.
func TestALineWithNoUsableTimestampIsDatedByTheClock(t *testing.T) {
	dir := t.TempDir()
	name := journalName("2026-08-26T090000")
	source := sourceOver(t, dir, name, "")
	poll(t, source)

	appendTo(t, filepath.Join(dir, name), `{"event":"FSDJump"}`+"\n")
	appendTo(t, filepath.Join(dir, name), `{"event":"Music","timestamp":"nonsense"}`+"\n")
	events := poll(t, source)
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	for _, each := range events {
		if !each.At().Equal(observed) {
			t.Fatalf("%q: got %v, want the clock reading %v", each.Name(), each.At(), observed)
		}
	}
}

// A new journal file means a new play session, so its contents genuinely are new and
// it is read from byte zero rather than from the old file's offset.
func TestANewJournalIsFollowedFromItsFirstByte(t *testing.T) {
	dir := t.TempDir()
	source := sourceOver(t, dir, journalName("2026-08-26T090000"),
		line("Shutdown", "2026-08-26T09:00:00Z"))
	poll(t, source)

	later := journalName("2026-08-26T100000")
	writeFile(t, dir, later, line("LoadGame", "2026-08-26T10:00:00Z"))

	events := poll(t, source)
	if len(events) != 1 || events[0].Name() != "LoadGame" {
		t.Fatalf("got %d events %v, want the new file read from its start", len(events), events)
	}
	if source.Path() != filepath.Join(dir, later) {
		t.Fatalf("path: got %q, want the newer journal %q", source.Path(), later)
	}
}

// If every journal disappears there is nothing to follow; that is worth
// reporting rather than reading as a quiet game.
func TestAPollReportsTheJournalsHavingGone(t *testing.T) {
	dir := t.TempDir()
	name := journalName("2026-08-26T090000")
	source := sourceOver(t, dir, name, "")
	poll(t, source)

	if err := os.Remove(filepath.Join(dir, name)); err != nil {
		t.Fatalf("removing the journal: %v", err)
	}
	if _, err := source.Poll(); err == nil {
		t.Fatal("a directory emptied of journals polled without error")
	}
}

func TestAJournalFileIsRecognisedByItsName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"Journal.2026-08-26T090000.01.log", true},
		{`C:\Saved Games\Journal.2026-08-26T090000.01.log`, true},
		{"Status.json", false},
		{"Journal.2026-08-26T090000.01.txt", false},
		{"NotAJournal.log", false},
		{"", false},
	}
	for _, each := range cases {
		if got := journal.IsJournalFile(each.name); got != each.want {
			t.Fatalf("%q: got %v, want %v", each.name, got, each.want)
		}
	}
}
