package main

// The plugin voices the Cast pane offers and casting one from the window. A plugin voice is
// named by the plugin that offered it and its own id within that plugin, so both halves travel
// in every direction (FR-565, FR-568, FR-569, FR-570).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin/plugintest"
)

// loadedPlugin is one plugin as a test lays it out: the file it is loaded from, the name it
// gives itself and the voices it offers.
type loadedPlugin struct {
	file   string
	name   string
	voices []plugintest.Voice
}

// offeringAll loads the plugins given from one folder, as the composition root would. Each file
// is written as a stand-in, since the opener answers from the list rather than reading it.
func offeringAll(t *testing.T, plugins ...loadedPlugin) *plugin.Set {
	t.Helper()
	dir := t.TempDir()
	byFile := map[string]loadedPlugin{}
	for _, each := range plugins {
		if err := os.WriteFile(filepath.Join(dir, each.file), []byte("stand-in"), 0o600); err != nil {
			t.Fatalf("writing the file: %v", err)
		}
		byFile[each.file] = each
	}
	set := plugin.Load(dir, func(path string) (plugin.Library, error) {
		each := byFile[filepath.Base(path)]
		return &plugintest.Plugin{Name: each.name, Voices: each.voices}, nil
	})
	t.Cleanup(set.Close)
	if len(set.Refusals) != 0 {
		t.Fatalf("a plugin was refused: %+v", set.Refusals)
	}
	return set
}

// offering loads one plugin offering the voices given.
func offering(t *testing.T, name string, voices ...plugintest.Voice) *plugin.Set {
	t.Helper()
	return offeringAll(t, loadedPlugin{file: "one.dll", name: name, voices: voices})
}

// displays reads the names a listing shows, in the order it gave them.
func displays(listed []PluginVoiceDTO) []string {
	out := make([]string, 0, len(listed))
	for _, each := range listed {
		out = append(out, each.Display)
	}
	return out
}

// Every voice every loaded plugin offers reaches the pane, with the plugin that offered it and
// its id within that plugin, which is what a cast sends back (FR-565, FR-569).
func TestEveryPluginVoiceReachesTheCastPane(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.session.plugins = offeringAll(t,
		loadedPlugin{file: "crew.dll", name: "Bridge Crew", voices: []plugintest.Voice{officer()}},
		loadedPlugin{file: "deck.dll", name: "Flight Deck", voices: []plugintest.Voice{
			{ID: "two", Name: "The Pilot", Ready: true},
		}},
	)

	listed := app.PluginVoices()

	if len(listed) != 2 {
		t.Fatalf("listed %+v, want both voices", listed)
	}
	first := listed[0]
	if first.Plugin != "Bridge Crew" || first.ID != "one" || first.Name != "The First Officer" {
		t.Errorf("the first is %+v, want the plugin, the id and the name it gave", first)
	}
	if !first.Ready || first.Reason != "" {
		t.Errorf("the first is %+v, want a voice with its audio behind it", first)
	}
	if listed[1].Plugin != "Flight Deck" || listed[1].ID != "two" {
		t.Errorf("the second is %+v, want the other plugin's voice", listed[1])
	}
}

// A voice whose audio is not on this machine is listed with the reason it gave rather than left
// out, which is the half of FR-570 the window owns. It cannot be cast, which the other half
// already refuses.
func TestAVoiceWithNoAudioIsListedWithTheReasonItGave(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.session.plugins = offering(t, "Bridge Crew", plugintest.Voice{
		ID: "one", Name: "The First Officer", Reason: "its recordings are not on this machine",
	})

	listed := app.PluginVoices()

	if len(listed) != 1 {
		t.Fatalf("listed %+v, want the voice shown rather than dropped", listed)
	}
	if listed[0].Ready {
		t.Error("a voice with no audio behind it is offered as ready to cast")
	}
	if listed[0].Reason != "its recordings are not on this machine" {
		t.Errorf("the reason is %q, want the one the plugin gave", listed[0].Reason)
	}
}

// Two plugins may honestly choose one name for a voice. Both are kept and each is shown with the
// plugin offering it, so the reader can tell which is which (FR-568).
func TestTwoVoicesUnderOneNameAreShownWithTheirPlugins(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.session.plugins = offeringAll(t,
		loadedPlugin{file: "crew.dll", name: "Bridge Crew", voices: []plugintest.Voice{
			{ID: "one", Name: "The First Officer", Ready: true},
			{ID: "two", Name: "The Pilot", Ready: true},
		}},
		loadedPlugin{file: "deck.dll", name: "Flight Deck", voices: []plugintest.Voice{
			{ID: "one", Name: "The First Officer", Ready: true},
		}},
	)

	shown := displays(app.PluginVoices())

	want := []string{"The First Officer (Bridge Crew)", "The Pilot", "The First Officer (Flight Deck)"}
	for index, each := range want {
		if shown[index] != each {
			t.Errorf("voice %d is shown as %q, want %q", index, shown[index], each)
		}
	}
}

