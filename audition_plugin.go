// The plugin voice half of the audition surface: a plugin voice auditioned on the groups of the takes
// its plugin answers, without being cast (FR-585 to FR-587).
//
// It sits beside audition.go for the reason audition_machine.go does: a kind of voice's own questions
// are a slice that comes out whole.

package main

import (
	"fmt"

	"github.com/oernster/bridge-talk/internal/infrastructure/library"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
)

// PluginAuditionGroups lists what the plugin voice named by its plugin and its id can be auditioned on,
// each group counting the takes of its moments switched on in Chatter (FR-586, FR-746). A voice no
// loaded plugin offers has nothing to list; neither has one that cannot speak.
//
// A moment whose call the plugin refuses counts no take; so does one whose answer will not read. The
// plugin voice answers such a call with nothing, as it does for the cast voice (FR-587).
func (a *App) PluginAuditionGroups(from, id string) []GroupDTO {
	voice, err := a.session.pluginVoice(from, id)
	if err != nil {
		return []GroupDTO{}
	}
	return a.session.groupsShown(a.session.pluginCatalogue(voice).Groups(a.session.heard()))
}

// AuditionPluginVoice plays one take drawn at random from a group's moments switched on in Chatter, for
// the plugin voice named by its plugin and its id, leaving the cast voice as it was (FR-586). It plays
// while muted as every audition does.
func (a *App) AuditionPluginVoice(from, id, group string) (AuditionDTO, error) {
	voice, err := a.session.pluginVoice(from, id)
	if err != nil {
		return AuditionDTO{}, err
	}
	drawn, ok := a.session.pluginCatalogue(voice).Audition(group, a.session.heard())
	if !ok {
		return AuditionDTO{}, nothingFor(voice.Name, group)
	}
	return a.play(group, drawn)
}

// pluginCatalogue is a catalogue over a plugin voice that need not be the cast one, which is what lets
// it be heard before it is cast. It is built on each question rather than kept: the plugin keeps its
// own answers, so nothing is saved by holding it.
func (s *session) pluginCatalogue(voice *plugin.Voice) *library.Catalogue {
	return catalogueOver(voice, voice.Name, s.table, s.chooser)
}

// pluginVoice finds the voice a plugin offers by the plugin's own name and the voice's id within it
// (FR-569). A voice whose audio is not on this machine is refused with the reason the plugin gave
// (FR-570), as is one no loaded plugin offers.
func (s *session) pluginVoice(from, id string) (*plugin.Voice, error) {
	if s.plugins == nil {
		return nil, fmt.Errorf("no plugin offers a voice")
	}
	for _, voice := range s.plugins.Voices() {
		if voice.Plugin().Name != from || voice.ID != id {
			continue
		}
		if !voice.Ready {
			return nil, fmt.Errorf("%s cannot speak: %s", voice.Name, voice.Reason)
		}
		return voice, nil
	}
	return nil, fmt.Errorf("no plugin named %s offers a voice called %s", from, id)
}
