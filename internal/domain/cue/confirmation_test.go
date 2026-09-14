package cue_test

// FR-232 and FR-521: the cue played on a cast is the application's own moment, found by its source
// rather than its id, so no id is written outside the cue table.

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// castCue builds a cue of the source given.
func castCue(t *testing.T, id, source string) cue.Cue {
	t.Helper()
	built, err := cue.New(cue.Definition{ID: id, Source: source, Event: "moment"})
	if err != nil {
		t.Fatalf("building cue %s: %v", id, err)
	}
	return built
}

func TestTheConfirmationIsTheCueWithTheApplicationAsItsSource(t *testing.T) {
	docked := castCue(t, "Docked", "journal")
	confirmed := castCue(t, "Cast.Confirmed", "application")

	if id, ok := cue.NewTable([]cue.Cue{docked, confirmed}).Confirmation(); !ok || id != "Cast.Confirmed" {
		t.Errorf("Confirmation = %q, %v; want Cast.Confirmed", id, ok)
	}
	if id, ok := cue.NewTable([]cue.Cue{docked}).Confirmation(); ok {
		t.Errorf("Confirmation = %q with no application cue, want none", id)
	}
}
