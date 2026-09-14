package making_test

// FR-513 and FR-553: a line carries its voice's pause, which changes the line's key only where the
// pause is not doubtful; every other line keeps the key it was made under before pauses.

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/pause"
)

// bookSilence is the silence the books below insert.
const bookSilence = 960

// foundIn is the digest the pauses below were found in.
const foundIn = "0123456789abcdef"

// oldSeparator kept a key's parts apart before pauses.
const oldSeparator = "\x00"

// pausedAt is a pause for Docked's second line, whose British sounds are "bɪ".
var pausedAt = pause.Entry{Cue: "Docked", Index: 1, Sounds: "bɪ", Digest: foundIn, Sample: 7}

// bookOf builds a book giving bf_emma the entries given, found with the files the plans are made
// against.
func bookOf(t *testing.T, entries ...pause.Entry) pause.Book {
	t.Helper()
	book, err := pause.NewBook(bookSilence, files.Model, map[string]pause.Voice{
		"bf_emma": {Style: files.Style, Entries: entries},
	})
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	return book
}

// oldKey is a key computed as every made line on a user's disk was keyed before pauses: the SHA-256
// of the sounds, the style file digest and the model digest, kept apart by a zero byte.
func oldKey(sounds string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{sounds, files.Style, files.Model}, oldSeparator)))
	return hex.EncodeToString(sum[:])
}

// FR-553: a line carries the pause the book gives its voice for it; the voice's other lines and every
// line of a voice the book gives nothing carry none.
func TestALineCarriesItsVoicesPauseAndNoOther(t *testing.T) {
	lines := making.New(voiced(t, "bə"), emma, files, bookOf(t, pausedAt), nil).ToMake()
	for index, line := range lines {
		given := index == 1
		if line.PauseGiven != given || line.Paused() != given || (given && line.Pause != pausedAt) {
			t.Errorf("bf_emma's line %d carries %+v given %v; want a pause only on Docked's second", index, line.Pause, line.PauseGiven)
		}
	}
	for _, line := range making.New(voiced(t, "bə"), michael, files, bookOf(t, pausedAt), nil).ToMake() {
		if line.PauseGiven || line.Paused() {
			t.Errorf("am_michael's %s line %d carries bf_emma's pause", line.Cue, line.Index)
		}
	}
}

// FR-513: a line with no pause and a line whose pause is doubtful keep exactly the key they were made
// under before pauses, so no made line already on a user's disk goes stale.
func TestALineWithNoPauseOrADoubtfulOneKeepsItsKeyFromBeforePauses(t *testing.T) {
	doubtful := pause.Entry{Cue: "Docked", Index: 0, Sounds: "bə", Digest: foundIn, Doubtful: true}
	lines := making.New(voiced(t, "bə"), emma, files, bookOf(t, doubtful), nil).ToMake()
	if !lines[0].PauseGiven || lines[0].Paused() {
		t.Fatalf("Docked's first line carries %+v given %v; want its doubtful pause, not paused", lines[0].Pause, lines[0].PauseGiven)
	}
	lines = append(lines, making.New(voiced(t, "bə"), michael, files, noPauses, nil).ToMake()...)
	for _, line := range lines {
		if want := oldKey(line.Sounds); line.Key != want {
			t.Errorf("%s line %d is keyed %s, want %s as before pauses", line.Cue, line.Index, line.Key, want)
		}
	}
	if making.Key("bə", files) != oldKey("bə") || making.PausedKey("bə", files, doubtful) != oldKey("bə") {
		t.Error("Key or a doubtful PausedKey differs from the key before pauses")
	}
}

// FR-513: a paused line's key changes with its pause's sample or the digest it was found in. The parts
// are kept apart, so a digit moved from the sample to the digest changes it too.
func TestAPausedLinesKeyChangesWithItsSampleOrItsDigest(t *testing.T) {
	base := making.PausedKey("bɪ", files, pausedAt)
	if base == oldKey("bɪ") {
		t.Error("a paused line kept the key it had before pauses")
	}
	moved, digest, boundary := pausedAt, pausedAt, pausedAt
	moved.Sample = 8
	digest.Digest = "fedcba9876543210"
	boundary.Sample, boundary.Digest = 71, foundIn[1:]
	for name, other := range map[string]pause.Entry{"sample": moved, "digest": digest, "boundary": boundary} {
		if making.PausedKey("bɪ", files, other) == base {
			t.Errorf("a changed %s left the key unchanged", name)
		}
	}
	if line := making.New(voiced(t, "bə"), emma, files, bookOf(t, pausedAt), nil).ToMake()[1]; line.Key != base {
		t.Errorf("the paused line is keyed %s, want %s", line.Key, base)
	}
}

// FR-513: a line whose pause changed or went is made again and no other line is.
func TestALineWhosePauseChangedIsTheOnlyOneMadeAgain(t *testing.T) {
	made := keysOf(making.New(voiced(t, "bə"), emma, files, bookOf(t, pausedAt), nil).ToMake())
	moved := pausedAt
	moved.Sample = 8
	for name, book := range map[string]pause.Book{"moved": bookOf(t, moved), "gone": noPauses} {
		again := making.New(voiced(t, "bə"), emma, files, book, made).ToMake()
		if len(again) != 1 || again[0].Cue != "Docked" || again[0].Index != 1 {
			t.Errorf("with the pause %s, to make again = %+v; want Docked's second line alone", name, again)
		}
	}
}
