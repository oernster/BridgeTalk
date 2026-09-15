package config_test

// Categories and station traffic in the cue table (FR-621, FR-634, FR-635, FR-637, FR-638).

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
)

// FR-634's acceptance: a table whose Docked entry has no category fails to load, naming Docked.
func TestACueWithNoCategoryIsRefusedByName(t *testing.T) {
	t.Parallel()
	path := overrideFile(t, "cues.toml", `
[[category]]
name = "Docking and stations"

[[cue]]
id = "Docked"
purpose = "When the ship docks."
source = "journal"
event = "Docked"
`)
	if _, err := config.LoadCueTable(path); !errorMentions(err, "Docked") || !errorMentions(err, "category") {
		t.Errorf("err = %v, want the load refused naming Docked and its category", err)
	}
}

// FR-634: a category the table does not list is refused by name.
func TestACategoryOutsideTheSetIsRefused(t *testing.T) {
	t.Parallel()
	path := overrideFile(t, "cues.toml", `
[[category]]
name = "Docking and stations"

[[cue]]
id = "Docked"
purpose = "When the ship docks."
source = "journal"
event = "Docked"
category = "Elsewhere"
`)
	if _, err := config.LoadCueTable(path); !errorMentions(err, `"Elsewhere"`) {
		t.Errorf("err = %v, want the load refused naming the category", err)
	}
}

// FR-621, FR-634's acceptance: every moment the shipped table holds has a switch and a listed category;
// Cast.Confirmed has neither.
func TestEveryShippedMomentHasACategory(t *testing.T) {
	t.Parallel()
	table := shippedTable(t)
	listed := map[string]bool{}
	for _, name := range table.Categories() {
		listed[name] = true
	}
	for _, item := range table.All() {
		switch {
		case item.ID() == "Cast.Confirmed":
			if item.Switchable() || item.Category() != "" {
				t.Errorf("Cast.Confirmed: switchable %t, category %q; want neither", item.Switchable(), item.Category())
			}
		case !item.Switchable() || !listed[item.Category()]:
			t.Errorf("%s: switchable %t, category %q; want a switch and a listed category", item.ID(), item.Switchable(), item.Category())
		}
	}
}

// FR-635's acceptance: the shipped categories are the set FR-635 gives, in its order, none of them empty.
func TestTheShippedCategoriesAreTheSetInOrder(t *testing.T) {
	t.Parallel()
	table := shippedTable(t)
	want := []string{
		"Combat and danger", "Flight and travel", "Docking and stations", "Comms", "Ship systems",
		"Exploration", "Trade, missions and outfitting", "Materials and engineering",
		"On foot and vehicles", "Fleet carriers", "Wings, squadrons and friends", "Session",
	}
	if got := table.Categories(); !reflect.DeepEqual(got, want) {
		t.Errorf("Categories() = %v, want %v", got, want)
	}
}

// FR-637's acceptance, FR-638: the shipped station traffic moment carries what FR-637 gives it and
// answers every station traffic key, whatever the rest of its stem.
func TestTheShippedStationTrafficMomentIsAsSpecified(t *testing.T) {
	t.Parallel()
	table := shippedTable(t)
	at := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	const id = cue.ID("ReceiveText.StationTraffic")

	for _, key := range []string{"$STATION_docking_granted;", "$DockingChatter_Cordial;", "$DockingFailed_Distance;", "$STATION_NoFireZone_exited;"} {
		received := event.New(event.SourceJournal, "ReceiveText", event.EdgeNone,
			map[string]any{"Channel": "npc", "Message": key}, at)
		resolved, ok := table.Resolve(received)
		if !ok || resolved.ID() != id {
			t.Errorf("%s resolved to %q, want %s", key, resolved.ID(), id)
			continue
		}
		if resolved.Priority() != cue.PriorityAmbient || resolved.Cooldown() != 30*time.Second {
			t.Errorf("%s: priority %v, cooldown %v; want ambient and 30 s", id, resolved.Priority(), resolved.Cooldown())
		}
		if resolved.Category() != "Docking and stations" || strings.TrimSpace(resolved.Purpose()) == "" {
			t.Errorf("%s: category %q, purpose %q; want Docking and stations and a purpose", id, resolved.Category(), resolved.Purpose())
		}
	}
}
