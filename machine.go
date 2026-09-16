// The machine voice surface of the facade and the session: casting one from the page or at start,
// refusing every one where there is nowhere to keep its lines and releasing the model on the way out.
//
// It sits beside cast.go for the reason cast.go sits beside app.go: a kind of voice's own questions
// are a slice that comes out whole.

package main

import (
	"fmt"
	"io"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/infrastructure/library"
)

// castVoice is the voice speaking: the name that identifies it, the name it is shown by and whether
// it is a machine voice.
type castVoice struct {
	Name    string
	Display string
	Machine bool
	// Plugin names the plugin a voice came from; empty for every other kind. With Name, which
	// holds the voice's id within that plugin, it is what the settings keep (FR-569).
	Plugin string
}

// releaser is the model's hold on memory, let go when the application closes.
type releaser interface{ Close() }

// makingEvent is emitted when making the cast machine voice's lines has moved.
const makingEvent = "making"

// MachineVoices lists every machine voice offered, in the order FR-508 gives, by the name the screen
// shows (FR-528) with the name alone and the group its pill is offered in (FR-720). Every one can be
// cast; one whose files cannot be read is refused when it is.
func (a *App) MachineVoices() []MachineVoiceDTO {
	offered := machinevoice.All()
	out := make([]MachineVoiceDTO, 0, len(offered))
	for _, voice := range offered {
		out = append(out, MachineVoiceDTO{ID: voice.ID(), Name: voice.Name(), Given: voice.Given(), Group: voice.Group()})
	}
	return out
}

// Making reports how far making the cast machine voice's lines has got (FR-515, FR-518, FR-520,
// FR-522, FR-530). The failures always cross as a list, empty where there are none, so the page
// never reads a missing one.
func (a *App) Making() MakingDTO {
	progress := a.session.making.Progress()
	failed := make([]LineFailureDTO, 0, len(progress.Failed))
	for _, failure := range progress.Failed {
		failed = append(failed, LineFailureDTO{
			Cue: a.session.cueEntry(failure.Cue), Line: failure.Index + 1, Reason: failure.Reason.Error(),
		})
	}
	return MakingDTO{
		Voice: progress.Voice, Making: progress.Making,
		Current: progress.Current, Total: progress.Total, CuesServed: progress.CuesServed,
		Failed: failed, Stopped: said(progress.Stopped), NotDeleted: said(progress.NotDeleted),
	}
}

// said words a reason for the page; empty where there is none.
func said(reason error) string {
	if reason == nil {
		return ""
	}
	return reason.Error()
}

// cueEntry names a cue for a reader by its id, with the purpose the cue table gives it. Every id it
// is asked about is in the table: a line's cue because the script is checked against it (FR-504); a
// reaction's cue because the reaction was matched from it.
func (s *session) cueEntry(id cue.ID) CueDTO {
	var purpose string
	for _, item := range s.table.All() {
		if item.ID() == id {
			purpose = item.Purpose()
			break
		}
	}
	return cueLine(id, purpose)
}

// makingKey is what the page is told about making, reduced to what can be compared: a change in
// any of it is news; a tick that changes none of it is not.
type makingKey struct {
	voice                          string
	making                         bool
	current, total, served, failed int
	stopped, notDeleted            string
}

// announceMaking tells the page how far making has got, once for each change (FR-515). The poll
// loop calls it on every tick, so an answer that has not moved says nothing.
func (a *App) announceMaking() {
	now := a.Making()
	key := makingKey{
		voice: now.Voice, making: now.Making, current: now.Current, total: now.Total,
		served: now.CuesServed, failed: len(now.Failed), stopped: now.Stopped, notDeleted: now.NotDeleted,
	}
	if key == a.session.announced {
		return
	}
	a.session.announced = key
	a.emit(makingEvent, now)
}

// CastMachineVoice casts the machine voice with the id given, which makes its confirmation's lines not
// yet made and loads the model (FR-511, FR-544).
//
// A voice that is not offered is refused with nothing changed; so is one whose files cannot be read
// (FR-519).
// Otherwise the voice confirms in a made line as soon as one is current (FR-521), the page is told
// and the voice is kept for the next run (FR-540). As with a recorded voice, a failure to keep it is
// reported without the cast being undone.
func (a *App) CastMachineVoice(id string) error {
	if err := a.session.castMachine(id); err != nil {
		return err
	}
	a.session.confirming.Store(true)
	a.confirmWhenMade()
	a.emitState()
	return a.rememberMachineVoice(id)
}

// castMachine casts a machine voice by id: its lines start being made and it speaks with those made
// so far (FR-514). Nothing changes where the voice is refused.
func (s *session) castMachine(id string) error {
	voice, err := machinevoice.Parse(id)
	if err != nil {
		return err
	}
	source, err := s.making.Cast(voice)
	if err != nil {
		return err
	}
	s.speakWith(source, castVoice{Name: voice.ID(), Display: voice.Name(), Machine: true})
	return nil
}

