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
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/infrastructure/library"
)

// castVoice is the voice speaking: the name that identifies it, the name it is shown by and whether
// it is a machine voice.
type castVoice struct {
	Name    string
	Display string
	Machine bool
}

// releaser is the model's hold on memory, let go when the application closes.
type releaser interface{ Close() }

// CastMachineVoice casts the machine voice with the id given, which starts making every line it has
// no current made line for (FR-511).
//
// A voice that is not offered is refused with nothing changed; so is one whose files cannot be read
// (FR-519).
// Otherwise the voice confirms in a made line where one is current (FR-521), the page is told and
// the voice is kept for the next run (FR-540). As with a recorded voice, a failure to keep it is
// reported without the cast being undone.
func (a *App) CastMachineVoice(id string) error {
	if err := a.session.castMachine(id); err != nil {
		return err
	}
	a.acknowledge()
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
func (s *session) castAtStart(machineID string, recorded library.Voice, warnings io.Writer) {
	if machineID != "" {
		err := s.castMachine(machineID)
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

// release stops making, keeping every line written (FR-517), then lets the model go. Nothing is made
// after it, so it comes last in a shutdown.
func (s *session) release() {
	s.making.Stop()
	s.maker.Close()
}
