package making_test

// FR-546 and FR-745: the lines a machine voice is auditioned on for one group, from its cues heard.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
)

// A group's lines are its cues' lines in the plan's order, made or not; a group no cue belongs to
// has none.
func TestAGroupGivesItsCuesLinesInOrderMadeOrNot(t *testing.T) {
	plan := making.New(voiced(t, "bə"), emma, files, noPauses, noEndings, []string{making.Key("dɪ", files)})

	want := []string{making.Key("du", files), making.Key("dɪ", files), making.Key("di", files)}
	if got := keysOf(plan.Group("Undocked", cue.HeardAll)); !slices.Equal(got, want) {
		t.Errorf("Undocked's lines = %v, want %v", got, want)
	}
	if got := plan.Group("ShieldState", cue.HeardAll); len(got) != 0 {
		t.Errorf("a group no cue belongs to gave %v, want nothing", got)
	}
}

// FR-745: a group gives the lines of its cues heard alone; with none heard it gives nothing.
func TestAGroupGivesOnlyTheLinesOfItsCuesHeard(t *testing.T) {
	const on, off = cue.ID("ShieldState.ShieldsUp.true"), cue.ID("ShieldState.ShieldsUp.false")
	built, err := scripttest.Build(map[string]script.Saved{
		string(on): {
			Lines:   []string{"Shields up.", "Shields holding.", "Shields back."},
			British: []string{"ʃu", "ʃh", "ʃb"}, American: []string{"ʃæu", "ʃæh", "ʃæb"},
		},
		string(off): {
			Lines:   []string{"Shields down.", "Shields gone.", "Shields failing."},
			British: []string{"ʃd", "ʃɡ", "ʃf"}, American: []string{"ʃæd", "ʃæɡ", "ʃæf"},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	plan := making.New(built, emma, files, noPauses, noEndings, nil)

	lines := plan.Group("ShieldState", func(id cue.ID) bool { return id != off })
	if len(lines) != 3 || slices.ContainsFunc(lines, func(line making.Line) bool { return line.Cue != on }) {
		t.Errorf("lines = %+v, want the three lines of %s alone", lines, on)
	}
	if got := plan.Group("ShieldState", func(cue.ID) bool { return false }); len(got) != 0 {
		t.Errorf("a group with no cue heard gave %+v, want nothing", got)
	}
}
