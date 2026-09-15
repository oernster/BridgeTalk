package services_test

// Chatter (FR-621 to FR-633): which moments are switched off, applied at once and kept for the next run.

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// keptSettings is a settings store held in memory; refuse makes every save fail with it.
type keptSettings struct {
	held   ports.Settings
	saves  int
	refuse error
}

func (k *keptSettings) Load() ports.Settings { return k.held }

func (k *keptSettings) Save(chosen ports.Settings) error {
	if k.refuse != nil {
		return k.refuse
	}
	k.held = chosen
	k.saves++
	return nil
}

// chatterTable holds two categories, three moments the game raises and the cue from the application.
func chatterTable(t *testing.T) cue.Table {
	t.Helper()
	table, err := cue.NewCategorisedTable([]string{"Docking and stations", "Session"}, []cue.Cue{
		cueFor(t, cue.Definition{ID: "Docked", Source: "journal", Event: "Docked", Category: "Docking and stations"}),
		cueFor(t, cue.Definition{ID: "Cast.Confirmed", Source: "application", Event: "cast"}),
		cueFor(t, cue.Definition{ID: "Undocked", Source: "journal", Event: "Undocked", Category: "Docking and stations"}),
		cueFor(t, cue.Definition{ID: "LoadGame", Source: "journal", Event: "LoadGame", Category: "Session"}),
	})
	if err != nil {
		t.Fatalf("building the table: %v", err)
	}
	return table
}

// states reads every moment's state, category by category, as "id on" or "id off".
func states(service *services.ChatterService) map[string][]string {
	got := map[string][]string{}
	for _, category := range service.Categories() {
		for _, moment := range category.Moments {
			state := "off"
			if moment.On {
				state = "on"
			}
			got[category.Name] = append(got[category.Name], string(moment.Cue.ID())+" "+state)
		}
	}
	return got
}

// FR-628: with nothing kept, as with no store at all, every moment is on.
func TestEveryMomentStartsSwitchedOnWithNothingKept(t *testing.T) {
	t.Parallel()
	for name, store := range map[string]ports.SettingsStore{"nothing kept": &keptSettings{}, "no store": nil} {
		service := services.NewChatterService(chatterTable(t), store)
		for _, id := range []cue.ID{"Docked", "Undocked", "LoadGame"} {
			if service.Off(id) {
				t.Errorf("%s: %s is off", name, id)
			}
		}
	}
}

// FR-629: the switches kept last run are the ones in force at start.
func TestTheSwitchesOutliveTheRun(t *testing.T) {
	t.Parallel()
	store := &keptSettings{held: ports.Settings{SwitchedOff: []string{"Docked"}}}
	service := services.NewChatterService(chatterTable(t), store)
	if !service.Off("Docked") || service.Off("Undocked") {
		t.Errorf("Docked off = %t, Undocked off = %t; want only Docked off", service.Off("Docked"), service.Off("Undocked"))
	}
}

// FR-627, FR-629: a switch applies at once and is kept beside every other setting, which it leaves alone.
func TestSettingASwitchIsAppliedAndKept(t *testing.T) {
	t.Parallel()
	store := &keptSettings{held: ports.Settings{LibraryRoot: `D:\Voices`}}
	service := services.NewChatterService(chatterTable(t), store)

	if err := service.SetMoment("Docked", false); err != nil {
		t.Fatalf("SetMoment: %v", err)
	}
	if !service.Off("Docked") {
		t.Error("Docked is still on")
	}
	if !reflect.DeepEqual(store.held.SwitchedOff, []string{"Docked"}) || store.held.LibraryRoot != `D:\Voices` {
		t.Errorf("kept %+v, want Docked off beside the recordings directory", store.held)
	}

	if err := service.SetMoment("Docked", true); err != nil {
		t.Fatalf("SetMoment: %v", err)
	}
	if service.Off("Docked") || len(store.held.SwitchedOff) != 0 {
		t.Errorf("Docked off = %t, kept %v; want it on and nothing kept", service.Off("Docked"), store.held.SwitchedOff)
	}
}

// FR-631: an id kept last run that no moment has is let go when a switch is next kept.
func TestAKeptSwitchForNoCueIsLetGo(t *testing.T) {
	t.Parallel()
	store := &keptSettings{held: ports.Settings{SwitchedOff: []string{"Gone"}}}
	service := services.NewChatterService(chatterTable(t), store)

	if err := service.SetMoment("Undocked", false); err != nil {
		t.Fatalf("SetMoment: %v", err)
	}
	if !reflect.DeepEqual(store.held.SwitchedOff, []string{"Undocked"}) {
		t.Errorf("kept %v, want Undocked alone", store.held.SwitchedOff)
	}
}

