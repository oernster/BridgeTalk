package script_test

// FR-546: the script's cues gathered into the groups the Audition pane offers a machine voice, each
// counting the lines its cues hold.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
)

// threeSaved is three lines with their speech sounds saved in each accent.
func threeSaved() script.Saved {
	return script.Saved{
		Lines:   []string{"One.", "Two.", "Three."},
		British: []string{"bə", "bɪ", "bi"}, American: []string{"æə", "æɪ", "æi"},
	}
}

// A group is every cue sharing the first segment of its id, as FR-216 groups a recorded voice's takes.
func TestGroupsGatherCuesByTheirFirstSegmentCountingTheirLines(t *testing.T) {
	voiced, err := scripttest.Build(map[string]script.Saved{
		"ShieldState.ShieldsUp.false": threeSaved(),
		"ShieldState.ShieldsUp.true":  threeSaved(),
		"Docked":                      threeSaved(),
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	want := []script.Group{{Key: "Docked", Lines: 3}, {Key: "ShieldState", Lines: 6}}
	if got := voiced.Groups(); !slices.Equal(got, want) {
		t.Errorf("groups = %v, want %v sorted by key", got, want)
	}
}
