// Package script holds the words every machine voice speaks: the lines the script gives each
// cue, checked against the cue table.
package script

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// ErrInvalidScript is returned for a script that cannot be spoken as it stands.
var ErrInvalidScript = errors.New("invalid script")

// LinesPerCue is how many lines the script gives each cue it names, so an event heard often
// does not sound the same each time (FR-505).
const LinesPerCue = 3

// Script is the lines each cue is spoken with.
type Script struct {
	byCue map[cue.ID][]speech.Line
}

// New builds a script from cue ids to lines, checked against the table. Every problem is
// returned together, each naming its cue and its line where it has one.
func New(entries map[string][]string, table cue.Table) (Script, error) {
	known := make(map[cue.ID]bool, table.Len())
	for _, item := range table.All() {
		known[item.ID()] = true
	}
	byCue := make(map[cue.ID][]speech.Line, len(entries))
	var problems []error
	for _, key := range slices.Sorted(maps.Keys(entries)) {
		if !known[cue.ID(key)] {
			problems = append(problems, fmt.Errorf("%w: %q is not a cue (FR-504)", ErrInvalidScript, key))
			continue
		}
		lines, found := readLines(key, entries[key])
		byCue[cue.ID(key)] = lines
		problems = append(problems, found...)
	}
	if len(problems) > 0 {
		return Script{}, errors.Join(problems...)
	}
	return Script{byCue: byCue}, nil
}

// readLines reads one cue's lines, returning every problem they hold.
func readLines(key string, texts []string) ([]speech.Line, []error) {
	var problems []error
	if len(texts) != LinesPerCue {
		problems = append(problems, fmt.Errorf(
			"%w: %q holds %d lines, not %d (FR-505)", ErrInvalidScript, key, len(texts), LinesPerCue,
		))
	}
	seen := make(map[string]bool, len(texts))
	lines := make([]speech.Line, 0, len(texts))
	for _, text := range texts {
		switch {
		case strings.TrimSpace(text) == "":
			problems = append(problems, fmt.Errorf("%w: %q holds an empty line (FR-505)", ErrInvalidScript, key))
			continue
		case seen[text]:
			problems = append(problems, fmt.Errorf("%w: %q holds %q twice (FR-505)", ErrInvalidScript, key, text))
			continue
		}
		seen[text] = true
		line, err := speech.Read(text)
		if err != nil {
			problems = append(problems, fmt.Errorf("%w: %q line %q: %w", ErrInvalidScript, key, text, err))
			continue
		}
		lines = append(lines, line)
	}
	return lines, problems
}

// Lines returns the lines the script gives a cue, reporting whether it gives any. The slice is
// the caller's own.
func (s Script) Lines(id cue.ID) ([]speech.Line, bool) {
	lines, ok := s.byCue[id]
	return slices.Clone(lines), ok
}

// Cues returns the cues the script gives lines, sorted, so anything written from the script is
// written in a stable order.
func (s Script) Cues() []cue.ID {
	return slices.Sorted(maps.Keys(s.byCue))
}

// Missing returns the cues the table holds that the script gives no lines, in the table's
// order (FR-507).
func (s Script) Missing(table cue.Table) []cue.ID {
	var missing []cue.ID
	for _, item := range table.All() {
		if _, ok := s.byCue[item.ID()]; !ok {
			missing = append(missing, item.ID())
		}
	}
	return missing
}
