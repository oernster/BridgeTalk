package pause_test

// FR-551 to FR-553: the pauses the tool found, handed out as the caller's own and refused where they
// cannot be shipped.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/pause"
)

// silence is 40 ms of silence at the model's 24 kHz, the pause FR-553 inserts.
const silence = 960

// paused is Docked's line at an index given a pause at a sample.
func paused(index, sample int) pause.Entry {
	return pause.Entry{Cue: "Docked", Index: index, Sounds: "dˈɒktkəmˈɑndə.", Digest: "0123456789abcdef", Sample: sample}
}

// doubtful is Docked's line at an index whose break was doubtful, so it has no sample.
func doubtful(index int) pause.Entry {
	entry := paused(index, 0)
	entry.Doubtful = true
	return entry
}

// bookRefused asserts that NewBook refuses as invalid, naming every fragment given.
func bookRefused(t *testing.T, inserted int, voices map[string]pause.Voice, fragments ...string) {
	t.Helper()
	_, err := pause.NewBook(inserted, modelDigest, voices)
	if !errors.Is(err, pause.ErrInvalidBook) {
		t.Fatalf("NewBook = %v, want ErrInvalidBook", err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("error %q does not name %q", err, fragment)
		}
	}
}

// bookAccepted asserts that NewBook accepts the voices given.
func bookAccepted(t *testing.T, inserted int, voices map[string]pause.Voice) {
	t.Helper()
	if _, err := pause.NewBook(inserted, modelDigest, voices); err != nil {
		t.Errorf("NewBook refused %v: %v", voices, err)
	}
}

// A book gives its silence, its model digest, its voices sorted and each voice's pauses in the order
// given; a line is found by its voice, its cue and its index.
func TestABookGivesWhatItWasMadeWith(t *testing.T) {
	t.Parallel()
	emma := pause.Voice{Style: "emma style digest", Entries: []pause.Entry{doubtful(2), paused(0, 33120)}}
	book, err := pause.NewBook(silence, modelDigest, map[string]pause.Voice{
		"bm_george": {Style: "george style digest", Entries: []pause.Entry{paused(0, 100)}},
		"bf_emma":   emma,
	})
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	if book.Silence() != silence || book.Model() != modelDigest {
		t.Errorf("silence %d model %q, want %d %q", book.Silence(), book.Model(), silence, modelDigest)
	}
	if got := book.Voices(); !slices.Equal(got, []string{"bf_emma", "bm_george"}) {
		t.Errorf("voices = %v, want bf_emma then bm_george", got)
	}
	if got, ok := book.Voice("bf_emma"); !ok || got.Style != emma.Style || !slices.Equal(got.Entries, emma.Entries) {
		t.Errorf("Voice(bf_emma) = %+v, %v; want %+v", got, ok, emma)
	}
	if _, ok := book.Voice("af_heart"); ok {
		t.Error("the book gives af_heart, which it was never given")
	}
	if got, ok := book.Entry("bf_emma", "Docked", 2); !ok || got != doubtful(2) {
		t.Errorf("Entry(bf_emma, Docked, 2) = %+v, %v; want %+v", got, ok, doubtful(2))
	}
	for _, each := range []struct {
		voice string
		index int
	}{{"bf_emma", 1}, {"af_heart", 0}} {
		if got, ok := book.Entry(each.voice, "Docked", each.index); ok {
			t.Errorf("Entry(%s, Docked, %d) = %+v, which the book was never given", each.voice, each.index, got)
		}
	}
}

