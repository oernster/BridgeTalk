// The audition surface: hearing a voice deliberately, rather than waiting for the game
// to say something.
//
// It sits beside app.go rather than inside it for two reasons. The facade is already
// the longest file here; auditioning touches only the voice catalogue and the player. It wires no application service, so the composition-root whitelist is
// untouched.
package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/take"
)

// auditionGap is the pause between the parts of the take a group is auditioned with. It
// is zero for the reason takeGap is: a take recorded in pieces is one utterance (FR-573).
const auditionGap = 0

// GroupDTO is one auditionable group as the pane shows it. Clips counts the takes or lines of its
// moments switched on in Chatter (FR-746); SwitchedOff says the voice has something for the group yet
// every one of its moments is switched off, which the pane leaves out (FR-747, FR-748). Category is the
// Chatter category the group is listed under, empty for none (FR-749, FR-750).
type GroupDTO struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Clips       int    `json:"clips"`
	SwitchedOff bool   `json:"switchedOff"`
	Category    string `json:"category"`
}

// AuditionDTO reports what an audition played, so the pane can name the clip rather
// than leaving the user wondering whether anything happened.
type AuditionDTO struct {
	Group string `json:"group"`
	Clip  string `json:"clip"`
}

// AuditionGroups lists what can be auditioned for a voice, which need not be the cast
// one: the point of an audition is to hear a voice before committing to it. Only the
// moments switched on in Chatter are counted (FR-746).
func (a *App) AuditionGroups(voice string) []GroupDTO {
	chosen, found := a.session.voiceNamed(voice)
	if !found {
		return []GroupDTO{}
	}
	groups := a.session.catalogueFor(chosen).Groups(a.session.heard())
	out := make([]GroupDTO, 0, len(groups))
	for _, group := range groups {
		out = append(out, a.session.groupShown(group.Key, len(group.Takes), group.SwitchedOff))
	}
	return a.session.inCategoryOrder(out)
}

// groupShown is one group as the pane shows it: its key read as words, its count, whether Chatter has
// switched it off and the category it is listed under (FR-749).
func (s *session) groupShown(key string, clips int, switchedOff bool) GroupDTO {
	return GroupDTO{
		Key: key, Label: label(key), Clips: clips, SwitchedOff: switchedOff,
		Category: s.table.GroupCategory(key),
	}
}

// inCategoryOrder answers groups in Chatter's category order, a group in no category last; within a
// category they keep the order they came in, which is their keys' (FR-216, FR-750).
func (s *session) inCategoryOrder(groups []GroupDTO) []GroupDTO {
	categories := s.table.Categories()
	place := func(group GroupDTO) int {
		if at := slices.Index(categories, group.Category); at >= 0 {
			return at
		}
		return len(categories)
	}
	slices.SortStableFunc(groups, func(first, second GroupDTO) int { return place(first) - place(second) })
	return groups
}

// Audition plays one clip drawn at random from a group's moments switched on in Chatter (FR-745).
//
// It ignores the mute, which silences the application's reactions to the game rather
// than the application. Pressing an audition button is an explicit request to hear
// something, so answering it with silence would read as a fault.
func (a *App) Audition(voice, group string) (AuditionDTO, error) {
	chosen, found := a.session.voiceNamed(voice)
	if !found {
		return AuditionDTO{}, fmt.Errorf("no voice named %q", voice)
	}
	drawn, ok := a.session.catalogueFor(chosen).Audition(group, a.session.heard())
	if !ok {
		return AuditionDTO{}, nothingFor(voice, group)
	}
	return a.play(group, drawn)
}

// nothingFor refuses a group a voice has nothing for, naming the group in the words the pane shows.
func nothingFor(voice, group string) error {
	return fmt.Errorf("%s has nothing for %s", voice, label(group))
}

// play plays the take an audition drew from a group, answering what it played.
func (a *App) play(group string, chosen take.Take) (AuditionDTO, error) {
	// FR-236: a press never cuts short what is already sounding, whether an earlier
	// audition or the ship speaking. The question and the start are one call on the
	// player, since asking first and playing second leaves a gap another caller can use.
	started, err := a.session.player.PlayIfIdle(chosen, auditionGap)
	if err != nil {
		return AuditionDTO{}, fmt.Errorf("playing %s: %w", label(group), err)
	}
	if !started {
		// Ignored rather than refused: the pane holds its buttons while anything plays,
		// so this is a press that raced the event; it says nothing.
		return AuditionDTO{}, nil
	}
	a.announcePlayback()
	// The pane names the take by its first part, which is the take's identity (take.Take.Key).
	return AuditionDTO{Group: group, Clip: clipName(chosen.Key())}, nil
}

// StopAudition ends whatever is playing, so a long clip can be cut short. A machine voice's
// line still being made for an audition is let go: it is kept once written but never played,
// and the page is told nothing is under way (FR-547).
func (a *App) StopAudition() {
	letGo := a.session.auditions.stop()
	a.session.player.Stop()
	if letGo {
		a.announcePlayback()
	}
}

// Playing reports whether the device is sounding anything or a machine voice's line is being
// made for an audition. A pane opened part way through either asks, so its buttons are held
// from the start rather than from the next event (FR-236, FR-547).
func (a *App) Playing() bool {
	return a.session.auditions.busy() || a.session.player.Playing()
}

// announcePlayback tells the front end whether anything is sounding. The run loop
// announces each end; a start is announced from here, so the audition buttons are held
// for as long as a clip plays (FR-236).
func (a *App) announcePlayback() {
	a.emit(playbackEvent, PlaybackDTO{Playing: a.Playing()})
}

// pollAndAnnounce polls, then announces a start where the poll set something playing,
// so a reaction to the game holds the audition buttons exactly as an audition does.
func (a *App) pollAndAnnounce() {
	was := a.Playing()
	a.poll()
	a.tickMaking()
	if !was && a.Playing() {
		a.announcePlayback()
	}
	a.announceMaking()
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
