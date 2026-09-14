package pause

import (
	"errors"
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// ErrStale is returned for each way the pauses no longer match the script, the voices or the files
// the lines are made from (FR-554).
var ErrStale = errors.New("stale pauses")

// Check returns every way the book is stale against the shipped script, the voices given and the
// digests the list gives the model file and each voice's style file, each naming what is stale
// (FR-554). The model comes first, then each voice in the order given: its style file, its joined
// lines in the script's order, then its pauses for lines that no longer join.
func Check(book Book, voices []machinevoice.Voice, voiced script.Voiced, model string, styles map[string]string) []error {
	var problems []error
	if book.model != model {
		problems = append(problems, fmt.Errorf(
			"%w: the pauses were found with the model digest %q where the list gives %q (FR-554)",
			ErrStale, book.model, model,
		))
	}
	for _, voice := range voices {
		problems = append(problems, checkVoice(book, voice, voiced, styles[voice.ID()])...)
	}
	return problems
}

// checkVoice returns every way one voice's pauses are stale against the lines the script joins in
// its accent and the digest the list gives its style file.
func checkVoice(book Book, voice machinevoice.Voice, voiced script.Voiced, style string) []error {
	id := voice.ID()
	given, ok := book.voices[id]
	if !ok {
		return []error{fmt.Errorf("%w: %s has no pauses (FR-554)", ErrStale, id)}
	}
	var problems []error
	if given.Style != style {
		problems = append(problems, fmt.Errorf(
			"%w: %s's pauses were found with the style digest %q where the list gives %q (FR-554)",
			ErrStale, id, given.Style, style,
		))
	}
	joins := make(map[place]bool)
	for _, line := range voiced.Joined(voice.Accent()) {
		key := place{cue: line.Cue, index: line.Index}
		joins[key] = true
		at, paused := book.places[id][key]
		switch {
		case !paused:
			problems = append(problems, fmt.Errorf(
				"%w: %s joins with no pause (FR-554)", ErrStale, spoken(voiced, id, line.Cue, line.Index),
			))
		case given.Entries[at].Sounds != line.Sounds:
			problems = append(problems, fmt.Errorf(
				"%w: %s was paused in speech sounds %q where its saved sounds are now %q (FR-554)",
				ErrStale, spoken(voiced, id, line.Cue, line.Index), given.Entries[at].Sounds, line.Sounds,
			))
		}
	}
	for _, entry := range given.Entries {
		if !joins[place{cue: entry.Cue, index: entry.Index}] {
			problems = append(problems, fmt.Errorf(
				"%w: %s has a pause but no longer joins (FR-554)", ErrStale, spoken(voiced, id, entry.Cue, entry.Index),
			))
		}
	}
	return problems
}

// spoken names a voice's line with its text where the script still has the line:
// bf_emma "Docked" line 1 "Docked, commander.".
func spoken(voiced script.Voiced, voice string, id cue.ID, index int) string {
	name := LineName(voice, id, index)
	if lines, ok := voiced.Lines(id); ok && index < len(lines) {
		return fmt.Sprintf("%s %q", name, lines[index].Text())
	}
	return name
}