// What a book hands out is the caller's own, as is what it was made from: changing either changes
// nothing in the book.
func TestABookIsTheCallersOwnCopy(t *testing.T) {
	t.Parallel()
	entries := []pause.Entry{paused(0, 5)}
	book, err := pause.NewBook(silence, modelDigest, map[string]pause.Voice{"bf_emma": {Style: "s", Entries: entries}})
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	entries[0].Sample = 9
	book.Voices()[0] = "af_heart"
	handed, _ := book.Voice("bf_emma")
	handed.Entries[0].Sample = 9
	if got, _ := book.Entry("bf_emma", "Docked", 0); got.Sample != 5 {
		t.Errorf("the entry's sample became %d, want 5", got.Sample)
	}
	if got, _ := book.Voice("bf_emma"); got.Entries[0].Sample != 5 {
		t.Errorf("the voice's entry became %+v", got.Entries[0])
	}
	if got := book.Voices(); got[0] != "bf_emma" {
		t.Errorf("voices became %v", got)
	}
}

// The empty book, shipped before the pauses tool first runs, holds no voices and no silence.
func TestTheEmptyBookIsValid(t *testing.T) {
	t.Parallel()
	book, err := pause.NewBook(0, "", nil)
	if err != nil || len(book.Voices()) != 0 || book.Silence() != 0 {
		t.Errorf("the empty book = %v voices, silence %d, %v", book.Voices(), book.Silence(), err)
	}
}

// FR-553: a line given a pause with no positive silence to insert is refused, naming the voice, the
// cue and the line; a book whose every line is doubtful needs no silence.
func TestAPauseWithNoSilenceToInsertIsRefusedNamingTheLine(t *testing.T) {
	t.Parallel()
	for _, inserted := range []int{0, -1} {
		bookRefused(t, inserted, map[string]pause.Voice{"bf_emma": {Entries: []pause.Entry{doubtful(0), paused(1, 5)}}},
			"silence", `bf_emma "Docked" line 2`)
	}
	bookAccepted(t, 0, map[string]pause.Voice{"bf_emma": {Entries: []pause.Entry{doubtful(0)}}})
}

// A line with no speech sounds or no digest is refused, every one together, each naming its line.
func TestALineWithNoSoundsOrNoDigestIsRefusedNamingIt(t *testing.T) {
	t.Parallel()
	soundless, digestless := paused(0, 5), paused(1, 5)
	soundless.Sounds, digestless.Digest = "", ""
	bookRefused(t, silence, map[string]pause.Voice{"bf_emma": {Entries: []pause.Entry{soundless, digestless}}},
		`bf_emma "Docked" line 1 has no speech sounds`, `bf_emma "Docked" line 2 has no digest`)
}

// A line placed before the first of its cue's lines is refused, naming the voice, the cue and the
// index.
func TestANegativeIndexIsRefusedNamingTheCue(t *testing.T) {
	t.Parallel()
	bookRefused(t, silence, map[string]pause.Voice{"bf_emma": {Entries: []pause.Entry{paused(-1, 5)}}},
		`bf_emma "Docked"`, "index -1")
}

// A pause at the very start of a line is refused, since its silence would come before the line
// rather than inside it; a doubtful line needs no sample.
func TestAPauseAtTheStartOfALineIsRefused(t *testing.T) {
	t.Parallel()
	bookRefused(t, silence, map[string]pause.Voice{"bf_emma": {Entries: []pause.Entry{paused(0, 0)}}},
		`bf_emma "Docked" line 1`, "sample 0")
	bookAccepted(t, silence, map[string]pause.Voice{"bf_emma": {Entries: []pause.Entry{doubtful(1), paused(2, 1)}}})
}

// The same line twice in one voice is refused, naming it; the same line in two voices is not.
func TestTheSameLineTwiceInOneVoiceIsRefused(t *testing.T) {
	t.Parallel()
	bookRefused(t, silence, map[string]pause.Voice{"bf_emma": {Entries: []pause.Entry{paused(0, 5), doubtful(0)}}},
		`bf_emma "Docked" line 1`, "twice")
	bookAccepted(t, silence, map[string]pause.Voice{
		"bf_emma":    {Entries: []pause.Entry{paused(0, 5)}},
		"am_michael": {Entries: []pause.Entry{paused(0, 7)}},
	})
}
