package config_test

// FR-555 and FR-557: the endings embedded beside the pauses, read strictly and written back the same.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
)

// twoVoicesEnding is a book of two voices: Emma's "You'll be hearing from me from here on." fades from
// sample 49200, while neither voice's "Back at the helm." fades.
func twoVoicesEnding(t *testing.T) ending.Book {
	t.Helper()
	book, err := ending.NewBook(240, "4f0c9f2a7d", map[string]ending.Voice{
		"bf_emma": {Style: "b7e1c3", Entries: []ending.Entry{
			{Cue: "Cast.Confirmed", Index: 2, Sounds: "jˌuːl biː hˈɪəɹɪŋ fɹɒm mˌiː fɹɒm hˈɪə ˈɒn.", Digest: "0123456789abcdef", Sample: 49200},
			{Cue: "InMainShip.Set", Index: 2, Sounds: "bˈak at ðə hˈɛlm.", Digest: "fedcba9876543210"},
		}},
		"am_michael": {Style: "a9d2f4", Entries: []ending.Entry{
			{Cue: "InMainShip.Set", Index: 2, Sounds: "bˈæk æt ðə hˈɛlm.", Digest: "00112233aabbccdd"},
		}},
	})
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	return book
}

// sameEndings asserts that two books hold the same fade, model digest, voices and endings.
func sameEndings(t *testing.T, got, want ending.Book) {
	t.Helper()
	if got.Fade() != want.Fade() || got.Model() != want.Model() || !slices.Equal(got.Voices(), want.Voices()) {
		t.Fatalf("got fade %d model %q voices %v, want %d %q %v",
			got.Fade(), got.Model(), got.Voices(), want.Fade(), want.Model(), want.Voices())
	}
	for _, id := range want.Voices() {
		gotVoice, _ := got.Voice(id)
		wantVoice, _ := want.Voice(id)
		if gotVoice.Style != wantVoice.Style || !slices.Equal(gotVoice.Entries, wantVoice.Entries) {
			t.Errorf("%s read back %+v, want %+v", id, gotVoice, wantVoice)
		}
	}
}

// encodedEndings writes a book, failing the test where it cannot be written.
func encodedEndings(t *testing.T, book ending.Book) string {
	t.Helper()
	written, err := config.EncodeEndings(book)
	if err != nil {
		t.Fatalf("EncodeEndings: %v", err)
	}
	return string(written)
}

// The shipped endings load as a book the rules accept.
func TestTheShippedEndingsLoad(t *testing.T) {
	t.Parallel()
	if _, err := config.LoadEndings(); err != nil {
		t.Errorf("loading the shipped endings: %v", err)
	}
}

// The empty book is written as the header alone, which reads back as the empty book. The header says
// the file is the pauses tool's to write.
func TestTheEmptyEndingsAreWrittenAsTheirHeaderAlone(t *testing.T) {
	t.Parallel()
	empty, err := ending.NewBook(0, "", nil)
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	written := encodedEndings(t, empty)
	for _, line := range strings.Split(strings.TrimSpace(written), "\n") {
		if !strings.HasPrefix(line, "#") {
			t.Errorf("the empty book wrote %q, which is not the header", line)
		}
	}
	if !strings.Contains(written, "go run ./tools/pauses") || !strings.Contains(written, "(FR-555)") {
		t.Errorf("the header does not say how the file is made:\n%s", written)
	}
	back, err := config.ParseEndings([]byte(written))
	if err != nil {
		t.Fatalf("ParseEndings: %v", err)
	}
	sameEndings(t, back, empty)
}

// Endings are written as a file that reads back as the same book; writing that book again changes
// nothing.
func TestEndingsAreWrittenAsAFileThatReadsBackTheSame(t *testing.T) {
	t.Parallel()
	book := twoVoicesEnding(t)
	written := encodedEndings(t, book)
	back, err := config.ParseEndings([]byte(written))
	if err != nil {
		t.Fatalf("ParseEndings: %v\n%s", err, written)
	}
	sameEndings(t, back, book)
	if again := encodedEndings(t, back); again != written {
		t.Errorf("writing the book read back changed the file:\n%s\nwant\n%s", again, written)
	}
}

// Voices are written sorted, each with its lines in the book's order, under the fade. A line with no
// fade is written with no sample; a fading line with its sample.
func TestALineWithNoFadeIsWrittenWithoutASample(t *testing.T) {
	t.Parallel()
	written := encodedEndings(t, twoVoicesEnding(t))
	if !strings.Contains(written, "fade = 240") {
		t.Errorf("the fade is not written:\n%s", written)
	}
	michael, emma := strings.Index(written, "[voices.am_michael]"), strings.Index(written, "[voices.bf_emma]")
	if michael < 0 || emma < michael {
		t.Errorf("am_michael at %d and bf_emma at %d, want both with am_michael first:\n%s", michael, emma, written)
	}
	var stillLines, fadingLines int
	for _, block := range strings.Split(written, "[[voices.")[1:] {
		switch {
		case strings.Contains(block, `cue = "InMainShip.Set"`):
			stillLines++
			if strings.Contains(block, "sample") {
				t.Errorf("a line with no fade is written as:\n%s", block)
			}
		case strings.Contains(block, `cue = "Cast.Confirmed"`):
			fadingLines++
			if !strings.Contains(block, "sample = 49200") {
				t.Errorf("a fading line is written as:\n%s", block)
			}
		}
	}
	if stillLines != 2 || fadingLines != 1 {
		t.Errorf("found %d lines with no fade and %d fading, want 2 and 1:\n%s", stillLines, fadingLines, written)
	}
}

// A key the endings' shape does not hold, such as a pause's doubtful, is refused rather than dropped.
func TestEndingsWithAKeyOutsideTheirShapeAreRefused(t *testing.T) {
	t.Parallel()
	raw := []byte("fade = 240\n" + lineOfDocked + "doubtful = true\n")
	if _, err := config.ParseEndings(raw); !errors.Is(err, tomlfile.ErrUnknownKey) {
		t.Errorf("got %v, want ErrUnknownKey", err)
	}
}

// Endings that are not TOML are refused.
func TestEndingsThatAreNotTomlAreRefused(t *testing.T) {
	t.Parallel()
	if _, err := config.ParseEndings([]byte("[voices\n")); err == nil {
		t.Error("a file that is not TOML was accepted")
	}
}

// Endings the book's rules refuse are refused: here a fade with no length to fade over.
func TestEndingsTheBooksRulesRefuseAreRefused(t *testing.T) {
	t.Parallel()
	raw := []byte(lineOfDocked + "sample = 5\n")
	if _, err := config.ParseEndings(raw); !errors.Is(err, ending.ErrInvalidBook) {
		t.Errorf("got %v, want ErrInvalidBook", err)
	}
}
