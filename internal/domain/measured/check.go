package measured

import (
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// Check returns every way the book is stale against the shipped script, the voices given and the
// digests the list gives the model file and each voice's style file, each naming what is stale
// (FR-554, FR-557). expected lists, for an accent, the lines each voice of it should have an entry
// for. The model comes first, then each voice in the order given: its style file, its expected lines
// in the script's order, then its entries for lines no longer expected.
func Check[E Entry](
	kind Kind, book Book[E], voices []machinevoice.Voice, voiced script.Voiced,
	expected func(machinevoice.Accent) []script.SavedLine, model string, styles map[string]string,
) []error {
	var problems []error
	if book.model != model {
		problems = append(problems, fmt.Errorf(
			"%w: the %s were found with the model digest %q where the list gives %q (%s)",
			kind.Stale, kind.Plural, book.model, model, kind.Checked,
		))
	}
	for _, voice := range voices {
		problems = append(problems, checkVoice(kind, book, voice, voiced, expected(voice.Accent()), styles[voice.ID()])...)
	}
	return problems
}

// checkVoice returns every way one voice's entries are stale against the lines expected in its accent
// and the digest the list gives its style file.
func checkVoice[E Entry](kind Kind, book Book[E], voice machinevoice.Voice, voiced script.Voiced, lines []script.SavedLine, style string) []error {
	id := voice.ID()
	given, ok := book.voices[id]
	if !ok {
		return []error{fmt.Errorf("%w: %s has no %s (%s)", kind.Stale, id, kind.Plural, kind.Checked)}
	}
	var problems []error
	if given.Style != style {
		problems = append(problems, fmt.Errorf(
			"%w: %s's %s were found with the style digest %q where the list gives %q (%s)",
			kind.Stale, id, kind.Plural, given.Style, style, kind.Checked,
		))
	}
	wanted := make(map[place]bool)
	for _, line := range lines {
		key := place{cue: line.Cue, index: line.Index}
		wanted[key] = true
		at, entered := book.places[id][key]
		switch {
		case !entered:
			problems = append(problems, fmt.Errorf(
				"%w: %s %s (%s)", kind.Stale, spoken(voiced, id, line.Cue, line.Index), kind.Missing, kind.Checked,
			))
		case given.Entries[at].Measured().Sounds != line.Sounds:
			problems = append(problems, fmt.Errorf(
				"%w: %s %s %q where its saved sounds are now %q (%s)",
				kind.Stale, spoken(voiced, id, line.Cue, line.Index), kind.MeasuredIn, given.Entries[at].Measured().Sounds, line.Sounds, kind.Checked,
			))
		}
	}
	for _, entry := range given.Entries {
		line := entry.Measured()
		if !wanted[place{cue: line.Cue, index: line.Index}] {
			problems = append(problems, fmt.Errorf(
				"%w: %s %s (%s)", kind.Stale, spoken(voiced, id, line.Cue, line.Index), kind.Gone, kind.Checked,
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