// castAtStart casts the voice a run opens with: the kept machine voice where there is one (FR-512),
// otherwise the recorded voice chosen for the run.
//
// A machine voice that cannot be cast is warned about rather than stopping the start; the recorded
// voice is then cast as though none were kept (FR-541). With no recorded voice found nothing is cast,
// which the window already copes with.
func (s *session) castAtStart(kept keptCast, recorded library.Voice, warnings io.Writer) {
	if kept.plugin != "" {
		err := s.castPlugin(kept.plugin, kept.pluginVoice)
		if err == nil {
			return
		}
		fmt.Fprintf(warnings, "warning: %v (casting no plugin voice)\n", err)
	}
	if kept.machine != "" {
		err := s.castMachine(kept.machine)
		if err == nil {
			return
		}
		fmt.Fprintf(warnings, "warning: %v (casting no machine voice)\n", err)
	}
	if len(s.available) > 0 {
		s.useVoice(recorded)
	}
}

// keptMachineVoice answers the machine voice a run opens with: the one kept, unless -voice names a
// voice for this run (FR-540, FR-701).
func keptMachineVoice(flagged, kept string) string {
	if flagged != "" {
		return ""
	}
	return kept
}

// keptCast is the voice the settings kept, of whichever kind. At most one of them holds a
// voice (FR-540, FR-569).
type keptCast struct {
	machine     string
	plugin      string
	pluginVoice string
}

// keptFrom answers what was kept, unless a voice was named on the command line.
//
// A flag is this run's instruction and it names a recorded voice, so it sets aside every kept
// voice of another kind rather than only the machine one: honouring the flag and then casting
// something else would be neither.
func keptFrom(flagged string, held ports.Settings) keptCast {
	return keptCast{
		machine:     keptMachineVoice(flagged, held.MachineVoice),
		plugin:      keptMachineVoice(flagged, held.Plugin),
		pluginVoice: keptMachineVoice(flagged, held.PluginVoice),
	}
}

// castPlugin casts the voice a plugin offers, by the plugin's own name and the voice's id
// within it (FR-569).
//
// A voice whose audio is not on this machine is refused rather than cast, with the reason the
// plugin gave, since casting it would be casting silence (FR-570).
func (s *session) castPlugin(plugin, id string) error {
	if s.plugins == nil {
		return fmt.Errorf("no plugin offers a voice")
	}
	for _, voice := range s.plugins.Voices() {
		if voice.Plugin().Name != plugin || voice.ID != id {
			continue
		}
		if !voice.Ready {
			return fmt.Errorf("%s cannot speak: %s", voice.Name, voice.Reason)
		}
		s.making.CastRecorded()
		s.speakWith(voice, castVoice{Name: voice.ID, Display: voice.Name, Plugin: plugin})
		return nil
	}
	return fmt.Errorf("no plugin named %s offers a voice called %s", plugin, id)
}

// refusedFiles refuses every machine voice, saying why none can be cast.
type refusedFiles struct{ reason error }

// Open refuses the voice.
func (r refusedFiles) Open(machinevoice.Voice) (ports.Material, error) {
	return ports.Material{}, fmt.Errorf("no machine voice can be cast: %w", r.reason)
}

// filesUnless answers the files a machine voice is read from; where the application or the made
// lines' folder could not be found, files that refuse every voice with why (FR-523, FR-539). A cast
// reads the files before it touches the store, so nothing is written or deleted.
func filesUnless(files ports.VoiceFiles, unavailable error) ports.VoiceFiles {
	if unavailable != nil {
		return refusedFiles{reason: unavailable}
	}
	return files
}

// tickMaking runs on every poll tick: it hands over each cue whose line was written since it fired
// (FR-514), then plays a confirmation written since the cast (FR-521).
func (a *App) tickMaking() {
	if a.session.hasVoice() {
		a.session.reactions.Tick()
	}
	a.confirmWhenMade()
}

// confirmWhenMade plays the cast machine voice's confirmation once one of its lines is current
// (FR-521): at once where one already is, otherwise on the poll tick after it is written. It stops
// waiting once the confirmation is had, once playback is muted or no device is open, once making has
// ended without one and once another voice is cast (session.speakWith).
//
// Whether making is under way is read before the confirmation is looked for: a line written between
// the two is then found rather than taken for a making that ended without it.
func (a *App) confirmWhenMade() {
	s := a.session
	if !s.confirming.Load() {
		return
	}
	making := s.making.Progress().Making
	if a.acknowledge() || s.muted || s.player == nil || !making {
		s.confirming.Store(false)
	}
}

// release stops making, keeping every line written (FR-517), then lets the model go. Nothing is made
// after it, so it comes last in a shutdown.
func (s *session) release() {
	s.making.Stop()
	s.maker.Close()
	if s.plugins != nil {
		s.plugins.Close()
	}
}
