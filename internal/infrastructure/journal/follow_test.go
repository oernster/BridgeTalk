package journal_test

// Following the live journal through the moments it is briefly out of reach, through a name in
// either form the game has written and across the change from one journal to the next. Each case
// once replayed history or lost the present; the design forbids both (ARCHITECTURE.md, the journal).

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/journal"
)

// lines renders count journal lines of one event.
func lines(name string, count int) string {
	var out strings.Builder
	for index := 0; index < count; index++ {
		out.WriteString(line(name, fmt.Sprintf("2026-10-03T12:%02d:%02d.000Z", index/60, index%60)))
	}
	return out.String()
}

// moveAway renames a file for a moment, as a sync tool or a scanner can; it answers the way back.
func moveAway(t *testing.T, path string) func() {
	t.Helper()
	away := path + ".away"
	if err := os.Rename(path, away); err != nil {
		t.Fatalf("moving %s away: %v", path, err)
	}
	return func() {
		if err := os.Rename(away, path); err != nil {
			t.Fatalf("moving %s back: %v", path, err)
		}
	}
}

// B-4: a file that cannot be stat'ed for one read is not a file that shrank. Reading it as one put
// the offset back to 0, so the next read replayed every line in it.
func TestAJournalOutOfReachForOneReadKeepsItsPlace(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, journalName("2026-10-03T120000"), lines("Scanned", 50))
	first := journal.ReadNewBytes(path, 0, nil)

	back := moveAway(t, path)
	gone := journal.ReadNewBytes(path, first.Offset, first.Partial)
	back()

	if gone.Offset != first.Offset {
		t.Errorf("the offset went from %d to %d while the file was out of reach, want it kept", first.Offset, gone.Offset)
	}
	if again := journal.ReadNewBytes(path, gone.Offset, gone.Partial); len(again.Lines) != 0 {
		t.Errorf("%d lines were read again once it came back, want none", len(again.Lines))
	}
}

// B-4: a file that came back smaller under its own name still shrank, which a successful stat
// says: it is read from the start, with the stale carry dropped.
func TestAJournalThatCameBackSmallerIsReadFromItsStart(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.log", "a long first line\n")
	first := journal.ReadNewBytes(path, 0, nil)

	writeFile(t, dir, "a.log", "short\n")
	again := journal.ReadNewBytes(path, first.Offset, []byte("stale"))

	sameLines(t, again.Lines, "short")
	if len(again.Partial) != 0 {
		t.Errorf("the carried partial survived a rotation: %q", again.Partial)
	}
}

// B-4: the newest journal missing for one poll made the one before it the newest, read from byte
// zero; the newest was then read from byte zero again when it came back. A source never moves to
// a journal older than the one it follows.
func TestTheNewestJournalOutOfReachForOnePollReplaysNothing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, journalName("2026-10-02T120000"), lines("FSDJump", 40))
	newest := filepath.Join(dir, journalName("2026-10-03T120000"))
	source := sourceOver(t, dir, filepath.Base(newest), lines("Scanned", 30))

	back := moveAway(t, newest)
	whileGone := poll(t, source)
	back()
	afterwards := poll(t, source)

	if len(whileGone) != 0 || len(afterwards) != 0 {
		t.Errorf("%d events while the newest was gone and %d once it came back, want none", len(whileGone), len(afterwards))
	}
	if source.Path() != newest {
		t.Errorf("following %s, want %s", source.Path(), newest)
	}
}

// B-5: a name of the dashless form sorted after every dashed name, since "Journal.21" is greater
// than "Journal.20", so a journal from 2021 was followed in place of today's. The newest is the
// one whose name states the latest time, in whichever form it states it.
func TestTheNewestJournalIsTheLatestTimeItsNameStatesInEitherForm(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Journal.210301120000.01.log", "")
	today := filepath.Join(dir, "Journal.2026-10-03T120000.01.log")
	source := sourceOver(t, dir, filepath.Base(today), "")

	appendTo(t, today, lines("Scanned", 5))

	if source.Path() != today {
		t.Errorf("following %s, want today's journal", filepath.Base(source.Path()))
	}
	if heard := poll(t, source); len(heard) != 5 {
		t.Errorf("%d live events heard, want 5", len(heard))
	}
}

// B-5: within one time, the later part is the newer; a name stating no time in either form is older
// than any that does. Two names stating one time and one part are told apart by name alone, so the
// choice is the same on every poll.
func TestALaterPartIsNewerAndANameWithNoTimeIsOldest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Journal.zzz.log", "")
	writeFile(t, dir, "Journal.2026-10-03T120000.01.log", "")
	writeFile(t, dir, "Journal.2026-10-03T120000.03.log", "")
	writeFile(t, dir, "Journal.261003120000.03.log", "")
	source := sourceOver(t, dir, "Journal.2026-10-03T120000.02.log", "")

	if got := filepath.Base(source.Path()); got != "Journal.261003120000.03.log" {
		t.Errorf("following %s, want the third part", got)
	}
}

// B-11: lines written to a journal just before the next one opens, within one tick, were lost: the
// source moved to the new file without reading the rest of the old one.
func TestTheLastLinesOfAJournalAreReadBeforeTheNextIsFollowed(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, journalName("2026-10-03T120000"))
	source := sourceOver(t, dir, filepath.Base(old), "")
	poll(t, source)

	appendTo(t, old, line("Continued", "2026-10-03T12:59:59Z"))
	writeFile(t, dir, journalName("2026-10-03T130000"), line("Next0", "2026-10-03T13:00:00Z"))

	heard := poll(t, source)
	var names []string
	for _, each := range heard {
		names = append(names, each.Name())
	}
	if strings.Join(names, " ") != "Continued Next0" {
		t.Errorf("heard %v, want the old journal's last line and then the new one's first", names)
	}
}
