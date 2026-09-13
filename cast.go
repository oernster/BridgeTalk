// The cast surface of the facade: which voices exist, which one speaks and what the
// chosen one has no lines for.
//
// It sits beside app.go rather than inside it for the reason audition.go does. The
// facade is the one file everything else hangs off, so it is first to reach the size
// cap; a pane's own questions are a slice that comes out whole.

package main

import (
	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// Voices lists every voice found in the library.
func (a *App) Voices() []VoiceDTO {
	out := make([]VoiceDTO, 0, len(a.session.available))
	for _, voice := range a.session.available {
		cues, used, present := a.session.coverageOf(voice)
		out = append(out, VoiceDTO{
			Name: voice.Name, Display: voice.Display(), Credit: voice.Credit,
			Cues: cues, InUse: used, Present: present,
		})
	}
	return out
}

// SelectVoice switches the active voice and records the choice.
//
// The choice is written down so the next run opens speaking with the same voice.
// Casting is the one deliberate act that names a voice, so it is the one place the
// name is stored; a voice that is no longer installed by then falls back at startup
// the way an unknown one on the command line does.
//
// The write is the last thing that happens, after the voice is already speaking, the
// window already knows and the voice has already said yes. A machine with nowhere to
// keep settings is a working application that forgets, so a failure here is reported
// without any of the above being undone: the cast succeeded and only the memory of it
// did not.
func (a *App) SelectVoice(name string) error {
	chosen, err := pick(a.session.available, name)
	if err != nil {
		return err
	}
	a.session.useVoice(chosen)
	a.acknowledge()
	a.emitState()
	return a.rememberVoice(chosen.Name)
}

// affirm plays one clip in which the newly cast voice agrees to the part.
//
// Casting from a list of names is a decision taken without hearing anything, so the
// only answer that settles it is the voice itself saying yes. It goes straight to the
// player rather than through the scheduler, for the reason an audition does: this
// answers a button rather than the game, so it must not queue behind what the ship is
// saying nor be dropped by the priority policy.
//
// It respects the mute, which an audition does not. An audition is a request to hear
// something; casting is a request to choose something; an application told to be
// quiet should stay quiet while being reorganised.
//
// Nothing here is worth an error. No acknowledgement material, no audio device and a
// clip that will not decode all mean the same thing to the commander: no confirming
// sound, on a cast that has already succeeded.
func (a *App) acknowledge() {
	if a.session.muted || a.session.player == nil || a.session.catalogue == nil {
		return
	}
	clip, ok := a.session.catalogue.Acknowledgement()
	if !ok {
		return
	}
	// Casting ends what was playing on purpose, so this is Play rather than PlayIfIdle.
	if a.session.player.Play([]string{clip}, auditionGap) == nil {
		a.announcePlayback()
	}
}

// CueBreakdown is one voice's whole relationship with the cue table: what it can
// speak for and what it cannot.
//
// It answers for ANY voice rather than only the cast one, because the question is
// asked from a row in the chooser and the point of asking is to decide whether to
// cast it. A name that matches nothing yields an empty breakdown rather than an
// error: the dialog behind it is a thing to read, so it has nothing to report.
func (a *App) CueBreakdown(name string) CueBreakdownDTO {
	chosen, err := pick(a.session.available, name)
	if err != nil {
		return CueBreakdownDTO{Voice: name}
	}
	catalogue := a.session.catalogueFor(chosen)
	return CueBreakdownDTO{
		Voice:    chosen.Name,
		Served:   cueLines(catalogue.Served()),
		Unserved: cueLines(catalogue.Unbound()),
	}
}

// cueLines maps cues into the shape the front end reads.
func cueLines(cues []cue.Cue) []CueDTO {
	out := make([]CueDTO, 0, len(cues))
	for _, item := range cues {
		out = append(out, CueDTO{
			ID:      string(item.ID()),
			Title:   item.Title(),
			Folder:  item.ID().Folder(),
			Purpose: item.Purpose(),
		})
	}
	return out
}
