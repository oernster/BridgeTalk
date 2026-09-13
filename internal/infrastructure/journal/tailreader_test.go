package journal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/journal"
)

// readBuffer is the size the reader drains a handle in. A file at or beyond it is
// what forces the read loop round more than once, so the tests derive their sizes
// from this rather than restating a number the reader owns.
const readBuffer = 64 * 1024

// writeFile writes a file into dir and returns its path.
func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing %q: %v", path, err)
	}
	return path
}

// appendTo adds bytes to an existing file, as the game does mid-session.
func appendTo(t *testing.T, path, body string) {
	t.Helper()
	handle, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("opening %q to append: %v", path, err)
	}
	defer func() { _ = handle.Close() }()
	if _, err := handle.WriteString(body); err != nil {
		t.Fatalf("appending to %q: %v", path, err)
	}
}

// sameLines asserts the reader returned exactly these lines, in order.
func sameLines(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d lines %v, want %d %v", len(got), got, len(want), want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("line %d: got %q, want %q", index, got[index], want[index])
		}
	}
}

func TestTheSizeOfAMissingFileReadsAsZero(t *testing.T) {
	if size := journal.FileSize(filepath.Join(t.TempDir(), "absent.log")); size != 0 {
		t.Fatalf("got %d, want 0", size)
	}
}

func TestTheSizeOfAFileIsItsLength(t *testing.T) {
	path := writeFile(t, t.TempDir(), "a.log", "twelve bytes")
	if size := journal.FileSize(path); size != int64(len("twelve bytes")) {
		t.Fatalf("got %d, want %d", size, len("twelve bytes"))
	}
}

func TestAFreshReadReturnsEveryCompleteLine(t *testing.T) {
	path := writeFile(t, t.TempDir(), "a.log", "first\nsecond\n")

	result := journal.ReadNewBytes(path, 0, nil)
	sameLines(t, result.Lines, "first", "second")
	if result.Offset != int64(len("first\nsecond\n")) {
		t.Fatalf("offset: got %d", result.Offset)
	}
	if len(result.Partial) != 0 {
		t.Fatalf("partial: got %q, want nothing carried", result.Partial)
	}
}

func TestReadingResumesFromTheRecordedOffset(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.log", "first\n")

	first := journal.ReadNewBytes(path, 0, nil)
	sameLines(t, first.Lines, "first")

	appendTo(t, path, "second\n")
	second := journal.ReadNewBytes(path, first.Offset, first.Partial)
	sameLines(t, second.Lines, "second")
}

// The game appends a line at a time, so a read can land between the line and its
// newline. Carrying the remainder is what guarantees no event is lost or doubled.
func TestALineCaughtMidWriteIsCarriedAndCompletedNextPass(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.log", "complete\npar")

	first := journal.ReadNewBytes(path, 0, nil)
	sameLines(t, first.Lines, "complete")
	if string(first.Partial) != "par" {
		t.Fatalf("partial: got %q, want %q", first.Partial, "par")
	}

	appendTo(t, path, "tial\n")
	second := journal.ReadNewBytes(path, first.Offset, first.Partial)
	sameLines(t, second.Lines, "partial")
	if len(second.Partial) != 0 {
		t.Fatalf("partial after completion: got %q", second.Partial)
	}
}

// A file smaller than the recorded offset was rotated or truncated. Resuming from
// the old offset would seek past the end and read nothing ever again.
func TestAFileThatShrankIsReadFromTheBeginningAgain(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.log", "a long first line\n")

	first := journal.ReadNewBytes(path, 0, nil)
	sameLines(t, first.Lines, "a long first line")

	writeFile(t, dir, "a.log", "short\n")
	second := journal.ReadNewBytes(path, first.Offset, []byte("stale"))
	sameLines(t, second.Lines, "short")
	if len(second.Partial) != 0 {
		t.Fatalf("the carried partial survived a rotation: %q", second.Partial)
	}
}

