package scripttest_test

// The builder the making tests share: a voiced script from what the sounds tool would have saved.

import (
	"errors"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
)

// docked is what the sounds tool would save for three lines of Docked.
var docked = script.Saved{
	Lines:   []string{"Docking complete.", "Down safely.", "Docked."},
	British: []string{"bə", "bɪ", "bi"}, American: []string{"æə", "æɪ", "æi"},
}

// A built script holds every saved cue with its lines' sounds in each accent.
func TestABuiltScriptHoldsWhatWasSaved(t *testing.T) {
	voiced, err := scripttest.Build(map[string]script.Saved{"Docked": docked})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := voiced.Cues(); !slices.Equal(got, []cue.ID{"Docked"}) {
		t.Errorf("cues = %v, want Docked alone", got)
	}
	for accent, want := range map[machinevoice.Accent][]string{
		machinevoice.British: docked.British, machinevoice.American: docked.American,
	} {
		if got, _ := voiced.Sounds("Docked", accent); !slices.Equal(got, want) {
			t.Errorf("%s sounds = %v, want %v", accent, got, want)
		}
	}
}

// A key no cue can take and a script the rules refuse are each refused, so a test never runs over
// a script other than the one it wrote.
func TestWhatTheRulesRefuseIsRefused(t *testing.T) {
	if _, err := scripttest.Build(map[string]script.Saved{"": docked}); !errors.Is(err, cue.ErrInvalidCue) {
		t.Errorf("an empty key built with %v, want ErrInvalidCue", err)
	}
	twoLines := docked
	twoLines.Lines = docked.Lines[:2]
	if _, err := scripttest.Build(map[string]script.Saved{"Docked": twoLines}); !errors.Is(err, script.ErrInvalidScript) {
		t.Errorf("two lines built with %v, want ErrInvalidScript", err)
	}
}

// FR-549 and FR-550: a script built joining gives the lines it joins with their sounds in each
// accent; one whose joined word the table of words lacks is refused.
func TestAScriptBuiltJoiningJoinsItsFinalWords(t *testing.T) {
	joining := script.Saved{
		Lines:   []string{"Docking complete.", "Down safely.", "Docked, commander."},
		British: []string{"bə", "bɪ", "bikəmˈɑndə."}, American: []string{"æə", "æɪ", "ækəmˈændəɹ."},
	}
	commander := map[string][]string{"commander": {"kəmˈɑndə", "kəmˈændəɹ"}}
	voiced, err := scripttest.BuildJoining(map[string]script.Saved{"Docked": joining}, commander, []string{"commander"})
	if err != nil {
		t.Fatalf("BuildJoining: %v", err)
	}
	want := []script.JoinedLine{{Cue: "Docked", Index: 2, Sounds: "ækəmˈændəɹ."}}
	if got := voiced.Joined(machinevoice.American); !slices.Equal(got, want) {
		t.Errorf("Joined(American) = %+v, want %+v", got, want)
	}
	_, err = scripttest.BuildJoining(map[string]script.Saved{"Docked": joining}, nil, []string{"commander"})
	if !errors.Is(err, script.ErrInvalidScript) {
		t.Errorf("a joined word the table lacks built with %v, want ErrInvalidScript", err)
	}
}
