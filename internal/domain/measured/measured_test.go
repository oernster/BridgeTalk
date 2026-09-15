package measured_test

// FR-551, FR-554, FR-555 and FR-557: the rules every measured book shares, held over a kind of entry
// made for the test so they are proved where they live rather than through pauses or endings alone.

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/measured"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
)

var (
	errInvalid = errors.New("invalid marks")
	errStale   = errors.New("stale marks")
)

// kind is the test's kind of entry: a mark on a line.
var kind = measured.Kind{
	Invalid: errInvalid, Stale: errStale, Found: "FR-found", Checked: "FR-checked",
	Plural: "marks", Missing: "is wanted with no mark", MeasuredIn: "was marked in speech sounds", Gone: "has a mark but is not wanted",
}

// mark is the test's entry: a line with a sample the kind's rule refuses below zero.
type mark struct {
	cue    cue.ID
	index  int
	sounds string
	digest string
	sample int
}

// Measured answers the line the mark was made in.
func (m mark) Measured() measured.Line {
	return measured.Line{Cue: m.cue, Index: m.index, Sounds: m.sounds, Digest: m.digest}
}

// markAt is Docked's line at an index marked at a sample.
func markAt(index, sample int) mark {
	return mark{cue: "Docked", index: index, sounds: "dˈWn.", digest: "0123456789abcdef", sample: sample}
}

// sampleRule refuses a sample below zero, naming the line.
func sampleRule(name string, entry mark) error {
	if entry.sample < 0 {
		return fmt.Errorf("%w: %s is marked at sample %d (FR-found)", errInvalid, name, entry.sample)
	}
	return nil
}

// bookOf builds a book of the voices given, failing the test where it is refused.
func bookOf(t *testing.T, voices map[string]measured.Voice[mark]) measured.Book[mark] {
	t.Helper()
	book, err := measured.NewBook(kind, "model digest", voices, sampleRule)
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	return book
}

// A book gives its model digest, its voices sorted and each voice's entries in the order given; a line
// is found by its voice, its cue and its index. A line never given answers the zero entry.
func TestABookGivesWhatItWasMadeWith(t *testing.T) {
	t.Parallel()
	emma := measured.Voice[mark]{Style: "emma style", Entries: []mark{markAt(2, 0), markAt(0, 5)}}
	book := bookOf(t, map[string]measured.Voice[mark]{"bm_george": {Style: "george style", Entries: []mark{markAt(0, 1)}}, "bf_emma": emma})
	if book.Model() != "model digest" || !slices.Equal(book.Voices(), []string{"bf_emma", "bm_george"}) {
		t.Errorf("model %q voices %v, want the model digest and bf_emma then bm_george", book.Model(), book.Voices())
	}
	if got, ok := book.Voice("bf_emma"); !ok || got.Style != emma.Style || !slices.Equal(got.Entries, emma.Entries) {
		t.Errorf("Voice(bf_emma) = %+v, %v; want %+v", got, ok, emma)
	}
	if got, ok := book.Entry("bf_emma", "Docked", 0); !ok || got != markAt(0, 5) {
		t.Errorf("Entry(bf_emma, Docked, 0) = %+v, %v; want %+v", got, ok, markAt(0, 5))
	}
	for _, each := range []struct {
		voice string
		index int
	}{{"bf_emma", 1}, {"af_heart", 0}} {
		if got, ok := book.Entry(each.voice, "Docked", each.index); ok || got != (mark{}) {
			t.Errorf("Entry(%s, Docked, %d) = %+v, %v; want the zero entry, not given", each.voice, each.index, got, ok)
		}
	}
}

// What a book hands out is the caller's own, as is what it was made from.
func TestABookIsTheCallersOwnCopy(t *testing.T) {
	t.Parallel()
	entries := []mark{markAt(0, 5)}
	book := bookOf(t, map[string]measured.Voice[mark]{"bf_emma": {Entries: entries}})
	entries[0].sample = 9
	book.Voices()[0] = "af_heart"
	handed, _ := book.Voice("bf_emma")
	handed.Entries[0].sample = 9
	if got, _ := book.Entry("bf_emma", "Docked", 0); got.sample != 5 || book.Voices()[0] != "bf_emma" {
		t.Errorf("the book became %+v with voices %v", got, book.Voices())
	}
}

// Every problem is refused together, each naming its line and citing the kind's requirement: no speech
// sounds, no digest, a sample the kind's rule refuses and the same line twice. A line at a negative
// index is refused for that alone.
func TestEveryProblemIsRefusedTogetherNamingItsLine(t *testing.T) {
	t.Parallel()
	soundless, digestless, early, twice := markAt(0, 1), markAt(1, 1), markAt(2, -1), markAt(2, 0)
	soundless.sounds, digestless.digest = "", ""
	before := markAt(-1, 1)
	before.sounds = ""
	_, err := measured.NewBook(kind, "", map[string]measured.Voice[mark]{"bf_emma": {Entries: []mark{soundless, digestless, early, twice, before}}}, sampleRule)
	if !errors.Is(err, errInvalid) {
		t.Fatalf("NewBook = %v, want errInvalid", err)
	}
	for _, fragment := range []string{
		`bf_emma "Docked" line 1 has no speech sounds (FR-found)`, `bf_emma "Docked" line 2 has no digest (FR-found)`,
		`bf_emma "Docked" line 3 is marked at sample -1`, `bf_emma "Docked" line 3 is given twice (FR-found)`,
		`bf_emma "Docked" has a line at index -1, before its first (FR-found)`,
	} {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("%q does not hold %q", err, fragment)
		}
	}
	if strings.Count(err.Error(), "no speech sounds") != 1 {
		t.Errorf("%q refuses the line at index -1 for more than its index", err)
	}
}

