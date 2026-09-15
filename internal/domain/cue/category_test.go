package cue_test

// Categories (FR-621, FR-634, FR-635): every cue the game raises sits in one listed category; the cue
// from the application sits in none and has no switch.

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// categorised builds a cue with a category.
func categorised(t *testing.T, id, source, category string) cue.Cue {
	t.Helper()
	return mustCue(t, cue.Definition{ID: id, Source: source, Event: id, Flag: id, Category: category})
}

// A categorised table holds its categories in the order given with each cue carrying its own; it resolves
// as any table does.
func TestACategorisedTableHoldsItsCategoriesInOrder(t *testing.T) {
	t.Parallel()
	categories := []string{"Session", "Docking and stations"}
	table, err := cue.NewCategorisedTable(categories, []cue.Cue{
		categorised(t, "Docked", "journal", "Docking and stations"),
		categorised(t, "LoadGame", "journal", "Session"),
		categorised(t, "Cast.Confirmed", "application", ""),
	})
	if err != nil {
		t.Fatalf("NewCategorisedTable: %v", err)
	}
	if got := table.Categories(); !reflect.DeepEqual(got, categories) {
		t.Errorf("Categories() = %v, want %v", got, categories)
	}
	if got := table.All()[0].Category(); got != "Docking and stations" {
		t.Errorf("Docked's category = %q", got)
	}
	if table.Len() != 3 {
		t.Errorf("Len() = %d, want 3", table.Len())
	}
	table.Categories()[0] = "changed"
	if table.Categories()[0] != "Session" {
		t.Error("changing the answer changed the table")
	}
}

// A table built without categories lists none.
func TestATableBuiltWithoutCategoriesListsNone(t *testing.T) {
	t.Parallel()
	if got := cue.NewTable(nil).Categories(); len(got) != 0 {
		t.Errorf("Categories() = %v, want none", got)
	}
}

// FR-634: each way a table can get its categories wrong is refused, naming what is wrong.
func TestATableWithItsCategoriesWrongIsRefusedNamingWhatIsWrong(t *testing.T) {
	t.Parallel()
	docked := func(category string) cue.Cue { return categorised(t, "Docked", "journal", category) }
	cases := map[string]struct {
		categories []string
		cues       []cue.Cue
		names      string
	}{
		"a blank category":               {[]string{" "}, nil, "no name"},
		"a category named twice":         {[]string{"Session", "Session"}, nil, `"Session"`},
		"a cue with no category":         {[]string{"Session"}, []cue.Cue{docked("")}, "Docked"},
		"a cue in an unlisted category":  {[]string{"Session"}, []cue.Cue{docked("Elsewhere")}, `"Elsewhere"`},
		"an application cue in one":      {[]string{"Session"}, []cue.Cue{categorised(t, "Cast.Confirmed", "application", "Session")}, "Cast.Confirmed"},
		"a category holding no cue":      {[]string{"Session", "Empty"}, []cue.Cue{categorised(t, "LoadGame", "journal", "Session")}, `"Empty"`},
		"a status cue with no category":  {[]string{"Session"}, []cue.Cue{categorised(t, "LightsOn", "status", "")}, "LightsOn"},
		"an unlisted category, sole cue": {nil, []cue.Cue{docked("Session")}, `"Session"`},
	}
	for name, each := range cases {
		_, err := cue.NewCategorisedTable(each.categories, each.cues)
		if !errors.Is(err, cue.ErrInvalidCue) || !strings.Contains(err.Error(), each.names) {
			t.Errorf("%s: err = %v, want a refusal naming %s", name, err, each.names)
		}
	}
}

// FR-621: every cue the game raises has a switch; the cue from the application has none.
func TestOnlyACueFromTheApplicationHasNoSwitch(t *testing.T) {
	t.Parallel()
	for source, want := range map[string]bool{"journal": true, "status": true, "application": false} {
		if got := categorised(t, "Docked", source, "").Switchable(); got != want {
			t.Errorf("a %s cue: Switchable() = %t, want %t", source, got, want)
		}
	}
}
