package main

// FR-745 to FR-748 through the facade: the Audition pane's groups and draws read the Chatter switches as
// they stand, for a recorded voice and a machine voice alike.

import (
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