// A file that has gone is a file that shrank to nothing, so it is a rotation and the
// position resets. Holding the old offset would leave the reader seeking past the end
// of whatever file appears next under that name.
func TestAFileThatVanishedIsTreatedAsARotation(t *testing.T) {
	result := journal.ReadNewBytes(filepath.Join(t.TempDir(), "absent.log"), 42, []byte("held"))

	if len(result.Lines) != 0 {
		t.Fatalf("got lines from a missing file: %v", result.Lines)
	}
	if result.Offset != 0 {
		t.Fatalf("offset: got %d, want a reset to 0", result.Offset)
	}
	if len(result.Partial) != 0 {
		t.Fatalf("partial: got %q, want the stale carry dropped", result.Partial)
	}
}

// A handle that opens but will not read is the same situation as one that will not
// open: say nothing and keep the position. A directory is the portable way to get a
// handle whose read fails.
func TestAHandleThatOpensButWillNotReadIsSilent(t *testing.T) {
	dir := t.TempDir()

	result := journal.ReadNewBytes(dir, 0, nil)
	if len(result.Lines) != 0 {
		t.Fatalf("got lines from a directory: %v", result.Lines)
	}
}

// A negative offset is not a position any file can be read from. It cannot arise
// from the source itself, which only ever records a size, so this guards the
// exported reader against a caller that has lost track of where it was.
func TestAnImpossibleOffsetYieldsNothingRatherThanPanicking(t *testing.T) {
	path := writeFile(t, t.TempDir(), "a.log", "first\n")

	result := journal.ReadNewBytes(path, -1, []byte("held"))
	if len(result.Lines) != 0 {
		t.Fatalf("got lines from an impossible offset: %v", result.Lines)
	}
	if result.Offset != -1 {
		t.Fatalf("offset: got %d, want it left as given at -1", result.Offset)
	}
	if string(result.Partial) != "held" {
		t.Fatalf("partial: got %q, want it carried unchanged", result.Partial)
	}
}

// The read loop drains the handle in fixed-size chunks. A file larger than one chunk
// is what proves it goes round again rather than truncating at the buffer.
func TestAFileLargerThanOneReadBufferIsDrainedWhole(t *testing.T) {
	line := strings.Repeat("x", 99) + "\n"
	repeats := (readBuffer / len(line)) + 10
	path := writeFile(t, t.TempDir(), "big.log", strings.Repeat(line, repeats))

	result := journal.ReadNewBytes(path, 0, nil)
	if len(result.Lines) != repeats {
		t.Fatalf("got %d lines, want %d", len(result.Lines), repeats)
	}
}

// A file whose length is an exact multiple of the read buffer takes one extra turn
// of the loop that returns nothing at all, which is the end-of-file path.
func TestAFileEndingExactlyOnABufferBoundaryIsDrainedWhole(t *testing.T) {
	body := strings.Repeat("y", readBuffer-1) + "\n"
	path := writeFile(t, t.TempDir(), "exact.log", body)

	result := journal.ReadNewBytes(path, 0, nil)
	sameLines(t, result.Lines, strings.Repeat("y", readBuffer-1))
	if result.Offset != int64(readBuffer) {
		t.Fatalf("offset: got %d, want %d", result.Offset, readBuffer)
	}
}

// Blank lines and whitespace-only lines are structure, not events. Passing them on
// would make the parser reject them one layer further up for no benefit.
func TestBlankAndWhitespaceLinesAreDropped(t *testing.T) {
	path := writeFile(t, t.TempDir(), "a.log", "first\n\n   \n\t\nsecond\n")

	result := journal.ReadNewBytes(path, 0, nil)
	sameLines(t, result.Lines, "first", "second")
}

// Each returned line is trimmed. The game writes CRLF on Windows, so without this
// every line would arrive with a carriage return glued to its closing brace.
func TestEachLineIsTrimmedOfSurroundingWhitespace(t *testing.T) {
	path := writeFile(t, t.TempDir(), "a.log", "  spaced  \r\n")

	result := journal.ReadNewBytes(path, 0, nil)
	sameLines(t, result.Lines, "spaced")
}

func TestAnEmptyFileYieldsNothingAndHoldsAtZero(t *testing.T) {
	path := writeFile(t, t.TempDir(), "a.log", "")

	result := journal.ReadNewBytes(path, 0, nil)
	if len(result.Lines) != 0 {
		t.Fatalf("got lines from an empty file: %v", result.Lines)
	}
	if result.Offset != 0 {
		t.Fatalf("offset: got %d, want 0", result.Offset)
	}
}
