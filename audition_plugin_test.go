package main

// FR-585 to FR-587 through the facade: a plugin voice auditioned on the takes its plugin answers,
// without being cast, with a moment the plugin refuses counting nothing.

import (
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/take"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin/plugintest"
)

// pilot is a voice with audio behind it answering the fixture's shields moment, which the voice
// cast in these tests does not answer.
func pilot() plugintest.Voice {
	return plugintest.Voice{
		ID: "two", Name: "The Pilot", Ready: true,
		Answers: map[string][]take.Take{"ShieldState.ShieldsUp.false": {take.Of(`C:\audio\shields.mp3`)}},
	}
}

// FR-586: a plugin voice that is not cast is auditioned on its own takes; the cast voice stays cast.
func TestAPluginVoiceIsAuditionedOnItsOwnTakesWithoutBeingCast(t *testing.T) {
	app, player, _ := fixtureApp(t)
	app.session.plugins = offering(t, "Bridge Crew", officer(), pilot())
	if err := app.CastPluginVoice("Bridge Crew", "one"); err != nil {
		t.Fatalf("casting the officer: %v", err)
	}

	groups := app.PluginAuditionGroups("Bridge Crew", "two")
	played, err := app.AuditionPluginVoice("Bridge Crew", "two", "ShieldState")

	if shields, ok := groupNamed(groups, "ShieldState"); !ok || shields.Clips != 1 {
		t.Errorf("groups = %+v, want ShieldState counting the pilot's one take", groups)
	}
	if _, ok := groupNamed(groups, "Docked"); ok {
		t.Errorf("groups = %+v, want none of the cast officer's groups", groups)
	}
	if err != nil || played.Group != "ShieldState" || played.Clip != "shields.mp3" {
		t.Fatalf("auditioned %+v, %v; want the pilot's shields take", played, err)
	}
	if last := playedSoFar(player); !slices.Equal(last[len(last)-1], take.Of(`C:\audio\shields.mp3`)) {
		t.Errorf("played %v, want the pilot's take last", last)
	}
	if app.session.active.Plugin != "Bridge Crew" || app.session.active.Name != "one" {
		t.Errorf("cast %+v, want the officer still cast", app.session.active)
	}
}

// FR-587: a moment whose call the plugin refuses counts no take and is never played; the moments it
// answers are auditioned as before.
func TestAMomentThePluginRefusesHasNoTakeOnAudition(t *testing.T) {
	app, player, _ := fixtureApp(t)
	refusing := pilot()
	refusing.Answers["Docked"] = []take.Take{take.Of(`C:\audio\docked.mp3`)}
	refusing.Refuses = []string{"ShieldState.ShieldsUp.false"}
	app.session.plugins = offering(t, "Bridge Crew", refusing)

	groups := app.PluginAuditionGroups("Bridge Crew", "two")
	_, refused := app.AuditionPluginVoice("Bridge Crew", "two", "ShieldState")

	if _, ok := groupNamed(groups, "ShieldState"); ok {
		t.Errorf("groups = %+v, want no ShieldState group for a moment refused", groups)
	}
	if docked, ok := groupNamed(groups, "Docked"); !ok || docked.Clips != 1 {
		t.Errorf("groups = %+v, want Docked counting its one take", groups)
	}
	if refused == nil || len(playedSoFar(player)) != 0 {
		t.Errorf("auditioning the refused moment answered %v and played %v; want a refusal and nothing played",
			refused, playedSoFar(player))
	}
}

// FR-585, FR-570: a voice that cannot speak has nothing to audition; nor has one no plugin offers. A
// press on either says why rather than playing silence.
func TestAPluginVoiceThatCannotSpeakHasNothingToAudition(t *testing.T) {
	app, player, _ := fixtureApp(t)
	app.session.plugins = offering(t, "Bridge Crew", plugintest.Voice{
		ID: "one", Name: "The First Officer", Reason: "its recordings are not on this machine",
	})

	for _, each := range []struct{ from, id, reason string }{
		{"Bridge Crew", "one", "not on this machine"},
		{"Bridge Crew", "nobody", "offers a voice called nobody"},
	} {
		if groups := app.PluginAuditionGroups(each.from, each.id); groups == nil || len(groups) != 0 {
			t.Errorf("%s has groups %#v, want an empty list", each.id, groups)
		}
		if _, err := app.AuditionPluginVoice(each.from, each.id, "Docked"); err == nil || !strings.Contains(err.Error(), each.reason) {
			t.Errorf("auditioning %s answered %v, want the reason %q", each.id, err, each.reason)
		}
	}
	if len(playedSoFar(player)) != 0 {
		t.Errorf("played %v, want nothing", playedSoFar(player))
	}
}

// FR-745 for a plugin voice: a moment switched off in Chatter is not drawn on.
func TestAPluginAuditionAsksChatterWhatIsSwitchedOn(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.session.plugins = offering(t, "Bridge Crew", officer())
	switchedOffDocked(t, app)

	groups := app.PluginAuditionGroups("Bridge Crew", "one")

	if docked, ok := groupNamed(groups, "Docked"); !ok || !docked.SwitchedOff || docked.Clips != 0 {
		t.Errorf("Docked = %+v, %v; want it marked switched off counting nothing", docked, ok)
	}
	if _, err := app.AuditionPluginVoice("Bridge Crew", "one", "Docked"); err == nil {
		t.Error("a group switched off in Chatter was auditioned")
	}
}
