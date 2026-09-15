package ending_test

// FR-555 and FR-556: the endings the tool found, handed out as the caller's own and refused where they
// cannot be shipped.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/ending"
)

// fade is 30 ms at the model's 24 kHz, the fade FR-556 applies.
const fade = 720

// modelDigest is the digest the list gives the model file.
const modelDigest = "model digest"

// faded is Docked's line at an index whose ending was found at a sample; sample 0 is a line with no
// burst.
func faded(index, sample int) ending.Entry {
	return ending.Entry{Cue: "Docked", Index: index, Sounds: "dˈWn.", Digest: "0123456789abcdef", Sample: sample}
}

// refused asserts that NewBook refuses as invalid, naming every fragment given.
func refused(t *testing.T, length int, voices map[string]ending.Voice, fragments ...string) {
	t.Helper()
	_, err := ending.NewBook(length, modelDigest, voices)
	if !errors.Is(err, ending.ErrInvalidBook) {
		t.Fatalf("NewBook = %v, want ErrInvalidBook", err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("error %q does not name %q", err, fragment)
		}
	}
}

// accepted asserts that NewBook accepts the voices given.
func accepted(t *testing.T, length int, voices map[string]ending.Voice) {
	t.Helper()
	if _, err := ending.NewBook(length, modelDigest, voices); err != nil {
		t.Errorf("NewBook refused %v: %v", voices, err)
	}
}

// A book gives its fade, its model digest, its voices sorted and each voice's endings in the order
// given; a line is found by its voice, its cue and its index. It fades only where it has a sample.
func TestABookGivesWhatItWasMadeWith(t *testing.T) {
	t.Parallel()
	emma := ending.Voice{Style: "emma style digest", Entries: []ending.Entry{faded(2, 0), faded(0, 49200)}}
	book, err := ending.NewBook(fade, modelDigest, map[string]ending.Voice{
		"bm_daniel": {Style: "daniel style digest", Entries: []ending.Entry{faded(0, 0)}},
		"bf_emma":   emma,
	})
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	if book.Fade() != fade || book.Model() != modelDigest {
		t.Errorf("fade %d model %q, want %d %q", book.Fade(), book.Model(), fade, modelDigest)
	}
	if got := book.Voices(); !slices.Equal(got, []string{"bf_emma", "bm_daniel"}) {
		t.Errorf("voices = %v, want bf_emma then bm_daniel", got)
	}
	if got, ok := book.Voice("bf_emma"); !ok || got.Style != emma.Style || !slices.Equal(got.Entries, emma.Entries) {
		t.Errorf("Voice(bf_emma) = %+v, %v; want %+v", got, ok, emma)
	}
	if got, ok := book.Entry("bf_emma", "Docked", 0); !ok || got != faded(0, 49200) || !got.Fades() {
		t.Errorf("Entry(bf_emma, Docked, 0) = %+v, %v; want %+v fading", got, ok, faded(0, 49200))
	}
	if got, ok := book.Entry("bf_emma", "Docked", 2); !ok || got.Fades() {
		t.Errorf("Entry(bf_emma, Docked, 2) = %+v, %v; want it given with no fade", got, ok)
	}
	if got, ok := book.Entry("af_heart", "Docked", 0); ok {
		t.Errorf("Entry(af_heart, Docked, 0) = %+v, which the book was never given", got)
	}
}

// What a book hands out is the caller's own, as is what it was made from.
func TestABookIsTheCallersOwnCopy(t *testing.T) {
	t.Parallel()
	entries := []ending.Entry{faded(0, 5)}
	book, err := ending.NewBook(fade, modelDigest, map[string]ending.Voice{"bf_emma": {Style: "s", Entries: entries}})
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
	if got := book.Voices(); got[0] != "bf_emma" {
		t.Errorf("voices became %v", got)
	}
}

// The empty book, shipped before the pauses tool first finds the endings, holds no voices and no fade.
func TestTheEmptyBookIsValid(t *testing.T) {
	t.Parallel()
	book, err := ending.NewBook(0, "", nil)
	if err != nil || len(book.Voices()) != 0 || book.Fade() != 0 {
		t.Errorf("the empty book = %v voices, fade %d, %v", book.Voices(), book.Fade(), err)
	}
}

// FR-556: a line given a fade with no positive length to fade over is refused, naming the voice, the
// cue and the line; a book with no line fading needs no length.
func TestAFadeWithNoLengthIsRefusedNamingTheLine(t *testing.T) {
	t.Parallel()
	for _, length := range []int{0, -1} {
		refused(t, length, map[string]ending.Voice{"bf_emma": {Entries: []ending.Entry{faded(0, 0), faded(1, 5)}}},
			"fade", `bf_emma "Docked" line 2`)
	}
	accepted(t, 0, map[string]ending.Voice{"bf_emma": {Entries: []ending.Entry{faded(0, 0)}}})
}

// FR-555: an ending that cannot be shipped is refused with the rule broken, every one together, each
// naming its line: no speech sounds, no digest, a place before the first line, a sample before the
// first and the same line twice. The same line in two voices is not refused.
func TestAnEndingThatCannotBeShippedIsRefusedNamingItsLine(t *testing.T) {
	t.Parallel()
	soundless, digestless, early := faded(0, 5), faded(1, 5), faded(2, -1)
	soundless.Sounds, digestless.Digest = "", ""
	refused(t, fade, map[string]ending.Voice{"bf_emma": {Entries: []ending.Entry{soundless, digestless, early}}},
		`bf_emma "Docked" line 1 has no speech sounds`, `bf_emma "Docked" line 2 has no digest`,
		`bf_emma "Docked" line 3`, "sample -1", "(FR-555)")
	refused(t, fade, map[string]ending.Voice{"bf_emma": {Entries: []ending.Entry{faded(-1, 5)}}}, `bf_emma "Docked"`, "index -1")
	refused(t, fade, map[string]ending.Voice{"bf_emma": {Entries: []ending.Entry{faded(0, 5), faded(0, 0)}}},
		`bf_emma "Docked" line 1`, "twice")
	accepted(t, fade, map[string]ending.Voice{
		"bf_emma":    {Entries: []ending.Entry{faded(0, 5)}},
		"am_michael": {Entries: []ending.Entry{faded(0, 7)}},
	})
}
