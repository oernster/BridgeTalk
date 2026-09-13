package cue_test

// FR-219 and FR-222 enforced where every cue table is built, whether it shipped with the
// application or was supplied by the user. The structural test over cues.toml warns
// early about the shipped table; this is what stops a user's own table from loading an
// id the flat form would misread.

import (
	"errors"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// An id whose final segment is digits alone is refused, naming the id.
func TestAnIdEndingInASegmentOfDigitsIsRefused(t *testing.T) {
	for _, id := range []string{"plant.2", "42", "DockingGranted.007"} {
		t.Run(id, func(t *testing.T) {
			_, err := cue.New(cue.Definition{ID: id, Source: "journal", Event: "X"})
			if !errors.Is(err, cue.ErrInvalidCue) || !strings.Contains(err.Error(), id) {
				t.Fatalf("New(%q) = %v, want ErrInvalidCue naming the id", id, err)
			}
		})
	}
}

// Digits are refused only as a whole final segment. Beside letters they are an ordinary
// part of a name; so they are in any segment before the last.
func TestDigitsElsewhereInAnIdAreAccepted(t *testing.T) {
	for _, id := range []string{"plant.v2", "plant2", "FSDJump.2nd.Leg"} {
		t.Run(id, func(t *testing.T) {
			if _, err := cue.New(cue.Definition{ID: id, Source: "journal", Event: "X"}); err != nil {
				t.Fatalf("New(%q) refused an id FR-219 allows: %v", id, err)
			}
		})
	}
}

// FR-222: an id ending in a dot or a space is refused, naming the id and the reason. An
// id ending in a dot has an empty final segment, which is not a segment of digits, so
// the refusal must not be misreported as FR-219.
func TestAnIdEndingInADotOrASpaceIsRefused(t *testing.T) {
	for _, id := range []string{"plant.", "plant ", "Docked.Set. ", "DockingGranted.2."} {
		t.Run(id, func(t *testing.T) {
			_, err := cue.New(cue.Definition{ID: id, Source: "journal", Event: "X"})
			if !errors.Is(err, cue.ErrInvalidCue) || !strings.Contains(err.Error(), "dot or a space") {
				t.Fatalf("New(%q) = %v, want ErrInvalidCue saying it ends in a dot or a space", id, err)
			}
			if strings.Contains(err.Error(), "digits") {
				t.Fatalf("New(%q) was reported as digits: %v", id, err)
			}
		})
	}
}

// A space or a dot anywhere but the end is an ordinary part of an id: the game spells
// some values with a space in them.
func TestASpaceOrDotBeforeTheEndIsAccepted(t *testing.T) {
	for _, id := range []string{"Synthesis.Name.Repair Basic", "LightsOn.Set"} {
		if _, err := cue.New(cue.Definition{ID: id, Source: "journal", Event: "X"}); err != nil {
			t.Errorf("New(%q) refused an id FR-222 allows: %v", id, err)
		}
	}
}
