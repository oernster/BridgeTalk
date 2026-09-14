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