// Where the two plugins carry one name as well, the file each was loaded from tells them apart,
// since that is the one thing about a plugin the user can see by opening the folder (FR-568).
func TestTwoPluginsUnderOneNameAreToldApartByTheirFiles(t *testing.T) {
	app, _, _ := fixtureApp(t)
	sharing := []plugintest.Voice{{ID: "one", Name: "The First Officer", Ready: true}}
	app.session.plugins = offeringAll(t,
		loadedPlugin{file: "crew.dll", name: "Bridge Crew", voices: sharing},
		loadedPlugin{file: "other.dll", name: "Bridge Crew", voices: sharing},
	)

	shown := displays(app.PluginVoices())

	want := []string{"The First Officer (crew.dll)", "The First Officer (other.dll)"}
	for index, each := range want {
		if shown[index] != each {
			t.Errorf("voice %d is shown as %q, want %q", index, shown[index], each)
		}
	}
}

// No plugin loaded is the ordinary case and is an answer rather than a fault: an empty list, not
// a missing one, so the page never reads a list that is not there (FR-562).
func TestNoPluginLoadedOffersNoVoice(t *testing.T) {
	app, _, _ := fixtureApp(t)

	listed := app.PluginVoices()

	if listed == nil || len(listed) != 0 {
		t.Errorf("listed %#v, want an empty list", listed)
	}
}

// Casting from the window casts the voice, tells the page and keeps the pair for the next run,
// which is the half of FR-569 nothing could reach while no voice was cast from the window.
func TestCastingAPluginVoiceFromTheWindowKeepsIt(t *testing.T) {
	app, _, log := fixtureApp(t)
	app.session.plugins = offering(t, "Bridge Crew", officer())
	store := &fakeSettings{held: ports.Settings{Voice: "Alpha", MachineVoice: "emma"}}
	app.settings = store

	if err := app.CastPluginVoice("Bridge Crew", "one"); err != nil {
		t.Fatalf("casting: %v", err)
	}

	if app.session.active.Plugin != "Bridge Crew" || app.session.active.Name != "one" {
		t.Errorf("cast %+v, want the plugin voice", app.session.active)
	}
	if store.held.Plugin != "Bridge Crew" || store.held.PluginVoice != "one" {
		t.Errorf("kept %+v, want the plugin and the voice's id within it", store.held)
	}
	if store.held.Voice != "" || store.held.MachineVoice != "" {
		t.Errorf("kept %+v, want every other kind of voice forgotten", store.held)
	}
	if log.countEmitted(stateEvent) == 0 {
		t.Error("the page was not told the cast voice had changed")
	}
}

// A voice that cannot speak is refused with the reason it gave and nothing is kept, so a restart
// does not open by casting a voice this run would not (FR-570).
func TestAPluginVoiceThatCannotSpeakIsNeverKept(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.session.plugins = offering(t, "Bridge Crew", plugintest.Voice{
		ID: "one", Name: "The First Officer", Reason: "its recordings are not on this machine",
	})
	store := &fakeSettings{}
	app.settings = store

	err := app.CastPluginVoice("Bridge Crew", "one")

	if err == nil {
		t.Fatal("a voice with no audio behind it was cast from the window")
	}
	if !strings.Contains(err.Error(), "not on this machine") {
		t.Errorf("refused with %q, want the reason the plugin gave", err)
	}
	if store.saves != 0 {
		t.Errorf("wrote the settings %d times, want a refusal to keep nothing", store.saves)
	}
}

// Casting a voice of another kind forgets the kept plugin voice. A start casts a kept plugin
// voice ahead of every other kind, so a plugin left behind here would speak next run in place of
// the voice just chosen (FR-540, FR-569).
func TestCastingAnotherKindForgetsTheKeptPluginVoice(t *testing.T) {
	for _, each := range []struct {
		name string
		cast func(app *App) error
	}{
		{"a recorded voice", func(app *App) error { return app.SelectVoice("Beta") }},
		{"a machine voice", func(app *App) error { return app.rememberMachineVoice("emma") }},
	} {
		t.Run(each.name, func(t *testing.T) {
			app, _, _ := fixtureApp(t)
			store := &fakeSettings{}
			app.settings = store
			if err := app.rememberPluginVoice("Bridge Crew", "one"); err != nil {
				t.Fatalf("keeping the plugin voice: %v", err)
			}

			if err := each.cast(app); err != nil {
				t.Fatalf("casting: %v", err)
			}

			if store.held.Plugin != "" || store.held.PluginVoice != "" {
				t.Errorf("kept %+v, want the plugin voice forgotten", store.held)
			}
		})
	}
}
