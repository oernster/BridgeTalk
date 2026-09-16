package cue_test

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// FR-745: with no switches to ask, every moment is heard, the cue from the application included.
func TestHeardAllHearsEveryMoment(t *testing.T) {
	t.Parallel()

	for _, id := range []cue.ID{"Docked", "ShieldState.ShieldsUp.false", "Cast.Confirmed", ""} {
		if !cue.HeardAll(id) {
			t.Errorf("HeardAll(%q) = false, want every moment heard", id)
		}
	}
}
