package speech_test

// FR-529 and FR-532: a line as it was written; a line written for one accent in the
// one-spelling form misaki reads.

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
)

// A line keeps the text it was written with, spellings and all, so saved speech sounds can be
// matched to it.
func TestALineKeepsItsTextAsWritten(t *testing.T) {
	text := "Flight [record](/ˈɹɛkɔːd/ˈɹɛkɚd/) saved."
	if got := mustRead(t, text).Text(); got != text {
		t.Errorf("Text() = %q, want %q", got, text)
	}
}

// Written for an accent, each spelled word carries that accent's one spelling; plain words stand
// as they are.
func TestALineIsWrittenForAnAccentWithThatAccentsSpelling(t *testing.T) {
	line := mustRead(t, "Flight [record](/ˈɹɛkɔːd/ˈɹɛkɚd/) saved.")
	for accent, want := range map[machinevoice.Accent]string{
		machinevoice.British:  "Flight [record](/ˈɹɛkɔːd/) saved.",
		machinevoice.American: "Flight [record](/ˈɹɛkɚd/) saved.",
	} {
		if got := line.ForAccent(accent); got != want {
			t.Errorf("ForAccent(%s) = %q, want %q", accent, got, want)
		}
	}
}

// A line with no spelling is written for either accent exactly as it stands.
func TestAPlainLineIsWrittenUnchanged(t *testing.T) {
	if got := mustRead(t, "Docking complete.").ForAccent(machinevoice.American); got != "Docking complete." {
		t.Errorf("ForAccent = %q, want the line unchanged", got)
	}
}
