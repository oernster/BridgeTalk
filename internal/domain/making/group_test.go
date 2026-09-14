package making_test

// FR-546: the lines a machine voice is auditioned on for one group.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
)

// A group's lines are its cues' lines in the plan's order, made or not; a group no cue belongs to
// has none.
func TestAGroupGivesItsCuesLinesInOrderMadeOrNot(t *testing.T) {
	plan := making.New(voiced(t, "bə"), machinevoice.British, files, []string{making.Key("dɪ", files)})

	want := []string{making.Key("du", files), making.Key("dɪ", files), making.Key("di", files)}
	if got := keysOf(plan.Group("Undocked")); !slices.Equal(got, want) {
		t.Errorf("Undocked's lines = %v, want %v", got, want)
	}
	if got := plan.Group("ShieldState"); len(got) != 0 {
		t.Errorf("a group no cue belongs to gave %v, want nothing", got)
	}
}
