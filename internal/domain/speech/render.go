package speech

import (
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
)

// Text returns the line as it was written, spellings and all, so speech sounds saved for it can
// be matched to it.
func (l Line) Text() string { return l.text }

// ForAccent writes the line for one accent: each spelled word carries that accent's one spelling
// as [word](/sounds/); plain words stand as they are (FR-529).
func (l Line) ForAccent(accent machinevoice.Accent) string {
	var written strings.Builder
	for _, piece := range l.pieces {
		if !piece.Given() {
			written.WriteString(piece.text)
			continue
		}
		written.WriteString(opening + piece.text + joining + slash + piece.Sounds(accent) + slash + closing)
	}
	return written.String()
}
