package main

// FR-529, FR-532, FR-549 and FR-550: every line made in each accent from that accent's spelling and
// the table of words, saved in order with a final commander joined.

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// recorder is a sounds maker that keeps what it was handed and answers as the test says.
type recorder struct {
	calls  []machinevoice.Accent
	handed map[machinevoice.Accent][]string
	answer func(accent machinevoice.Accent, lines []string) ([]string, error)
}

// Make records the call and answers as the test says.
func (r *recorder) Make(accent machinevoice.Accent, lines []string) ([]string, error) {
	r.calls = append(r.calls, accent)
	if r.handed == nil {
		r.handed = make(map[machinevoice.Accent][]string)
	}
	r.handed[accent] = lines
	return r.answer(accent, lines)
}

// echo answers every line with its accent followed by the text it was given.
func echo(accent machinevoice.Accent, lines []string) ([]string, error) {
	answered := make([]string, 0, len(lines))
	for _, line := range lines {
		answered = append(answered, accent.String()+": "+line)
	}
	return answered, nil
}

// givenSpelling finds a word given with its speech sounds, keeping the sounds.
var givenSpelling = regexp.MustCompile(`\[[^\]]*\]\(/([^/)]*)/\)`)

// sounder answers every line as misaki does a spelled word: the word's given sounds stand in its
// place and every other character stays.
func sounder(_ machinevoice.Accent, lines []string) ([]string, error) {
	answered := make([]string, 0, len(lines))
	for _, line := range lines {
		answered = append(answered, givenSpelling.ReplaceAllString(line, "$1"))
	}
	return answered, nil
}

// scriptOf builds a script for Docked and Undocked from their lines.
func scriptOf(t *testing.T, entries map[string][]string) script.Script {
	t.Helper()
	var cues []cue.Cue
	for _, id := range []string{"Docked", "Undocked"} {
		item, err := cue.New(cue.Definition{ID: id, Source: "journal", Event: id, Purpose: "When it happens."})
		if err != nil {
			t.Fatalf("building cue %q: %v", id, err)
		}
		cues = append(cues, item)
	}
	built, err := script.New(entries, cue.NewTable(cues))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return built
}

// undocked is a cue's three plain lines.
var undocked = []string{"Clear of the pad.", "Undocked.", "Leaving the station."}

// twoCues builds a script for Docked and Undocked; one Docked line spells a word both ways.
func twoCues(t *testing.T) script.Script {
	t.Helper()
	return scriptOf(t, map[string][]string{
		"Undocked": undocked,
		"Docked":   {"Docked.", "Flight [record](/ˈɹɛkɔːd/ˈɹɛkɚd/) saved.", "Down safely."},
	})
}

// withCommander builds a script whose Docked lines hold commander, the table giving its sounds and
// joining it after a final comma.
func withCommander(t *testing.T, docked ...string) script.Script {
	t.Helper()
	built, err := scriptOf(t, map[string][]string{"Undocked": undocked, "Docked": docked}).
		WithWords(map[string][]string{"commander": {"kəmˈɑndə", "kəmˈændəɹ"}}, []string{"commander"})
	if err != nil {
		t.Fatalf("WithWords: %v", err)
	}
	return built
}

// Every line is made once for each accent, British first, from that accent's spelling; the
// sounds are saved against the lines as written, cue by cue in order.
func TestEveryLineIsMadeInEachAccentFromItsSpelling(t *testing.T) {
	maker := &recorder{answer: echo}
	saved, err := makeSounds(twoCues(t), maker)
	if err != nil {
		t.Fatalf("makeSounds: %v", err)
	}
	if want := machinevoice.Accents(); !slices.Equal(maker.calls, want) {
		t.Errorf("calls = %v, want one per accent %v", maker.calls, want)
	}
	docked := saved["Docked"]
	if want := []string{"Docked.", "Flight [record](/ˈɹɛkɔːd/ˈɹɛkɚd/) saved.", "Down safely."}; !slices.Equal(docked.Lines, want) {
		t.Errorf("Docked lines = %q, want %q", docked.Lines, want)
	}
	if got, want := docked.British[1], "British: Flight [record](/ˈɹɛkɔːd/) saved."; got != want {
		t.Errorf("British sounds = %q, want %q", got, want)
	}
	if got, want := docked.American[1], "American: Flight [record](/ˈɹɛkɚd/) saved."; got != want {
		t.Errorf("American sounds = %q, want %q", got, want)
	}
	if got, want := saved["Undocked"].American[2], "American: Leaving the station."; got != want {
		t.Errorf("Undocked's last American sounds = %q, want %q", got, want)
	}
}

