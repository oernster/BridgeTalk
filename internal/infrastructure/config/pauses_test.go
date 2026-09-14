package config_test

// FR-551 and FR-554: the pauses embedded beside the saved speech sounds, read strictly and written
// back the same.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/pause"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
)

// twoVoices is a book of two voices: each gives BreathableAtmosphere.Set's second line a pause, while
// Emma's break in a line of Docked was doubtful, given the sample asked for, which means nothing.
func twoVoices(t *testing.T, doubtfulSample int) pause.Book {
	t.Helper()
	book, err := pause.NewBook(960, "4f0c9f2a7d", map[string]pause.Voice{
		"bf_emma": {Style: "b7e1c3", Entries: []pause.Entry{
			{Cue: "BreathableAtmosphere.Set", Index: 1, Sounds: "bɹˈiːðəbᵊl ˈatməsfɪəkəmˈɑndə.", Digest: "0123456789abcdef", Sample: 33120},
			{Cue: "Docked", Index: 2, Sounds: "dˈɒkt and sɪkjˈʊəkəmˈɑndə.", Digest: "fedcba9876543210", Sample: doubtfulSample, Doubtful: true},
		}},
		"am_michael": {Style: "a9d2f4", Entries: []pause.Entry{
			{Cue: "BreathableAtmosphere.Set", Index: 1, Sounds: "bɹˈiðəbᵊl ˈætməsfˌɪɹkəmˈændəɹ.", Digest: "00112233aabbccdd", Sample: 30240},
		}},
	})
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	return book
}

// sameBook asserts that two books hold the same silence, model digest, voices and pauses.
func sameBook(t *testing.T, got, want pause.Book) {
	t.Helper()
	if got.Silence() != want.Silence() || got.Model() != want.Model() || !slices.Equal(got.Voices(), want.Voices()) {
		t.Fatalf("got silence %d model %q voices %v, want %d %q %v",
			got.Silence(), got.Model(), got.Voices(), want.Silence(), want.Model(), want.Voices())
	}
	for _, id := range want.Voices() {
		gotVoice, _ := got.Voice(id)
		wantVoice, _ := want.Voice(id)
		if gotVoice.Style != wantVoice.Style || !slices.Equal(gotVoice.Entries, wantVoice.Entries) {
			t.Errorf("%s read back %+v, want %+v", id, gotVoice, wantVoice)
		}
	}
}

// encoded writes a book, failing the test where it cannot be written.
func encoded(t *testing.T, book pause.Book) string {
	t.Helper()
	written, err := config.EncodePauses(book)
	if err != nil {
		t.Fatalf("EncodePauses: %v", err)
	}
	return string(written)
}

// The shipped pauses load as a book the rules accept.
func TestTheShippedPausesLoad(t *testing.T) {
	t.Parallel()
	if _, err := config.LoadPauses(); err != nil {
		t.Errorf("loading the shipped pauses: %v", err)
	}
}

// The empty book is written as the header alone, which reads back as the empty book: the file shipped
// before the pauses tool first runs. The header says the file is the tool's to write.
func TestTheEmptyBookIsWrittenAsItsHeaderAlone(t *testing.T) {
	t.Parallel()
	empty, err := pause.NewBook(0, "", nil)
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	written := encoded(t, empty)
	for _, line := range strings.Split(strings.TrimSpace(written), "\n") {
		if !strings.HasPrefix(line, "#") {
			t.Errorf("the empty book wrote %q, which is not the header", line)
		}
	}
	if !strings.Contains(written, "go run ./tools/pauses") || !strings.Contains(written, "(FR-551)") {
		t.Errorf("the header does not say how the file is made:\n%s", written)
	}
	back, err := config.ParsePauses([]byte(written))
	if err != nil {
		t.Fatalf("ParsePauses: %v", err)
	}
	sameBook(t, back, empty)
}

