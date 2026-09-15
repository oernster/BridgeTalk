package script_test

// FR-555: the lines of a voiced script whose last speech sound in an accent is a nasal, listed with
// their saved sounds for the pauses tool and the stale check.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
)

// FR-555: each accent lists its own lines ending on a nasal, cue by cue in the order Cues gives, then
// line by line; a line ending on a nasal in one accent alone is listed for that accent alone.
func TestEndingOnNasalListsEachLineEndingOnANasalInItsAccentInCueThenLineOrder(t *testing.T) {
	voiced, err := scripttest.Build(map[string]script.Saved{
		"Undocked": {
			Lines:    []string{"Undocked.", "Moving on.", "Gone."},
			British:  []string{"ʌndˈɒkt.", "mˈuːvɪŋ ˈɒn.", "ɡˈɒn."},
			American: []string{"ʌndˈɑkt.", "mˈuvɪŋ ˈɔn.", "ɡˈɔ."},
		},
		"Docked": {
			Lines:    []string{"Docking.", "Down.", "Home, commander."},
			British:  []string{"dˈɒkɪŋ.", "dˈWn.", "hˈQm, kəmˈɑndə."},
			American: []string{"dˈɑkɪŋ.", "dˈWn.", "hˈOm, kəmˈændəɹ."},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for accent, want := range map[machinevoice.Accent][]script.SavedLine{
		machinevoice.British: {
			{Cue: "Docked", Index: 0, Sounds: "dˈɒkɪŋ."}, {Cue: "Docked", Index: 1, Sounds: "dˈWn."},
			{Cue: "Undocked", Index: 1, Sounds: "mˈuːvɪŋ ˈɒn."}, {Cue: "Undocked", Index: 2, Sounds: "ɡˈɒn."},
		},
		machinevoice.American: {
			{Cue: "Docked", Index: 0, Sounds: "dˈɑkɪŋ."}, {Cue: "Docked", Index: 1, Sounds: "dˈWn."},
			{Cue: "Undocked", Index: 1, Sounds: "mˈuvɪŋ ˈɔn."},
		},
	} {
		if got := voiced.EndingOnNasal(accent); !slices.Equal(got, want) {
			t.Errorf("%s EndingOnNasal = %+v, want %+v", accent, got, want)
		}
	}
}
