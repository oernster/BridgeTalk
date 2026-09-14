package making_test

// FR-511 and FR-514: the order a machine voice's lines are made in; a cue's lines still to make.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
)

// cueOf builds a cue of the source and priority given; an empty priority is ambient.
func cueOf(t *testing.T, id, source, priority string) cue.Cue {
	t.Helper()
	built, err := cue.New(cue.Definition{ID: id, Source: source, Event: "moment", Priority: priority})
	if err != nil {
		t.Fatalf("building cue %s: %v", id, err)
	}
	return built
}

// cuesOf returns the cue of each line given, in order.
func cuesOf(lines []making.Line) []cue.ID {
	ids := make([]cue.ID, 0, len(lines))
	for _, line := range lines {
		ids = append(ids, line.Cue)
	}
	return ids
}

// FR-511: the cue the application plays on a cast comes first, then alert, notice, ambient and
// flavour cues, each priority in the order the table lists it.
func TestTheCuePlayedOnACastComesFirstThenEachPriorityInTableOrder(t *testing.T) {
	table := cue.NewTable([]cue.Cue{
		cueOf(t, "Scanned", "journal", "flavour"),
		cueOf(t, "Docked", "journal", "notice"),
		cueOf(t, "HullDamage", "journal", "alert"),
		cueOf(t, "Cast.Confirmed", "application", "notice"),
		cueOf(t, "Interdicted", "journal", "alert"),
		cueOf(t, "Undocked", "journal", ""),
	})

	got := making.Order(table)

	want := []cue.ID{"Cast.Confirmed", "HullDamage", "Interdicted", "Docked", "Undocked", "Scanned"}
	if !slices.Equal(got, want) {
		t.Errorf("Order = %v, want %v", got, want)
	}
}

// FR-511: a plan's lines follow the order. A cue the order does not name comes after; a cue the order
// names without lines is passed over.
func TestAPlansLinesFollowTheOrder(t *testing.T) {
	plan := making.New(voiced(t, "bə"), []cue.ID{"Undocked", "Scanned"}, machinevoice.British, files, nil)

	want := []cue.ID{"Undocked", "Undocked", "Undocked", "Docked", "Docked", "Docked"}
	if got := cuesOf(plan.ToMake()); !slices.Equal(got, want) {
		t.Errorf("lines to make are for %v, want %v", got, want)
	}
	if plan.Total() != len(want) {
		t.Errorf("Total = %d, want %d", plan.Total(), len(want))
	}
}

// FR-514: a cue's lines still to make come in line order; a line already made is not among them.
func TestUnmadeGivesACuesLinesStillToMakeInLineOrder(t *testing.T) {
	all := making.New(voiced(t, "bə"), nil, machinevoice.British, files, nil).ToMake()
	plan := making.New(voiced(t, "bə"), nil, machinevoice.British, files, []string{all[1].Key})

	got := plan.Unmade("Docked")

	if len(got) != 2 || got[0] != all[0] || got[1] != all[2] {
		t.Errorf("Unmade(Docked) = %+v, want its first and third lines", got)
	}
	if none := plan.Unmade("Scanned"); len(none) != 0 {
		t.Errorf("Unmade of a cue with no lines = %+v, want none", none)
	}
}