// Pauses are written as a file that reads back as the same book; writing that book again changes
// nothing, so a run of the tool finding the same pauses leaves the file as it was.
func TestPausesAreWrittenAsAFileThatReadsBackTheSame(t *testing.T) {
	t.Parallel()
	book := twoVoices(t, 0)
	written := encoded(t, book)
	back, err := config.ParsePauses([]byte(written))
	if err != nil {
		t.Fatalf("ParsePauses: %v\n%s", err, written)
	}
	sameBook(t, back, book)
	if again := encoded(t, back); again != written {
		t.Errorf("writing the book read back changed the file:\n%s\nwant\n%s", again, written)
	}
}

// Voices are written sorted, each with its lines in the book's order. A doubtful line is written with
// doubtful set and no sample, whatever sample it was given; a paused line with its sample and no
// doubtful.
func TestADoubtfulLineIsWrittenWithoutASampleAndAPausedLineWithoutDoubtful(t *testing.T) {
	t.Parallel()
	written := encoded(t, twoVoices(t, 41))
	michael, emma := strings.Index(written, "[voices.am_michael]"), strings.Index(written, "[voices.bf_emma]")
	if michael < 0 || emma < michael {
		t.Errorf("am_michael at %d and bf_emma at %d, want both with am_michael first:\n%s", michael, emma, written)
	}
	if strings.Index(written, "33120") > strings.Index(written, "fedcba9876543210") {
		t.Errorf("bf_emma's lines are not in the book's order:\n%s", written)
	}
	var doubtfulLines, pausedLines int
	for _, block := range strings.Split(written, "[[voices.")[1:] {
		switch {
		case strings.Contains(block, `cue = "Docked"`):
			doubtfulLines++
			if !strings.Contains(block, "doubtful = true") || strings.Contains(block, "sample") {
				t.Errorf("a doubtful line is written as:\n%s", block)
			}
		case strings.Contains(block, `cue = "BreathableAtmosphere.Set"`):
			pausedLines++
			if !strings.Contains(block, "sample = ") || strings.Contains(block, "doubtful") {
				t.Errorf("a paused line is written as:\n%s", block)
			}
		}
	}
	if doubtfulLines != 1 || pausedLines != 2 {
		t.Errorf("found %d doubtful and %d paused lines, want 1 and 2:\n%s", doubtfulLines, pausedLines, written)
	}
}

// lineOfDocked is pauses for one line of Docked, the last key given by the caller.
const lineOfDocked = "[voices.bf_emma]\nstyle = \"a\"\n[[voices.bf_emma.lines]]\ncue = \"Docked\"\nindex = 0\n" +
	"sounds = \"dˈɒkt\"\ndigest = \"0123456789abcdef\"\n"

// A key the pauses' shape does not hold is refused rather than dropped in silence.
func TestPausesWithAKeyOutsideTheirShapeAreRefused(t *testing.T) {
	t.Parallel()
	raw := []byte("silence = 960\n" + lineOfDocked + "sampel = 5\n")
	if _, err := config.ParsePauses(raw); !errors.Is(err, tomlfile.ErrUnknownKey) {
		t.Errorf("got %v, want ErrUnknownKey", err)
	}
}

// Pauses that are not TOML are refused.
func TestPausesThatAreNotTomlAreRefused(t *testing.T) {
	t.Parallel()
	if _, err := config.ParsePauses([]byte("[voices\n")); err == nil {
		t.Error("a file that is not TOML was accepted")
	}
}

// Pauses the book's rules refuse are refused: here a pause with no silence to insert.
func TestPausesTheBooksRulesRefuseAreRefused(t *testing.T) {
	t.Parallel()
	raw := []byte(lineOfDocked + "sample = 5\n")
	if _, err := config.ParsePauses(raw); !errors.Is(err, pause.ErrInvalidBook) {
		t.Errorf("got %v, want ErrInvalidBook", err)
	}
}
