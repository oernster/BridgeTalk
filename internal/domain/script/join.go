package script

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// ErrUnjoinable is returned for a joining line whose speech sounds do not end with the comma, the
// word's spelling and the final mark, so the comma cannot be left out of them (FR-550).
var ErrUnjoinable = errors.New("speech sounds cannot be joined")

// commaBefore is what stands between a joined word and the word before it in a line and in its
// speech sounds: a comma and a space (FR-550).
const commaBefore = ", "

// finalMarks are the marks a joining line may end with, exactly one of them (FR-550).
var finalMarks = []string{".", "!", "?"}

// SavedLine is one line of the script with its saved speech sounds in one accent: a line FR-550
// joins or a line ending on a nasal (FR-555).
type SavedLine struct {
	// Cue is the cue the line is spoken for.
	Cue cue.ID
	// Index is the line's place among its cue's lines, from zero.
	Index int
	// Sounds is the line's saved speech sounds in the accent asked for.
	Sounds string
}

// WithWords gives the script its table of words (FR-549) and the words joined after a final comma
// (FR-550). A table that cannot be read or a joined word it gives no sounds refuses the script,
// naming the word; every joined word missing is named together.
func (s Script) WithWords(spellings map[string][]string, joinAfterComma []string) (Script, error) {
	words, err := speech.NewWords(spellings)
	if err != nil {
		return Script{}, fmt.Errorf("%w: %w (FR-549)", ErrInvalidScript, err)
	}
	var problems []error
	for _, word := range joinAfterComma {
		if _, given := words.Sounds(word, machinevoice.British); !given {
			problems = append(problems, fmt.Errorf(
				"%w: %q is joined after a comma but the table of words gives it no speech sounds (FR-550)",
				ErrInvalidScript, word,
			))
		}
	}
	if len(problems) > 0 {
		return Script{}, errors.Join(problems...)
	}
	s.words = words
	s.joinAfterComma = slices.Clone(joinAfterComma)
	return s, nil
}

// Words returns the script's table of words, which spells each line for the sounds maker
// (FR-549).
func (s Script) Words() speech.Words { return s.words }

// joining returns the word a line joins and the mark it ends with, reporting whether it joins: its
// spoken words end with a comma, a joined word and exactly one final mark (FR-550).
func (s Script) joining(line speech.Line) (string, string, bool) {
	spoken := line.Words()
	for _, word := range s.joinAfterComma {
		for _, mark := range finalMarks {
			if strings.HasSuffix(spoken, commaBefore+word+mark) {
				return word, mark, true
			}
		}
	}
	return "", "", false
}

// Join leaves the comma and its space before a joined word out of a line's speech sounds in one
// accent, finding the word by the table's spelling; the sounds of a line that does not join are
// returned as they are (FR-550).
func (s Script) Join(line speech.Line, accent machinevoice.Accent, sounds string) (string, error) {
	word, mark, joins := s.joining(line)
	if !joins {
		return sounds, nil
	}
	spelling, _ := s.words.Sounds(word, accent)
	ending := commaBefore + spelling + mark
	if !strings.HasSuffix(sounds, ending) {
		return "", fmt.Errorf("%w: %q does not end %q (FR-550)", ErrUnjoinable, sounds, ending)
	}
	return strings.TrimSuffix(sounds, ending) + spelling + mark, nil
}

// Joined returns every line the script joins with its saved speech sounds in one accent, cue by
// cue in the order Cues gives, then line by line (FR-550).
func (v Voiced) Joined(accent machinevoice.Accent) []SavedLine {
	return v.linesWhere(accent, func(line speech.Line, _ string) bool {
		_, _, joins := v.joining(line)
		return joins
	})
}
