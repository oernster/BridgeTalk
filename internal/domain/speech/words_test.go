package speech_test

// FR-529 and FR-549: a word's speech sounds given once for the whole script, refused where they
// cannot be read and spelled into every line holding the word whole.

import (
	"errors"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// british and american are commander's speech sounds as the shipped table gives them.
const (
	british  = "kəmˈɑndə"
	american = "kəmˈændəɹ"
)

// mustWords builds a table of words the test expects to be well formed.
func mustWords(t *testing.T, spellings map[string][]string) speech.Words {
	t.Helper()
	words, err := speech.NewWords(spellings)
	if err != nil {
		t.Fatalf("NewWords(%v): %v", spellings, err)
	}
	return words
}

// commander is a table giving commander a British and an American spelling.
func commander(t *testing.T) speech.Words {
	t.Helper()
	return mustWords(t, map[string][]string{"commander": {british, american}})
}

// FR-549: of a word's two spellings, a British voice uses the first and an American voice the
// second; a word the table does not hold has none.
func TestAWordsTwoSpellingsGiveBritishThenAmerican(t *testing.T) {
	words := commander(t)
	for accent, want := range map[machinevoice.Accent]string{machinevoice.British: british, machinevoice.American: american} {
		if got, ok := words.Sounds("commander", accent); !ok || got != want {
			t.Errorf("%s sounds = %q, %v; want %q", accent, got, ok, want)
		}
	}
	if _, ok := words.Sounds("pilot", machinevoice.British); ok {
		t.Error("pilot has sounds the table never gave it")
	}
}

// FR-549: one spelling of a word serves both accents, as it does in a line (FR-529).
func TestOneSpellingOfAWordServesBothAccents(t *testing.T) {
	words := mustWords(t, map[string][]string{"record": {"ˈɹɛkɔːd"}})
	for _, accent := range machinevoice.Accents() {
		if got, _ := words.Sounds("record", accent); got != "ˈɹɛkɔːd" {
			t.Errorf("%s sounds = %q, want ˈɹɛkɔːd", accent, got)
		}
	}
}

// FR-549: every broken entry is refused naming its word and saying why, reported together.
func TestABrokenWordIsRefusedNamingItAndSayingWhy(t *testing.T) {
	for _, each := range []struct {
		word      string
		spellings []string
		reason    string
	}{
		{"", []string{british}, "no word"},
		{"the commander", []string{british}, "more than one word"},
		{"commander", nil, "no spelling"},
		{"commander", []string{british, american, british}, "more than two spellings"},
		{"commander", []string{british, ""}, "empty spelling"},
		{"commander", []string{"kʘm"}, "does not read"},
		{"commander", []string{"kəm/ˈɑndə"}, "bracket or slash"},
		{"com]mander", []string{british}, "bracket or slash"},
	} {
		_, err := speech.NewWords(map[string][]string{each.word: each.spellings})
		if !errors.Is(err, speech.ErrBrokenSpelling) || !strings.Contains(err.Error(), each.reason) ||
			!strings.Contains(err.Error(), `word "`+each.word+`"`) {
			t.Errorf("NewWords(%q: %q) = %v, want ErrBrokenSpelling naming the word and saying %q",
				each.word, each.spellings, err, each.reason)
		}
	}
	_, err := speech.NewWords(map[string][]string{"commander": nil, "pilot": {""}})
	for _, word := range []string{`"commander"`, `"pilot"`} {
		if err == nil || !strings.Contains(err.Error(), word) {
			t.Errorf("error %v does not name %s", err, word)
		}
	}
}

// FR-549's acceptance: every whole occurrence of a table word in a line's plain words is written
// with that accent's spelling, wherever it stands.
func TestATableWordIsSpelledWhereverItStandsWhole(t *testing.T) {
	line := mustRead(t, "commander, Sold, commander. A commander's ship, commander")
	words := commander(t)
	for accent, sounds := range map[machinevoice.Accent]string{machinevoice.British: british, machinevoice.American: american} {
		spelled := "[commander](/" + sounds + "/)"
		want := spelled + ", Sold, " + spelled + ". A " + spelled + "'s ship, " + spelled
		if got := words.Spell(line, accent); got != want {
			t.Errorf("Spell(%s) = %q, want %q", accent, got, want)
		}
	}
}

// FR-549: a word is spelled only whole and in its exact case; a letter or a digit against it on
// either side, even one inside a spelling the line gives, leaves it as it stands.
func TestATableWordIsSpelledOnlyWholeAndInItsExactCase(t *testing.T) {
	words := commander(t)
	for _, text := range []string{
		"Commander.", "COMMANDER.", "commanders.", "decommander.", "commander2.", "2commander.",
		"[record](/ˈɹɛkɔːd/)commander.", "commander[s](/z/).",
	} {
		line := mustRead(t, text)
		if got, want := words.Spell(line, machinevoice.British), line.ForAccent(machinevoice.British); got != want {
			t.Errorf("Spell(%q) = %q, want it unchanged as %q", text, got, want)
		}
	}
}

// FR-549: a word the line already spells keeps the line's own spelling (FR-529).
func TestAWordTheLineSpellsKeepsTheLinesOwnSpelling(t *testing.T) {
	line := mustRead(t, "[commander](/kəmˈɑːndə/), commander.")
	want := "[commander](/kəmˈɑːndə/), [commander](/" + british + "/)."
	if got := commander(t).Spell(line, machinevoice.British); got != want {
		t.Errorf("Spell = %q, want %q", got, want)
	}
}

// Where one table word begins another, the longer is spelled where it stands whole.
func TestTheLongerOfTwoTableWordsIsSpelledFirst(t *testing.T) {
	words := mustWords(t, map[string][]string{"star": {"stˈɑː"}, "star-port": {"stˈɑːpɔːt"}})
	line := mustRead(t, "A star-port by a star.")
	if got, want := words.Spell(line, machinevoice.British), "A [star-port](/stˈɑːpɔːt/) by a [star](/stˈɑː/)."; got != want {
		t.Errorf("Spell = %q, want %q", got, want)
	}
}

// An empty table spells a line as ForAccent writes it.
func TestAnEmptyTableSpellsALineAsItIsWrittenForTheAccent(t *testing.T) {
	line := mustRead(t, "Flight [record](/ˈɹɛkɔːd/ˈɹɛkɚd/) saved, commander.")
	if got, want := (speech.Words{}).Spell(line, machinevoice.American), "Flight [record](/ˈɹɛkɚd/) saved, commander."; got != want {
		t.Errorf("Spell = %q, want %q", got, want)
	}
}
