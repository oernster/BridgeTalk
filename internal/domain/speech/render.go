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
	return Words{}.Spell(l, accent)
}

// spell writes a word with its spellings in the form a line gives them: [word](/sounds/) or
// [word](/British/American/) (FR-529).
func spell(word string, spellings ...string) string {
	return opening + word + joining + slash + strings.Join(spellings, slash) + slash + closing
}
