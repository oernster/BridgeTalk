package selection_test

// Switches (FR-622, FR-628, FR-629, FR-631).

import (
	"reflect"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/selection"
)

// FR-628: a set that names nothing has every cue switched on, the zero value included.
func TestEveryMomentStartsSwitchedOn(t *testing.T) {
	t.Parallel()
	for name, switches := range map[string]selection.Switches{
		"zero":  {},
		"empty": selection.NewSwitches(nil),
	} {
		if switches.Off("Docked") {
			t.Errorf("the %s set has Docked off", name)
		}
	}
}

// A set switches off exactly the ids it was built with.
func TestAnIdNamedIsSwitchedOffAndNoOther(t *testing.T) {
	t.Parallel()
	switches := selection.NewSwitches([]cue.ID{"Docked"})
	if !switches.Off("Docked") {
		t.Error("Docked is on")
	}
	if switches.Off("Undocked") {
		t.Error("Undocked is off")
	}
}

// A change answers a new set, changing the ids named alone; the set it was made from is left as it
// was, which is what lets one goroutine read a set while another changes it.
func TestWithChangesOnlyTheIdsNamedAndLeavesTheOriginalAlone(t *testing.T) {
	t.Parallel()
	original := selection.NewSwitches([]cue.ID{"Docked"})

	offMore := original.With(false, "Undocked", "Liftoff")
	for _, id := range []cue.ID{"Docked", "Undocked", "Liftoff"} {
		if !offMore.Off(id) {
			t.Errorf("%s is on after switching it off", id)
		}
	}
	if original.Off("Undocked") {
		t.Error("switching off in the copy changed the original")
	}

	onAgain := offMore.With(true, "Docked", "Undocked")
	if onAgain.Off("Docked") || onAgain.Off("Undocked") {
		t.Error("an id switched on is still off")
	}
	if !onAgain.Off("Liftoff") {
		t.Error("an id not named was switched on")
	}
	if !offMore.Off("Docked") {
		t.Error("switching on in the copy changed the set it was made from")
	}
}

// FR-629, FR-631: what is kept is the switched off cues the table holds with a switch, in id order; an
// id no cue has and the cue from the application are left out.
func TestKeptHoldsOnlyTheTablesSwitchableCuesInIdOrder(t *testing.T) {
	t.Parallel()
	table := cue.NewTable([]cue.Cue{
		definedCue(t, "Docked", "journal"),
		definedCue(t, "Cast.Confirmed", "application"),
		definedCue(t, "Bounty", "journal"),
		definedCue(t, "Undocked", "journal"),
	})
	switches := selection.NewSwitches([]cue.ID{"Docked", "Gone", "Cast.Confirmed", "Bounty"})

	want := []cue.ID{"Bounty", "Docked"}
	if got := switches.Kept(table); !reflect.DeepEqual(got, want) {
		t.Errorf("Kept() = %v, want %v", got, want)
	}
}

// definedCue builds a cue from one source, named for the event it listens for.
func definedCue(t *testing.T, id, source string) cue.Cue {
	t.Helper()
	built, err := cue.New(cue.Definition{ID: id, Source: source, Event: id})
	if err != nil {
		t.Fatalf("building %s: %v", id, err)
	}
	return built
}
