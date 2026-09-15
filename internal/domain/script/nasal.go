package script

import (
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// EndingOnNasal returns every line whose last speech sound in one accent is a nasal, with its saved
// speech sounds in that accent, cue by cue in the order Cues gives, then line by line (FR-555).
func (v Voiced) EndingOnNasal(accent machinevoice.Accent) []SavedLine {
	return v.linesWhere(accent, func(_ speech.Line, sounds string) bool { return speech.EndsOnNasal(sounds) })
}
