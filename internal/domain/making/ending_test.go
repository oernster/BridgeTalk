package making_test

// FR-513 and FR-556: a line carries its voice's ending, which changes the line's key only where it
// gives a fade; a line with a pause and a fade is keyed apart from a line with either alone.

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/pause"
)

// noEndings is a book giving no voice an ending.
var noEndings = ending.Book{}

// bookFade is the fade the books below apply.
const bookFade = 240

// fadedAt is an ending for Docked's second line, whose British sounds are "bɪ", fading from sample 7.
var fadedAt = ending.Entry{Cue: "Docked", Index: 1, Sounds: "bɪ", Digest: foundIn, Sample: 7}

// endingsOf builds a book giving bf_emma the entries given, found with the files the plans are made
// against.
func endingsOf(t *testing.T, entries ...ending.Entry) ending.Book {
	t.Helper()
	book, err := ending.NewBook(bookFade, files.Model, map[string]ending.Voice{
		"bf_emma": {Style: files.Style, Entries: entries},
	})
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	return book
}

// docked answers bf_emma's plan line for Docked's second line with the books given.
func docked(t *testing.T, pauses pause.Book, endings ending.Book) making.Line {
	t.Helper()
	return making.New(voiced(t, "bə"), emma, files, pauses, endings, nil).ToMake()[1]
}

// FR-556: a line carries the ending the book gives its voice for it; the voice's other lines and every
// line of a voice the book gives nothing carry none.
func TestALineCarriesItsVoicesEndingAndNoOther(t *testing.T) {
	lines := making.New(voiced(t, "bə"), emma, files, noPauses, endingsOf(t, fadedAt), nil).ToMake()
	for index, line := range lines {
		given := index == 1
		if line.EndingGiven != given || line.Faded() != given || (given && line.Ending != fadedAt) {
			t.Errorf("bf_emma's line %d carries %+v given %v; want an ending only on Docked's second", index, line.Ending, line.EndingGiven)
		}
	}
	for _, line := range making.New(voiced(t, "bə"), michael, files, noPauses, endingsOf(t, fadedAt), nil).ToMake() {
		if line.EndingGiven || line.Faded() {
			t.Errorf("am_michael's %s line %d carries bf_emma's ending", line.Cue, line.Index)
		}
	}
}

// FR-513: a line with no ending and a line whose ending gives no fade keep exactly the key they were
// made under before endings, so no made line already on a user's disk goes stale.
func TestALineWithNoEndingOrNoFadeKeepsItsKeyFromBeforeEndings(t *testing.T) {
	still := fadedAt
	still.Sample = 0
	line := docked(t, noPauses, endingsOf(t, still))
	if !line.EndingGiven || line.Faded() {
		t.Fatalf("Docked's second line carries %+v given %v; want its ending with no fade", line.Ending, line.EndingGiven)
	}
	for _, each := range append(making.New(voiced(t, "bə"), emma, files, noPauses, endingsOf(t, still), nil).ToMake(),
		making.New(voiced(t, "bə"), michael, files, noPauses, noEndings, nil).ToMake()...) {
		if want := oldKey(each.Sounds); each.Key != want {
			t.Errorf("%s line %d is keyed %s, want %s as before endings", each.Cue, each.Index, each.Key, want)
		}
	}
	paused := pausedAt
	if got, want := docked(t, bookOf(t, paused), endingsOf(t, still)).Key, making.PausedKey("bɪ", files, paused); got != want {
		t.Errorf("a paused line with no fade is keyed %s, want its paused key %s", got, want)
	}
}

// FR-513: a faded line's key changes with its fade's sample or the digest it was found in, the parts
// kept apart so a digit moved from one to the other changes it too.
func TestAFadedLinesKeyChangesWithItsSampleOrItsDigest(t *testing.T) {
	base := docked(t, noPauses, endingsOf(t, fadedAt)).Key
	if base == oldKey("bɪ") {
		t.Error("a faded line kept the key it had before endings")
	}
	moved, digest, boundary := fadedAt, fadedAt, fadedAt
	moved.Sample = 8
	digest.Digest = "fedcba9876543210"
	boundary.Sample, boundary.Digest = 71, foundIn[1:]
	for name, other := range map[string]ending.Entry{"sample": moved, "digest": digest, "boundary": boundary} {
		if docked(t, noPauses, endingsOf(t, other)).Key == base {
			t.Errorf("a changed %s left the key unchanged", name)
		}
	}
}

// FR-513: a pause and a fade with the same sample and digest are keyed apart; a line with both is
// keyed apart from either alone.
func TestAPauseAndAFadeAreKeyedApart(t *testing.T) {
	same := pause.Entry{Cue: fadedAt.Cue, Index: fadedAt.Index, Sounds: fadedAt.Sounds, Digest: fadedAt.Digest, Sample: fadedAt.Sample}
	keys := map[string]string{
		"pause": docked(t, bookOf(t, same), noEndings).Key,
		"fade":  docked(t, noPauses, endingsOf(t, fadedAt)).Key,
		"both":  docked(t, bookOf(t, same), endingsOf(t, fadedAt)).Key,
	}
	seen := make(map[string]string)
	for name, key := range keys {
		if other, taken := seen[key]; taken {
			t.Errorf("a line with %s and one with %s share the key %s", name, other, key)
		}
		seen[key] = name
	}
}

// FR-513: a line whose fade changed or went is made again and no other line is.
func TestALineWhoseFadeChangedIsTheOnlyOneMadeAgain(t *testing.T) {
	made := keysOf(making.New(voiced(t, "bə"), emma, files, noPauses, endingsOf(t, fadedAt), nil).ToMake())
	moved := fadedAt
	moved.Sample = 8
	for name, book := range map[string]ending.Book{"moved": endingsOf(t, moved), "gone": noEndings} {
		again := making.New(voiced(t, "bə"), emma, files, noPauses, book, made).ToMake()
		if len(again) != 1 || again[0].Cue != "Docked" || again[0].Index != 1 {
			t.Errorf("with the fade %s, to make again = %+v; want Docked's second line alone", name, again)
		}
	}
}
