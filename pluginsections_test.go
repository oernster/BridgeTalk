package main

// The section and group each plugin voice stands under on the Cast pane (FR-583, FR-584). The
// facade answers both for every voice; the pane does the grouping, so what is pinned here is what
// each voice carries.

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/plugin/plugintest"
)

// placed is where one listed voice stands: its section, its group and the name shown inside them.
type placed struct{ section, group, name string }

// placings reads where each listed voice stands, in the order the listing gave them.
func placings(listed []PluginVoiceDTO) []placed {
	out := make([]placed, 0, len(listed))
	for _, each := range listed {
		out = append(out, placed{each.Section, each.Group, each.Name})
	}
	return out
}

// Each voice stands under the plugin that offered it, in the group that plugin named for it; a
// voice named in no group stands in none (FR-582, FR-583, FR-584).
func TestEachPluginVoiceStandsUnderItsPluginAndGroup(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.session.plugins = offeringAll(t,
		loadedPlugin{file: "quartermaster.dll", name: "Quartermaster Voices", voices: []plugintest.Voice{
			{ID: "ada", Name: "Ada", Group: "Crew", Ready: true},
			{ID: "bo", Name: "Bo", Ready: true},
		}},
		loadedPlugin{file: "station.dll", name: "Station Voices", voices: []plugintest.Voice{
			{ID: "cy", Name: "Cy", Group: "Stations", Ready: true},
		}},
	)

	got := placings(app.PluginVoices())

	want := []placed{
		{"Quartermaster Voices", "Crew", "Ada"},
		{"Quartermaster Voices", "", "Bo"},
		{"Station Voices", "Stations", "Cy"},
	}
	if len(got) != len(want) {
		t.Fatalf("listed %+v, want %+v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("voice %d stands at %+v, want %+v", index, got[index], want[index])
		}
	}
}

// Two plugins carrying one name head two sections, each told apart by the file it was loaded from,
// while a voice name the two share is shown plainly inside its section (FR-583, FR-568).
func TestTwoPluginsUnderOneNameHeadSectionsToldApartByTheirFiles(t *testing.T) {
	app, _, _ := fixtureApp(t)
	sharing := []plugintest.Voice{{ID: "one", Name: "The First Officer", Ready: true}}
	app.session.plugins = offeringAll(t,
		loadedPlugin{file: "crew.dll", name: "Bridge Crew", voices: sharing},
		loadedPlugin{file: "other.dll", name: "Bridge Crew", voices: sharing},
	)

	got := placings(app.PluginVoices())

	want := []placed{
		{"Bridge Crew (crew.dll)", "", "The First Officer"},
		{"Bridge Crew (other.dll)", "", "The First Officer"},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("listed %+v, want %+v", got, want)
	}
}
