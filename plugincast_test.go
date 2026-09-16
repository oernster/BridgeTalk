package main

// Casting the plugin voice the settings kept. A plugin voice is cast by the plugin that
// offered it and its own id within that plugin, since an id is unique only there (FR-569).
// Everything a cast has to survive is here: the voice gone, the plugin gone, the voice
// present but with no audio behind it.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin/plugintest"
)

// offering loads one plugin offering the voices given, as the composition root would.
func offering(t *testing.T, name string, voices ...plugintest.Voice) *plugin.Set {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.dll"), []byte("stand-in"), 0o600); err != nil {
		t.Fatalf("writing the file: %v", err)
	}
	set := plugin.Load(dir, func(string) (plugin.Library, error) {
		return &plugintest.Plugin{Name: name, Voices: voices}, nil
	})
	t.Cleanup(set.Close)
	if len(set.Refusals) != 0 {
		t.Fatalf("the plugin was refused: %+v", set.Refusals)
	}
	return set
}

// officer is a voice with audio behind it, answering the fixture's docking cue.
func officer() plugintest.Voice {
	return plugintest.Voice{
		ID: "one", Name: "The First Officer", Ready: true,
		Answers: map[string][][]string{"Docked": {{`C:\audio\docked.mp3`}}},
	}
}

func TestAKeptPluginVoiceIsCastAtStart(t *testing.T) {
	current, _ := fixtureSession(t, newFakePlayer())
	current.plugins = offering(t, "Bridge Crew", officer())
	var warnings strings.Builder

	current.castAtStart(keptCast{plugin: "Bridge Crew", pluginVoice: "one"}, current.available[0], &warnings)

	if warnings.Len() != 0 {
		t.Fatalf("warned %q, want the cast made in silence", warnings.String())
	}
	if current.active.Name != "one" || current.active.Plugin != "Bridge Crew" {
		t.Errorf("cast %+v, want the voice by its id and the plugin that offered it", current.active)
	}
	if current.active.Display != "The First Officer" {
		t.Errorf("shown as %q, want the name the plugin gave", current.active.Display)
	}
	if current.active.Machine {
		t.Error("a plugin voice was cast as a machine voice")
	}
}

// The cast is not just a label: the voice speaks through the same port every other kind
// speaks through, so the catalogue answers the takes the plugin holds.
func TestACastPluginVoiceAnswersTheCatalogue(t *testing.T) {
	current, _ := fixtureSession(t, newFakePlayer())
	current.plugins = offering(t, "Bridge Crew", officer())

	if err := current.castPlugin("Bridge Crew", "one"); err != nil {
		t.Fatalf("casting: %v", err)
	}

	performance, served := current.catalogue.Clips("Docked")
	if !served || len(performance.Takes) != 1 {
		t.Fatalf("the catalogue answered %v, %v; want the take the plugin holds", performance, served)
	}
	if got := performance.Takes[0].Key(); got != `C:\audio\docked.mp3` {
		t.Errorf("the take is %q, want the path the plugin gave", got)
	}
}

// Each of these is a kept plugin voice that cannot be cast. All warn with something the
// reader can act on, then fall back to a recorded voice rather than starting silent.
func TestAKeptPluginVoiceThatCannotBeCastFallsBackToARecordedVoice(t *testing.T) {
	absent := plugintest.Voice{
		ID: "one", Name: "The First Officer",
		Reason: "its recordings are not on this machine",
	}

	for _, each := range []struct {
		name, plugin, id, said string
		offered                []plugintest.Voice
		loaded                 bool
	}{
		{"the voice is no longer offered", "Bridge Crew", "gone", "gone", []plugintest.Voice{officer()}, true},
		{"the plugin is no longer there", "Another Crew", "one", "Another Crew", []plugintest.Voice{officer()}, true},
		{"its audio is not on this machine", "Bridge Crew", "one", "not on this machine", []plugintest.Voice{absent}, true},
		{"no plugin is loaded at all", "Bridge Crew", "one", "no plugin", nil, false},
	} {
		t.Run(each.name, func(t *testing.T) {
			current, _ := fixtureSession(t, newFakePlayer())
			if each.loaded {
				current.plugins = offering(t, "Bridge Crew", each.offered...)
			}
			var warnings strings.Builder

			current.castAtStart(
				keptCast{plugin: each.plugin, pluginVoice: each.id}, current.available[0], &warnings)

			if !strings.Contains(warnings.String(), each.said) {
				t.Errorf("warned %q, want it to say %q", warnings.String(), each.said)
			}
			if current.active.Name != "Alpha" || current.active.Plugin != "" {
				t.Errorf("cast %+v, want the recorded voice instead", current.active)
			}
		})
	}
}

// A voice whose audio is not on this machine is refused rather than cast, since casting it
// would be casting silence (FR-570). The reason the plugin gave is what the reader is told.
func TestAPluginVoiceWithNoAudioIsRefusedWithItsReason(t *testing.T) {
	current, _ := fixtureSession(t, newFakePlayer())
	current.plugins = offering(t, "Bridge Crew", plugintest.Voice{
		ID: "one", Name: "The First Officer", Reason: "its recordings are not on this machine",
	})

	err := current.castPlugin("Bridge Crew", "one")

	if err == nil {
		t.Fatal("a voice with no audio behind it was cast")
	}
	if !strings.Contains(err.Error(), "its recordings are not on this machine") {
		t.Errorf("refused with %q, want the reason the plugin gave", err)
	}
	if !strings.Contains(err.Error(), "The First Officer") {
		t.Errorf("refused with %q, want the voice named", err)
	}
}