// FR-633: a switch that cannot be kept says why and still applies for the run.
func TestASwitchThatCannotBeKeptSaysWhy(t *testing.T) {
	t.Parallel()
	store := &keptSettings{refuse: errors.New("the disk is full")}
	service := services.NewChatterService(chatterTable(t), store)

	err := service.SetMoment("Docked", false)
	if err == nil || !strings.Contains(err.Error(), "the disk is full") {
		t.Errorf("err = %v, want the reason it was not kept", err)
	}
	if !service.Off("Docked") {
		t.Error("a switch that could not be kept did not apply")
	}
}

// FR-621: the cue from the application, an id no cue has and a category not listed cannot be switched.
func TestOnlyAMomentChatterListsCanBeSwitched(t *testing.T) {
	t.Parallel()
	service := services.NewChatterService(chatterTable(t), &keptSettings{})
	for name, err := range map[string]error{
		"the application's cue": service.SetMoment("Cast.Confirmed", false),
		"an id no cue has":      service.SetMoment("Gone", false),
		"a category not listed": service.SetCategory("Elsewhere", false),
	} {
		if !errors.Is(err, services.ErrNoSuchMoment) {
			t.Errorf("%s: err = %v, want ErrNoSuchMoment", name, err)
		}
	}
}

// FR-727: every moment is listed under its category in the table's order, with its state; the cue from
// the application is not listed.
func TestTheCategoriesListEveryMomentInOrderWithItsState(t *testing.T) {
	t.Parallel()
	service := services.NewChatterService(chatterTable(t), &keptSettings{})
	if err := service.SetMoment("Undocked", false); err != nil {
		t.Fatalf("SetMoment: %v", err)
	}

	names := []string{}
	for _, category := range service.Categories() {
		names = append(names, category.Name)
	}
	if !reflect.DeepEqual(names, []string{"Docking and stations", "Session"}) {
		t.Errorf("categories %v, want the table's order", names)
	}
	want := map[string][]string{
		"Docking and stations": {"Docked on", "Undocked off"},
		"Session":              {"LoadGame on"},
	}
	if got := states(service); !reflect.DeepEqual(got, want) {
		t.Errorf("states %v, want %v", got, want)
	}
}

// FR-731: a category's switch changes every moment in it and none elsewhere.
func TestACategorysSwitchChangesEveryMomentInIt(t *testing.T) {
	t.Parallel()
	service := services.NewChatterService(chatterTable(t), &keptSettings{})
	if err := service.SetCategory("Docking and stations", false); err != nil {
		t.Fatalf("SetCategory: %v", err)
	}
	if !service.Off("Docked") || !service.Off("Undocked") || service.Off("LoadGame") {
		t.Errorf("states %v, want both docking moments off and LoadGame on", states(service))
	}
}

// FR-732: every moment is switched off, then on, at once.
func TestSwitchAllChangesEveryMoment(t *testing.T) {
	t.Parallel()
	store := &keptSettings{}
	service := services.NewChatterService(chatterTable(t), store)

	if err := service.SetAll(false); err != nil {
		t.Fatalf("SetAll off: %v", err)
	}
	if !reflect.DeepEqual(store.held.SwitchedOff, []string{"Docked", "LoadGame", "Undocked"}) {
		t.Errorf("kept %v, want every moment, in id order", store.held.SwitchedOff)
	}
	if err := service.SetAll(true); err != nil {
		t.Fatalf("SetAll on: %v", err)
	}
	if len(store.held.SwitchedOff) != 0 || store.saves != 2 {
		t.Errorf("kept %v after %d saves, want nothing after two", store.held.SwitchedOff, store.saves)
	}
}

// With no store a switch still applies; there is simply nowhere to keep it.
func TestWithNoStoreASwitchStillApplies(t *testing.T) {
	t.Parallel()
	service := services.NewChatterService(chatterTable(t), nil)
	if err := service.SetMoment("Docked", false); err != nil || !service.Off("Docked") {
		t.Errorf("err = %v, Docked off = %t; want it off with no error", err, service.Off("Docked"))
	}
}

// A table built without categories gives Chatter nothing to list.
func TestATableWithoutCategoriesListsNoMoments(t *testing.T) {
	t.Parallel()
	table := cue.NewTable([]cue.Cue{cueFor(t, cue.Definition{ID: "Docked", Source: "journal", Event: "Docked"})})
	for _, category := range services.NewChatterService(table, nil).Categories() {
		t.Errorf("listed %q", category.Name)
	}
}
