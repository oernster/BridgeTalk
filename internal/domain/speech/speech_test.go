package speech_test

// FR-506, FR-529 and FR-531: speech sounds turned into the numbers the model reads; the
// spellings a line gives read out of it.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// mustRead reads a line the test expects to be well formed.
func mustRead(t *testing.T, text string) speech.Line {
	t.Helper()
	line, err := speech.Read(text)
	if err != nil {
		t.Fatalf("Read(%q): %v", text, err)
	}
	return line
}

// Speech sounds become the model's numbers, with the boundary marker at each end. The numbers
// are the tokenizer file's: k is 53, ə is 83, m is 55.
func TestSpeechSoundsBecomeTheModelsNumbersBetweenBoundaries(t *testing.T) {
	got, err := speech.Tokens("kəm")
	if err != nil {
		t.Fatalf("Tokens: %v", err)
	}
	if want := []int64{0, 53, 83, 55, 0}; !slices.Equal(got, want) {
		t.Errorf("Tokens(kəm) = %v, want %v", got, want)
	}
}

// The table of symbols is handed out as the caller's own copy: changing it changes neither what
// Tokens reads nor the next copy. The structural tests hold it to the model's tokenizer file.
func TestTheSymbolTableHandedOutIsTheCallersOwn(t *testing.T) {
	table := speech.Symbols()
	if table['k'] != 53 || table['ə'] != 83 {
		t.Fatalf("the table reads k as %d and ə as %d, want 53 and 83", table['k'], table['ə'])
	}

	delete(table, 'k')
	table['ə'] = 0

	if got, err := speech.Tokens("kə"); err != nil || !slices.Equal(got, []int64{0, 53, 83, 0}) {
		t.Errorf("Tokens(kə) after changing a copy = %v, %v; want [0 53 83 0]", got, err)
	}
	if again := speech.Symbols(); again['k'] != 53 || again['ə'] != 83 {
		t.Errorf("the next copy reads k as %d and ə as %d, want 53 and 83", again['k'], again['ə'])
	}
}

// FR-506: 510 symbols is the most a line may come to. Symbols are counted, not bytes, so a
// line of a two-byte symbol is held to the same number.
func TestALineOfAtMostFiveHundredAndTenSymbolsIsAccepted(t *testing.T) {
	if _, err := speech.Tokens(strings.Repeat("ə", 510)); err != nil {
		t.Errorf("510 symbols refused: %v", err)
	}
	if _, err := speech.Tokens(strings.Repeat("ə", 511)); !errors.Is(err, speech.ErrTooLong) {
		t.Errorf("511 symbols: got %v, want ErrTooLong", err)
	}
}

// A symbol the model does not read is refused, naming it. The boundary marker is one: it is
// never a speech sound.
func TestASymbolTheModelDoesNotReadIsRefusedNamingIt(t *testing.T) {
	for sounds, symbol := range map[string]string{"kʘm": "ʘ", "a$": "$"} {
		_, err := speech.Tokens(sounds)
		if !errors.Is(err, speech.ErrUnreadableSymbol) || !strings.Contains(err.Error(), symbol) {
			t.Errorf("Tokens(%q) = %v, want ErrUnreadableSymbol naming %s", sounds, err, symbol)
		}
	}
}

// A line with no spelling is one piece of plain words that gives no speech sounds.
func TestAPlainLineIsOnePieceGivingNoSounds(t *testing.T) {
	line := mustRead(t, "Docking complete.")
	pieces := line.Pieces()
	if len(pieces) != 1 {
		t.Fatalf("got %d pieces, want 1", len(pieces))
	}
	if pieces[0].Given() || pieces[0].Sounds(machinevoice.British) != "" {
		t.Error("a plain piece gave speech sounds")
	}
	if line.Words() != "Docking complete." {
		t.Errorf("Words() = %q", line.Words())
	}
}

