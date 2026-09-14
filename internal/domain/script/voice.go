package script

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// Saved is what the sounds tool saved for one cue: its lines as they were written when the tool
// ran, with their speech sounds in each accent in the same order (FR-532).
type Saved struct {
	Lines    []string
	British  []string
	American []string
}

// forAccent returns the speech sounds saved for one accent.
func (s Saved) forAccent(accent machinevoice.Accent) []string {
	if accent == machinevoice.American {
		return s.American
	}
	return s.British
}

// Voiced is a script whose every line has speech sounds saved for it in each accent, made from
// its current text and fit for the model.
type Voiced struct {
	Script
	saved map[cue.ID]Saved
}

// Voice checks saved speech sounds against the script they were made from, returning every
// problem together (FR-506, FR-533).
func Voice(built Script, saved map[string]Saved) (Voiced, error) {
	var problems []error
	for _, id := range built.Cues() {
		entry, ok := saved[string(id)]
		if !ok {
			problems = append(problems, fmt.Errorf("%w: %q has no saved speech sounds (FR-533)", ErrInvalidScript, id))
			continue
		}
		problems = append(problems, checkSaved(id, built.byCue[id], entry)...)
	}
	for _, key := range slices.Sorted(maps.Keys(saved)) {
		if _, ok := built.byCue[cue.ID(key)]; !ok {
			problems = append(problems, fmt.Errorf(
				"%w: saved speech sounds remain for %q, which the script no longer holds (FR-533)",
				ErrInvalidScript, key,
			))
		}
	}
	if len(problems) > 0 {
		return Voiced{}, errors.Join(problems...)
	}
	kept := make(map[cue.ID]Saved, len(saved))
	for key, entry := range saved {
		kept[cue.ID(key)] = entry
	}
	return Voiced{Script: built, saved: kept}, nil
}

// checkSaved returns every problem with one cue's saved speech sounds. Sounds are only checked
// once the lines they were made from match the script, since until then they belong to other
// lines.
func checkSaved(id cue.ID, lines []speech.Line, entry Saved) []error {
	var problems []error
	texts := make([]string, 0, len(lines))
	for index, line := range lines {
		texts = append(texts, line.Text())
		if index >= len(entry.Lines) || entry.Lines[index] != line.Text() {
			problems = append(problems, fmt.Errorf(
				"%w: %q line %q has no saved speech sounds made from its text (FR-533)",
				ErrInvalidScript, id, line.Text(),
			))
		}
	}
	for _, text := range entry.Lines {
		if !slices.Contains(texts, text) {
			problems = append(problems, fmt.Errorf(
				"%w: saved speech sounds remain for %q line %q, which the script no longer holds (FR-533)",
				ErrInvalidScript, id, text,
			))
		}
	}
	if len(problems) > 0 {
		return problems
	}
	for _, accent := range machinevoice.Accents() {
		sounds := entry.forAccent(accent)
		if len(sounds) != len(lines) {
			problems = append(problems, fmt.Errorf(
				"%w: %q has %d %s speech sounds for %d lines (FR-533)",
				ErrInvalidScript, id, len(sounds), accent, len(lines),
			))
			continue
		}
		for index, each := range sounds {
			if _, err := speech.Tokens(each); err != nil {
				problems = append(problems, fmt.Errorf(
					"%w: %q line %q %s speech sounds: %w", ErrInvalidScript, id, lines[index].Text(), accent, err,
				))
			}
		}
	}
	return problems
}

// Sounds returns the speech sounds saved for a cue's lines in one accent, in line order,
// reporting whether the script gives the cue any. The slice is the caller's own.
func (v Voiced) Sounds(id cue.ID, accent machinevoice.Accent) ([]string, bool) {
	entry, ok := v.saved[id]
	return slices.Clone(entry.forAccent(accent)), ok
}
