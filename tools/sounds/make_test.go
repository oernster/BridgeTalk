package main

// FR-529 and FR-532: every line made in each accent from that accent's spelling, saved in order.

import (
	"errors"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// recorder is a sounds maker that answers each line with its accent and the text it was given.
type recorder struct {
	calls  []machinevoice.Accent
	answer func(accent machinevoice.Accent, lines []string) ([]string, error)
}

// Make records the call and answers as the test says.
func (r *recorder) Make(accent machinevoice.Accent, lines []string) ([]string, error) {
	r.calls = append(r.calls, accent)
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

// twoCues builds a script for Docked and Undocked; one Docked line spells a word both ways.
func twoCues(t *testing.T) script.Script {
	t.Helper()
	var cues []cue.Cue
	for _, id := range []string{"Docked", "Undocked"} {
		item, err := cue.New(cue.Definition{ID: id, Source: "journal", Event: id, Purpose: "When it happens."})
		if err != nil {
			t.Fatalf("building cue %q: %v", id, err)
		}
		cues = append(cues, item)
	}
	built, err := script.New(map[string][]string{
		"Undocked": {"Clear of the pad.", "Undocked.", "Leaving the station."},
		"Docked":   {"Docked.", "Flight [record](/ˈɹɛkɔːd/ˈɹɛkɚd/) saved.", "Down safely."},
	}, cue.NewTable(cues))
	if err != nil {
		t.Fatalf("New: %v", err)
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
