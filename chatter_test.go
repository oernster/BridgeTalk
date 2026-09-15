package main

// The Chatter surface of the facade (FR-727, FR-729, FR-731 to FR-733).

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
)

// chatterStore is a settings store held in memory; refuse makes every save fail with it.
type chatterStore struct {
	held   ports.Settings
	refuse error
}

func (c *chatterStore) Load() ports.Settings { return c.held }

func (c *chatterStore) Save(chosen ports.Settings) error {
	if c.refuse != nil {
		return c.refuse
	}
	c.held = chosen
	return nil
}

// chatterApp is a facade over two categories, three moments the game raises and the cast confirmation.
func chatterApp(t *testing.T, store ports.SettingsStore) *App {
	t.Helper()
	var cues []cue.Cue
	for _, each := range []cue.Definition{
		{ID: "Docked", Source: "journal", Event: "Docked", Purpose: "When the ship docks.", Category: "Docking and stations"},
		{ID: "Undocked", Source: "journal", Event: "Undocked", Purpose: "When the ship leaves.", Category: "Docking and stations"},
		{ID: "LoadGame", Source: "journal", Event: "LoadGame", Purpose: "When the game loads.", Category: "Session"},
		{ID: "Cast.Confirmed", Source: "application", Event: "cast", Purpose: "When this voice is cast."},
	} {
		built, err := cue.New(each)
		if err != nil {
			t.Fatalf("building %s: %v", each.ID, err)
		}
		cues = append(cues, built)
	}
	table, err := cue.NewCategorisedTable([]string{"Docking and stations", "Session"}, cues)
	if err != nil {
		t.Fatalf("building the table: %v", err)
	}
	return &App{session: &session{table: table, chatter: services.NewChatterService(table, store)}}
}

// onStates reads each moment's id and whether it is on, category by category.
func onStates(pane ChatterDTO) map[string]map[string]bool {
	got := map[string]map[string]bool{}
	for _, category := range pane.Categories {
		got[category.Name] = map[string]bool{}
		for _, moment := range category.Moments {
			got[category.Name][moment.Cue.ID] = moment.On
		}
	}
	return got
}

// FR-727's acceptance: each moment is listed under its category with its full title, its purpose and its
// state; the cast confirmation is not listed.
func TestChatterListsEveryMomentUnderItsCategory(t *testing.T) {
	t.Parallel()
	pane := chatterApp(t, &chatterStore{}).Chatter()

	if len(pane.Categories) != 2 || pane.Categories[0].Name != "Docking and stations" || pane.Problem != "" {
		t.Fatalf("pane = %+v, want two categories in order and no problem", pane)
	}
	first := pane.Categories[0].Moments[0]
	if first.Cue.Title != "Docked" || first.Cue.Purpose != "When the ship docks." || !first.On {
		t.Errorf("first moment = %+v, want Docked with its purpose, switched on", first)
	}
	want := map[string]map[string]bool{
		"Docking and stations": {"Docked": true, "Undocked": true},
		"Session":              {"LoadGame": true},
	}
	if got := onStates(pane); !reflect.DeepEqual(got, want) {
		t.Errorf("states %v, want %v", got, want)
	}
}

// FR-729: a moment switched from the pane is switched, kept and shown so in the answer.
func TestSettingAMomentFromThePaneSwitchesAndKeepsIt(t *testing.T) {
	t.Parallel()
	store := &chatterStore{}
	app := chatterApp(t, store)

	pane, err := app.SetMoment("Docked", false)
	if err != nil {
		t.Fatalf("SetMoment: %v", err)
	}
	if onStates(pane)["Docking and stations"]["Docked"] || !reflect.DeepEqual(store.held.SwitchedOff, []string{"Docked"}) {
		t.Errorf("pane %v, kept %v; want Docked off and kept", onStates(pane), store.held.SwitchedOff)
	}
}

