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
	return script.Voice(built, saved)
}
