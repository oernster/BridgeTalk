// Package scripttest builds voiced scripts for tests from what the sounds tool would have saved,
// so every suite that needs one builds it the same way.
package scripttest

import (
	"maps"
	"slices"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// Build makes a voiced script from saved speech sounds: each key's lines with their sounds in each
// accent, against a table holding a cue for each key. It refuses whatever cue.New, script.New or
// script.Voice refuses.
func Build(saved map[string]script.Saved) (script.Voiced, error) {
	return build(saved, func(built script.Script) (script.Script, error) { return built, nil })
}

// BuildJoining makes a voiced script as Build does from a script given a table of words and the
// words joined after a final comma (FR-549, FR-550). It also refuses whatever WithWords refuses.
func BuildJoining(saved map[string]script.Saved, spellings map[string][]string, joinAfterComma []string) (script.Voiced, error) {
	return build(saved, func(built script.Script) (script.Script, error) {
		return built.WithWords(spellings, joinAfterComma)
	})
}

// build makes a voiced script from saved speech sounds, handing the script to words before it is
// voiced.
func build(saved map[string]script.Saved, words func(script.Script) (script.Script, error)) (script.Voiced, error) {
	lines := make(map[string][]string, len(saved))
	cues := make([]cue.Cue, 0, len(saved))
	for _, id := range slices.Sorted(maps.Keys(saved)) {
		item, err := cue.New(cue.Definition{ID: id, Source: "journal", Event: id, Purpose: "When it happens."})
		if err != nil {
			return script.Voiced{}, err
		}
		cues = append(cues, item)
		lines[id] = saved[id].Lines
	}
	built, err := script.New(lines, cue.NewTable(cues))
	if err != nil {
		return script.Voiced{}, err
	}
	spoken, err := words(built)
	if err != nil {
		return script.Voiced{}, err
	}
	return script.Voice(spoken, saved)
}