// FR-549: a line holding a table word is handed to the maker with the word spelled for each
// accent, while the line is saved as it is written.
func TestATableWordIsHandedToTheMakerSpelledForEachAccent(t *testing.T) {
	maker := &recorder{answer: echo}
	saved, err := makeSounds(withCommander(t, "Docked.", "Sold, commander. Credits are in.", "A commander docked."), maker)
	if err != nil {
		t.Fatalf("makeSounds: %v", err)
	}
	if got, want := maker.handed[machinevoice.British][1], "Sold, [commander](/kəmˈɑndə/). Credits are in."; got != want {
		t.Errorf("British line handed = %q, want %q", got, want)
	}
	if got, want := maker.handed[machinevoice.American][2], "A [commander](/kəmˈændəɹ/) docked."; got != want {
		t.Errorf("American line handed = %q, want %q", got, want)
	}
	if got, want := saved["Docked"].Lines[1], "Sold, commander. Credits are in."; got != want {
		t.Errorf("line saved = %q, want it as written %q", got, want)
	}
}

// FR-550: a line ending with a comma and commander is saved with the comma and its space left out
// in each accent; a line where commander does not end it keeps its comma.
func TestAJoiningLineIsSavedWithItsCommaLeftOut(t *testing.T) {
	maker := &recorder{answer: sounder}
	saved, err := makeSounds(withCommander(t, "Docked.", "Breathable atmosphere, commander.", "Sold, commander. Credits are in."), maker)
	if err != nil {
		t.Fatalf("makeSounds: %v", err)
	}
	docked := saved["Docked"]
	for got, want := range map[string]string{
		docked.British[1]:  "Breathable atmospherekəmˈɑndə.",
		docked.American[1]: "Breathable atmospherekəmˈændəɹ.",
		docked.British[2]:  "Sold, kəmˈɑndə. Credits are in.",
	} {
		if got != want {
			t.Errorf("saved %q, want %q", got, want)
		}
	}
}

// FR-550: an answer for a joining line that does not end with the comma and the word's spelling
// stops the run, naming the cue, the line and the accent.
func TestAnAnswerThatCannotBeJoinedStopsTheRunNamingTheCueTheLineAndTheAccent(t *testing.T) {
	maker := &recorder{answer: echo}
	_, err := makeSounds(withCommander(t, "Docked.", "Breathable atmosphere, commander.", "Down safely."), maker)
	if !errors.Is(err, script.ErrUnjoinable) {
		t.Fatalf("got %v, want ErrUnjoinable", err)
	}
	for _, fragment := range []string{`"Docked"`, `"Breathable atmosphere, commander."`, "British"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("error %q does not name %s", err, fragment)
		}
	}
}

// A maker that fails stops the run, keeping its reason.
func TestAMakerThatFailsStopsTheRun(t *testing.T) {
	broken := errors.New("no venv")
	maker := &recorder{answer: func(machinevoice.Accent, []string) ([]string, error) { return nil, broken }}
	if _, err := makeSounds(twoCues(t), maker); !errors.Is(err, broken) {
		t.Errorf("got %v, want the maker's own error", err)
	}
}

// A maker answering a different number of lines than it was given is refused, since its sounds
// could no longer be matched to lines.
func TestAMakerAnsweringTheWrongNumberOfLinesIsRefused(t *testing.T) {
	maker := &recorder{answer: func(accent machinevoice.Accent, lines []string) ([]string, error) {
		answered, _ := echo(accent, lines)
		return answered[1:], nil
	}}
	if _, err := makeSounds(twoCues(t), maker); !errors.Is(err, errLineCount) {
		t.Errorf("got %v, want errLineCount", err)
	}
}