// FR-731, FR-732: a category, then every moment, is switched from the pane.
func TestACategoryAndEveryMomentAreSwitchedFromThePane(t *testing.T) {
	t.Parallel()
	app := chatterApp(t, &chatterStore{})

	pane, err := app.SetCategory("Session", false)
	if err != nil || onStates(pane)["Session"]["LoadGame"] || !onStates(pane)["Docking and stations"]["Docked"] {
		t.Errorf("after the Session switch: %v, err %v; want LoadGame alone off", onStates(pane), err)
	}
	pane, err = app.SetAllMoments(false)
	if err != nil || onStates(pane)["Docking and stations"]["Undocked"] {
		t.Errorf("after all off: %v, err %v; want every moment off", onStates(pane), err)
	}
	pane, err = app.SetAllMoments(true)
	if err != nil || !onStates(pane)["Session"]["LoadGame"] {
		t.Errorf("after all on: %v, err %v; want every moment on", onStates(pane), err)
	}
}

// A moment or a category the pane does not list is refused with the reason.
func TestAMomentThePaneDoesNotListIsRefused(t *testing.T) {
	t.Parallel()
	app := chatterApp(t, &chatterStore{})
	_, momentErr := app.SetMoment("Cast.Confirmed", false)
	_, categoryErr := app.SetCategory("Elsewhere", false)
	for name, err := range map[string]error{"moment": momentErr, "category": categoryErr} {
		if !errors.Is(err, services.ErrNoSuchMoment) {
			t.Errorf("%s: err = %v, want a refusal", name, err)
		}
	}
}

// FR-633's acceptance: a switch that cannot be kept is not refused; the pane shows why, with the moment
// switched off for the run.
func TestASwitchThatCannotBeKeptIsShownOnThePane(t *testing.T) {
	t.Parallel()
	app := chatterApp(t, &chatterStore{refuse: errors.New("the disk is full")})

	pane, err := app.SetMoment("Docked", false)
	if err != nil {
		t.Fatalf("SetMoment refused a switch that applied: %v", err)
	}
	if !strings.Contains(pane.Problem, "the disk is full") || onStates(pane)["Docking and stations"]["Docked"] {
		t.Errorf("pane = %+v, want the reason shown with Docked off", pane)
	}
}

// FR-630's acceptance: a moment switched off stays off when another voice is cast, because each cast
// hands the one set of switches to the reaction service it builds.
func TestCastingAVoiceLeavesTheSwitchesAlone(t *testing.T) {
	app, _, _ := fixtureApp(t)
	docked, err := cue.New(cue.Definition{ID: "Docked", Source: "journal", Event: "Docked", Category: "Docking and stations"})
	if err != nil {
		t.Fatalf("building Docked: %v", err)
	}
	listed, err := cue.NewCategorisedTable([]string{"Docking and stations"}, []cue.Cue{docked})
	if err != nil {
		t.Fatalf("building the table: %v", err)
	}
	app.session.chatter = services.NewChatterService(listed, nil)
	// main wires the facade in as the reporter; the fixture leaves that to the test.
	app.session.reporter = reporter{app: app}

	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting Alpha: %v", err)
	}
	if err := app.session.chatter.SetMoment("Docked", false); err != nil {
		t.Fatalf("switching Docked off: %v", err)
	}
	if err := app.SelectVoice("Beta"); err != nil {
		t.Fatalf("casting Beta: %v", err)
	}
	at := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	app.session.reactions.HandleAll([]event.Event{event.New(event.SourceJournal, "Docked", event.EdgeNone, nil, at)})

	history := app.Reactions()
	for index := len(history) - 1; index >= 0; index-- {
		if history[index].Cue == "Docked" {
			if history[index].Outcome != ports.OutcomeOff {
				t.Errorf("Docked was recorded %q with Beta cast, want %q", history[index].Outcome, ports.OutcomeOff)
			}
			return
		}
	}
	t.Errorf("no decision was recorded for Docked; history %+v", history)
}

// A session built with nothing holding the switches lists nothing and refuses a switch.
func TestChatterWithNothingHoldingTheSwitchesListsNothing(t *testing.T) {
	t.Parallel()
	app := &App{session: &session{}}
	if pane := app.Chatter(); pane.Categories == nil || len(pane.Categories) != 0 {
		t.Errorf("Chatter() = %+v, want an empty list rather than none", pane)
	}
	if _, err := app.SetAllMoments(false); !errors.Is(err, errChatterNotReady) {
		t.Errorf("err = %v, want the refusal", err)
	}
}
