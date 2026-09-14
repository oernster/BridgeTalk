package making

import (
	"slices"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
)

// Order is the order a machine voice's cues are made in (FR-511): the application's own cues, the one
// played on a cast among them, then the rest from the highest priority down. Cues that rank alike keep
// the order the table lists them in.
func Order(table cue.Table) []cue.ID {
	ranked := slices.Clone(table.All())
	slices.SortStableFunc(ranked, func(a, b cue.Cue) int { return rank(b) - rank(a) })
	ids := make([]cue.ID, 0, len(ranked))
	for _, item := range ranked {
		ids = append(ids, item.ID())
	}
	return ids
}

// rank places a cue in the making order: each priority by its own value, with the application's own
// cues above the highest. They are found by their source, as the catalogue finds the confirmation, so
// no cue id is written here.
func rank(item cue.Cue) int {
	if item.Source() == event.SourceApplication {
		return int(cue.PriorityAlert) + 1
	}
	return int(item.Priority())
}

// sequence gives the cues in the order given, then every cue of the script the order leaves out in the
// script's own order. A cue the order names that the script has no lines for gives New no lines, so it
// needs no passing over here.
func sequence(scripted, order []cue.ID) []cue.ID {
	out := make([]cue.ID, 0, len(scripted))
	for _, id := range order {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	for _, id := range scripted {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}
