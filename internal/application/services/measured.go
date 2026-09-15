package services

import (
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/pause"
)

// measure answers a made line's samples as they are to be written: faded where its ending fades
// (FR-556), then with the book's silence inserted where it is paused (FR-553). Each change is made
// only where the samples as made are those it was measured in; where they differ, that change is left
// out and the line is logged. A fade or a pause outside the samples, which only a corrupt book could
// give, answers why: the line cannot be made (FR-518).
func (m *MakingService) measure(voice machinevoice.Voice, line making.Line, samples []float32) ([]float32, error) {
	if !line.Faded() && !line.Paused() {
		return samples, nil
	}
	made := pause.Digest(samples)
	written := samples
	if line.Faded() && m.measuredIn(voice, line, made, line.Ending.Digest, "ending", "fade", "FR-556") {
		faded, err := ending.FadeOut(samples, line.Ending.Sample, m.endings.Fade())
		if err != nil {
			return nil, err
		}
		written = faded
	}
	if line.Paused() && m.measuredIn(voice, line, made, line.Pause.Digest, "pause", "pause", "FR-553") {
		return pause.Insert(written, line.Pause.Sample, m.pauses.Silence())
	}
	return written, nil
}

// measuredIn reports whether the digest of the samples made is the one a change to the line was
// measured in, logging the line where it is not: its voice, its cue, its line with its text, both
// digests, what was measured, the change left out and the requirement.
func (m *MakingService) measuredIn(voice machinevoice.Voice, line making.Line, made, found, measured, change, rule string) bool {
	if made == found {
		return true
	}
	m.log.Log(fmt.Sprintf(
		"%s was made with samples of digest %q where its %s was found in samples of digest %q; it is written without its %s (%s)",
		m.lineName(voice, line), made, measured, found, change, rule,
	))
	return false
}

// lineName names a voice's line as the script counts its lines, from one, with its text:
// bf_emma "Docked" line 1 "Docked, commander.". The plan's lines are the script's, so the line is
// always had.
func (m *MakingService) lineName(voice machinevoice.Voice, line making.Line) string {
	lines, _ := m.voiced.Lines(line.Cue)
	return fmt.Sprintf("%s %q line %d %q", voice.ID(), line.Cue, line.Index+1, lines[line.Index].Text())
}
