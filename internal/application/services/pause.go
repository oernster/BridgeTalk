package services

import (
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/pause"
)

// withPause answers a made line's samples as they are to be written (FR-553). A line whose pause is
// not doubtful gets the book's silence inserted at the pause's sample where its samples are those the
// pause was found in; where they differ, the samples are answered as made and the line is logged. A
// pause outside the samples, which only a corrupt book could give, answers why: the line cannot be
// made (FR-518).
func (m *MakingService) withPause(voice machinevoice.Voice, line making.Line, samples []float32) ([]float32, error) {
	if !line.Paused() {
		return samples, nil
	}
	if made := pause.Digest(samples); made != line.Pause.Digest {
		m.log.Log(fmt.Sprintf(
			"%s was made with samples of digest %q where its pause was found in samples of digest %q; it is written without its pause (FR-553)",
			m.lineName(voice, line), made, line.Pause.Digest,
		))
		return samples, nil
	}
	return pause.Insert(samples, line.Pause.Sample, m.pauses.Silence())
}

// lineName names a voice's line as the script counts its lines, from one, with its text:
// bf_emma "Docked" line 1 "Docked, commander.". The plan's lines are the script's, so the line is
// always had.
func (m *MakingService) lineName(voice machinevoice.Voice, line making.Line) string {
	lines, _ := m.voiced.Lines(line.Cue)
	return fmt.Sprintf("%s %q line %d %q", voice.ID(), line.Cue, line.Index+1, lines[line.Index].Text())
}
