// The audition surface: hearing a voice deliberately, rather than waiting for the game
// to say something.
//
// It sits beside app.go rather than inside it for two reasons. The facade is already
// the longest file here; auditioning touches only the voice catalogue and the player. It wires no application service, so the composition-root whitelist is
// untouched.
package main

import (
	"fmt"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// auditionGap is the pause between clips when a group is auditioned. An audition
// plays one clip, so it exists only to satisfy the player's signature.
const auditionGap = 0

// GroupDTO is one auditionable group as the pane shows it.
type GroupDTO struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Clips int    `json:"clips"`
}

// AuditionDTO reports what an audition played, so the pane can name the clip rather
// than leaving the user wondering whether anything happened.
type AuditionDTO struct {
	Group string `json:"group"`
	Clip  string `json:"clip"`
}

// AuditionGroups lists what can be auditioned for a voice, which need not be the cast
// one: the point of an audition is to hear a voice before committing to it.
func (a *App) AuditionGroups(voice string) []GroupDTO {
	chosen, found := a.session.voiceNamed(voice)
	if !found {
		return []GroupDTO{}
	}
	groups := a.session.catalogueFor(chosen).Groups()
	out := make([]GroupDTO, 0, len(groups))
	for _, group := range groups {
		out = append(out, GroupDTO{
			Key:   group.Key,
			Label: label(group.Key),
			Clips: len(group.Clips),
		})
	}
	return out
}

// Audition plays one clip drawn at random from a group.
//
// It ignores the mute, which silences the application's reactions to the game rather
// than the application. Pressing an audition button is an explicit request to hear
// something, so answering it with silence would read as a fault.
func (a *App) Audition(voice, group string) (AuditionDTO, error) {
	chosen, found := a.session.voiceNamed(voice)
	if !found {
		return AuditionDTO{}, fmt.Errorf("no voice named %q", voice)
	}
	clip, ok := a.session.catalogueFor(chosen).Audition(group)
	if !ok {
		return AuditionDTO{}, fmt.Errorf("%s has nothing for %s", voice, label(group))
	}
	if a.session.player == nil {
		return AuditionDTO{}, fmt.Errorf("there is no audio device to play through")
	}
	if err := a.session.player.Play([]string{clip}, auditionGap); err != nil {
		return AuditionDTO{}, fmt.Errorf("playing %s: %w", label(group), err)
	}
	return AuditionDTO{Group: group, Clip: clipName(clip)}, nil
}

// StopAudition ends whatever is playing, so a long clip can be cut short.
func (a *App) StopAudition() {
	if a.session.player != nil {
		a.session.player.Stop()
	}
}

// label turns a group key into the words the pane shows, by the same reading that
// heads the group in the breakdown dialog.
func label(key string) string { return cue.ID(key).Heading() }

// clipName reduces a clip's path to its file name, which is what the pane shows.
// The full path is the user's own directory and says nothing they do not know.
func clipName(path string) string {
	normalised := strings.ReplaceAll(path, "\\", "/")
	if cut := strings.LastIndexByte(normalised, '/'); cut >= 0 {
		return normalised[cut+1:]
	}
	return normalised
}
