package main

import (
	"errors"
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// errLineCount means the sounds maker answered a different number of lines than it was given,
// so its sounds could no longer be matched to lines.
var errLineCount = errors.New("the sounds maker answered a different number of lines")

// maker turns lines into speech sounds in one accent, answering in the same order.
type maker interface {
	Make(accent machinevoice.Accent, lines []string) ([]string, error)
}

// makeSounds makes every line's speech sounds in each accent, one call an accent, each line
// written with that accent's spelling (FR-529, FR-532).
func makeSounds(loaded script.Script, sounds maker) (map[string]script.Saved, error) {
	made := make(map[machinevoice.Accent][]string, len(machinevoice.Accents()))
	for _, accent := range machinevoice.Accents() {
		var written []string
		for _, id := range loaded.Cues() {
			lines, _ := loaded.Lines(id)
			for _, line := range lines {
				written = append(written, line.ForAccent(accent))
			}
		}
		answered, err := sounds.Make(accent, written)
		if err != nil {
			return nil, fmt.Errorf("making %s speech sounds: %w", accent, err)
		}
		if len(answered) != len(written) {
			return nil, fmt.Errorf("%w: %d for %d %s lines", errLineCount, len(answered), len(written), accent)
		}
		made[accent] = answered
	}
	saved := make(map[string]script.Saved, len(loaded.Cues()))
	next := 0
	for _, id := range loaded.Cues() {
		lines, _ := loaded.Lines(id)
		var entry script.Saved
		for _, line := range lines {
			entry.Lines = append(entry.Lines, line.Text())
			entry.British = append(entry.British, made[machinevoice.British][next])
			entry.American = append(entry.American, made[machinevoice.American][next])
			next++
		}
		saved[string(id)] = entry
	}
	return saved, nil
}
