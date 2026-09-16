package main

// FR-745 to FR-748 through the facade: the Audition pane's groups and draws read the Chatter switches as
// they stand, for a recorded voice and a machine voice alike.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// switchedOffDocked gives the fixture app's session a Chatter listing Docked, then switches it off.
func switchedOffDocked(t *testing.T, app *App) {
	t.Helper()
	docked, err := cue.New(cue.Definition{
		ID: "Docked", Source: "journal", Event: "Docked", Purpose: "When the ship docks.", Category: "Docking and stations",
	})
	if err != nil {
		t.Fatalf("building Docked: %v", err)
	}
	table, err := cue.NewCategorisedTable([]string{"Docking and stations"}, []cue.Cue{docked})
	if err != nil {
		t.Fatalf("building the table: %v", err)
	}
	app.session.chatter = services.NewChatterService(table, nil)
	if _, err := app.SetMoment("Docked", false); err != nil {
		t.Fatalf("switching Docked off: %v", err)
	}
}

// groupNamed answers the group under key; false where the list has none.
func groupNamed(groups []GroupDTO, key string) (GroupDTO, bool) {
	for _, group := range groups {
		if group.Key == key {
			return group, true
		}
	}
	return GroupDTO{}, false
}

// FR-745, FR-747: with Docked switched off in Chatter, a recorded voice's Docked group is marked
// switched off counting nothing and cannot be auditioned; its other groups are untouched, the cue
// from the application among them, which Chatter does not list.
func TestTheAuditionPaneAsksChatterWhatIsSwitchedOn(t *testing.T) {
	app, _, _ := fixtureApp(t)
	switchedOffDocked(t, app)

	groups := app.AuditionGroups("Alpha")

	if docked, ok := groupNamed(groups, "Docked"); !ok || !docked.SwitchedOff || docked.Clips != 0 {
		t.Errorf("Docked = %+v, %v; want it marked switched off counting nothing", docked, ok)
	}
	for _, key := range []string{"Cast", "ShieldState"} {
		if group, ok := groupNamed(groups, key); !ok || group.SwitchedOff || group.Clips == 0 {
			t.Errorf("%s = %+v, %v; want it offered as before", key, group, ok)
		}
	}
	if _, err := app.Audition("Alpha", "Docked"); err == nil {
		t.Error("a group with every moment switched off was auditioned")
	}
}

// FR-749, FR-750: the groups come in Chatter's category order, each carrying its category, a group in
// no category last; recorded and machine voices alike.
func TestTheAuditionGroupsComeInCategoryOrder(t *testing.T) {
	app, _, _ := fixtureApp(t)
	var cues []cue.Cue
	for _, each := range []cue.Definition{
		{ID: "ShieldState.ShieldsUp.false", Source: "journal", Event: "ShieldState", Purpose: "When the shields fail.", Category: "Combat and danger"},
		{ID: "Docked", Source: "journal", Event: "Docked", Purpose: "When the ship docks.", Category: "Docking and stations"},
		{ID: "Cast.Confirmed", Source: "application", Event: "cast", Purpose: "When this voice is cast."},
	} {
		built, err := cue.New(each)
		if err != nil {
			t.Fatalf("building %s: %v", each.ID, err)
		}
		cues = append(cues, built)
	}
	table, err := cue.NewCategorisedTable([]string{"Combat and danger", "Docking and stations"}, cues)
	if err != nil {
		t.Fatalf("building the table: %v", err)
	}
	app.session.table = table

	shown := func(groups []GroupDTO) []string {
		out := make([]string, 0, len(groups))
		for _, group := range groups {
			out = append(out, group.Key+"/"+group.Category)
		}
		return out
	}
	recorded := []string{"ShieldState/Combat and danger", "Docked/Docking and stations", "Cast/"}
	if got := shown(app.AuditionGroups("Alpha")); !slices.Equal(got, recorded) {
		t.Errorf("recorded groups = %v, want %v", got, recorded)
	}
	machine := []string{"Docked/Docking and stations", "Cast/"}
	if got := shown(app.MachineAuditionGroups()); !slices.Equal(got, machine) {
		t.Errorf("machine groups = %v, want %v", got, machine)
	}
}

// FR-745, FR-747: a machine voice reads the same switches.
func TestAMachineAuditionAsksChatterWhatIsSwitchedOn(t *testing.T) {
	app, _, _ := fixtureApp(t)
	switchedOffDocked(t, app)

	groups := app.MachineAuditionGroups()

	if docked, ok := groupNamed(groups, "Docked"); !ok || !docked.SwitchedOff || docked.Clips != 0 {
		t.Errorf("Docked = %+v, %v; want it marked switched off counting nothing", docked, ok)
	}
	if cast, ok := groupNamed(groups, "Cast"); !ok || cast.SwitchedOff {
		t.Errorf("Cast = %+v, %v; want it offered as before", cast, ok)
	}
	if _, err := app.AuditionMachineVoice("bf_emma", "Docked"); err == nil {
		t.Error("a machine voice group with every moment switched off was auditioned")
	}
}