// The empty book is valid.
func TestTheEmptyBookIsValid(t *testing.T) {
	t.Parallel()
	if book := bookOf(t, nil); len(book.Voices()) != 0 || book.Model() != "model digest" {
		t.Errorf("the empty book = %v voices, model %q", book.Voices(), book.Model())
	}
}

// FirstWhere names the first line whose entry matches, voices sorted then each voice's entries in the
// order given; with none matching it answers nothing.
func TestFirstWhereNamesTheFirstMatchingLine(t *testing.T) {
	t.Parallel()
	voices := map[string]measured.Voice[mark]{
		"bm_george": {Entries: []mark{markAt(0, 7)}},
		"bf_emma":   {Entries: []mark{markAt(2, 0), markAt(1, 7), markAt(0, 7)}},
	}
	marked := func(entry mark) bool { return entry.sample > 0 }
	if name, ok := measured.FirstWhere(voices, marked); !ok || name != `bf_emma "Docked" line 2` {
		t.Errorf("FirstWhere = %q, %v; want bf_emma's Docked line 2", name, ok)
	}
	if name, ok := measured.FirstWhere(voices, func(mark) bool { return false }); ok || name != "" {
		t.Errorf("FirstWhere with nothing matching = %q, %v", name, ok)
	}
}

// wanted is the lines each voice should have a mark for in either accent: Docked's lines 1 and 3.
func wanted(machinevoice.Accent) []script.SavedLine {
	return []script.SavedLine{{Cue: "Docked", Index: 0, Sounds: "dˈWn."}, {Cue: "Docked", Index: 2, Sounds: "ɡˈɒn."}}
}

// checked checks the voices given, bf_emma then am_michael, against a script of Docked, answering what
// each problem says; every problem must be errStale citing the kind's requirement.
func checked(t *testing.T, voices map[string]measured.Voice[mark], model string, styles map[string]string) []string {
	t.Helper()
	voiced, err := scripttest.Build(map[string]script.Saved{"Docked": {
		Lines: []string{"Down.", "Docked.", "Gone."}, British: []string{"dˈWn.", "dˈɒkt.", "ɡˈɒn."}, American: []string{"dˈWn.", "dˈɑkt.", "ɡˈɔn."},
	}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var cast []machinevoice.Voice
	for _, id := range []string{"bf_emma", "am_michael"} {
		voice, err := machinevoice.Parse(id)
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		cast = append(cast, voice)
	}
	var words []string
	for _, problem := range measured.Check(kind, bookOf(t, voices), cast, voiced, wanted, model, styles) {
		if !errors.Is(problem, errStale) || !strings.HasSuffix(problem.Error(), "(FR-checked)") {
			t.Errorf("%v is not errStale citing FR-checked", problem)
		}
		words = append(words, problem.Error())
	}
	return words
}

// fresh is a mark for every wanted line for both voices, found with the listed styles.
func fresh() map[string]measured.Voice[mark] {
	return map[string]measured.Voice[mark]{
		"bf_emma":    {Style: "emma style", Entries: []mark{markAt(0, 1), {cue: "Docked", index: 2, sounds: "ɡˈɒn.", digest: "d"}}},
		"am_michael": {Style: "michael style", Entries: []mark{markAt(0, 1), {cue: "Docked", index: 2, sounds: "ɡˈɒn.", digest: "d"}}},
	}
}

// listedStyles are the digests the list gives each voice's style file.
var listedStyles = map[string]string{"bf_emma": "emma style", "am_michael": "michael style"}

// Fresh marks are not stale. Stale ones are named in one order, each in the kind's words: the model,
// then each voice in the order given with its style file, its wanted lines with no mark or marked in
// other sounds, then its marks for lines no longer wanted with their text where the script has it.
func TestStaleEntriesAreNamedInOneOrderInTheKindsWords(t *testing.T) {
	t.Parallel()
	if words := checked(t, fresh(), "model digest", listedStyles); len(words) != 0 {
		t.Errorf("fresh marks are stale: %q", words)
	}
	voices := fresh()
	emma := voices["bf_emma"]
	emma.Style = "old emma style"
	emma.Entries = []mark{{cue: "Docked", index: 2, sounds: "ɡˈɒt.", digest: "d"}, markAt(1, 0), {cue: "Launched", index: 0, sounds: "l", digest: "d"}}
	voices["bf_emma"] = emma
	delete(voices, "am_michael")
	want := []string{
		`the marks were found with the model digest "model digest" where the list gives "new model digest"`,
		`bf_emma's marks were found with the style digest "old emma style" where the list gives "emma style"`,
		`bf_emma "Docked" line 1 "Down." is wanted with no mark`,
		`bf_emma "Docked" line 3 "Gone." was marked in speech sounds "ɡˈɒt." where its saved sounds are now "ɡˈɒn."`,
		`bf_emma "Docked" line 2 "Docked." has a mark but is not wanted`,
		`bf_emma "Launched" line 1 has a mark but is not wanted`,
		"am_michael has no marks",
	}
	for range 10 {
		words := checked(t, voices, "new model digest", listedStyles)
		if len(words) != len(want) {
			t.Fatalf("got %d problems %q, want %d", len(words), words, len(want))
		}
		for at, fragment := range want {
			if !strings.Contains(words[at], fragment) {
				t.Errorf("problem %d %q does not hold %q", at+1, words[at], fragment)
			}
		}
	}
}