// FR-529: one spelling serves both accents; the line is spoken as its words alone.
func TestOneSpellingServesBothAccents(t *testing.T) {
	line := mustRead(t, "Flight [record](/ˈɹɛkɔːd/) saved.")
	given := line.Pieces()[1]
	if !given.Given() || given.Text() != "record" {
		t.Fatalf("second piece = %q, given %v; want record, given", given.Text(), given.Given())
	}
	for _, accent := range []machinevoice.Accent{machinevoice.British, machinevoice.American} {
		if got := given.Sounds(accent); got != "ˈɹɛkɔːd" {
			t.Errorf("%s sounds = %q, want ˈɹɛkɔːd", accent, got)
		}
	}
	if line.Words() != "Flight record saved." {
		t.Errorf("Words() = %q, want Flight record saved.", line.Words())
	}
}

// FR-529's acceptance: of two spellings, a British voice uses the first and an American voice
// the second.
func TestTwoSpellingsGiveBritishThenAmerican(t *testing.T) {
	given := mustRead(t, "Flight [record](/ˈɹɛkɔːd/ˈɹɛkɚd/) saved.").Pieces()[1]
	if got := given.Sounds(machinevoice.British); got != "ˈɹɛkɔːd" {
		t.Errorf("British sounds = %q, want ˈɹɛkɔːd", got)
	}
	if got := given.Sounds(machinevoice.American); got != "ˈɹɛkɚd" {
		t.Errorf("American sounds = %q, want ˈɹɛkɚd", got)
	}
}

// Spellings at either end of a line are read, leaving the plain words between them.
func TestSpellingsAtEitherEndOfALineAreRead(t *testing.T) {
	line := mustRead(t, "[Docked](/dˈɒkt/) and [secure](/sɪkjˈʊə/)")
	var texts []string
	var given []bool
	for _, piece := range line.Pieces() {
		texts = append(texts, piece.Text())
		given = append(given, piece.Given())
	}
	if want := []string{"Docked", " and ", "secure"}; !slices.Equal(texts, want) {
		t.Errorf("pieces = %q, want %q", texts, want)
	}
	if want := []bool{true, false, true}; !slices.Equal(given, want) {
		t.Errorf("given = %v, want %v", given, want)
	}
	if line.Words() != "Docked and secure" {
		t.Errorf("Words() = %q", line.Words())
	}
}

// FR-531: every broken form is refused, saying why.
func TestABrokenSpellingIsRefusedSayingWhy(t *testing.T) {
	for _, each := range []struct{ line, reason string }{
		{"Flight [record](/ˈɹɛkɔːd) saved.", "between slashes"},
		{"[record](ˈɹɛkɔːd)", "between slashes"},
		{"[record](/)", "between slashes"},
		{"Flight [record saved.", "not closed"},
		{"Flight [record](/ˈɹɛkɔːd/ saved.", "not closed"},
		{"[re[cord](/kɔːd/)", "not closed"},
		{"[re]cord](/kɔːd/)", "not closed"},
		{"[](/ˈɹɛkɔːd/)", "no word"},
		{"[flight record](/ˈɹɛkɔːd/)", "more than one word"},
		{"[record](/a/b/c/)", "more than two spellings"},
		{"[record](//)", "empty spelling"},
		{"[record](/ˈɹɛkɔːd//)", "empty spelling"},
		{"[record](/kʘ/)", "does not read"},
	} {
		_, err := speech.Read(each.line)
		if !errors.Is(err, speech.ErrBrokenSpelling) || !strings.Contains(err.Error(), each.reason) {
			t.Errorf("Read(%q) = %v, want ErrBrokenSpelling saying %q", each.line, err, each.reason)
		}
	}
}

// A symbol the model does not read makes a spelling broken; it is still that symbol's fault.
func TestAnUnreadableSymbolInASpellingIsReportedAsSuch(t *testing.T) {
	if _, err := speech.Read("[record](/kʘ/)"); !errors.Is(err, speech.ErrUnreadableSymbol) {
		t.Errorf("got %v, want ErrUnreadableSymbol as well", err)
	}
}

// The pieces handed out are a copy, so no caller can change the line.
func TestThePiecesHandedOutAreTheCallersOwn(t *testing.T) {
	line := mustRead(t, "Flight [record](/ˈɹɛkɔːd/) saved.")
	pieces := line.Pieces()
	pieces[0] = speech.Piece{}
	if line.Pieces()[0].Text() != "Flight " {
		t.Error("changing the returned pieces changed the line")
	}
}
