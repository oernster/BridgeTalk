// Tests for the words a cue carries, kept apart from cue_test.go so neither file
// approaches the module size cap.

package cue_test

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// Every title is generated from the id, never written, so the reading has to hold for
// each shape the table spells: an event alone, an event narrowed by a field, a flag with
// its edge, a status value and an initialism at either end of a word.
func TestACueReadsItsOwnIdInWords(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Docked":                        "Docked",
		"StartJump.JumpType.Hyperspace": "Start jump: jump type hyperspace",
		"ShieldState.ShieldsUp.false":   "Shield state: shields up false",
		"LandingGearDown.Cleared":       "Landing gear down: cleared",
		"GuiFocus.GalaxyMap":            "Gui focus: galaxy map",
		"FSDJump":                       "FSD jump",
		"SRVDestroyed":                  "SRV destroyed",
		"SAASignalsFound":               "SAA signals found",
		"Synthesis.Name.Repair Basic":   "Synthesis: name repair basic",
		"under_score":                   "Under score",
		"Fire2Group":                    "Fire2 group",
	}
	for id, want := range cases {
		item := mustCue(t, cue.Definition{ID: id, Source: "journal", Event: "Any"})
		if got := item.Title(); got != want {
			t.Errorf("Title() for %q = %q, want %q", id, got, want)
		}
	}
}

// A heading is the group read the same way, so a list of cues is headed by the moment
// they share rather than by a second list of names kept somewhere else.
func TestAHeadingIsTheGroupInWords(t *testing.T) {
	t.Parallel()
	cases := map[cue.ID]string{
		"StartJump.JumpType.Hyperspace": "Start jump",
		"FSDJump":                       "FSD jump",
		"":                              "",
	}
	for id, want := range cases {
		if got := id.Heading(); got != want {
			t.Errorf("ID(%q).Heading() = %q, want %q", id, got, want)
		}
	}
}

// The reading never panics on the edges of the shape: nothing at all, a leading dot or
// a trailing one.
func TestAnIdWithAnEmptySegmentStillReads(t *testing.T) {
	t.Parallel()
	cases := map[cue.ID]string{
		"":          "",
		".Leading":  "Leading",
		"Trailing.": "Trailing",
	}
	for id, want := range cases {
		if got := id.Title(); got != want {
			t.Errorf("ID(%q).Title() = %q, want %q", id, got, want)
		}
	}
}

// The zero Cue never reaches a reader, since New is the only way to build one and it
// rejects an empty id. The guard is here so the reading cannot panic on a slice of
// nothing if that ever stops being true.
func TestTheZeroCueTitlesAsNothingRatherThanPanicking(t *testing.T) {
	t.Parallel()
	var empty cue.Cue

	if got := empty.Title(); got != "" {
		t.Errorf("Title() = %q, want the empty string", got)
	}
}
