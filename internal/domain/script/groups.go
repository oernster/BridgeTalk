package script

import (
	"maps"
	"slices"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// Group is one group of the script's cues as the Audition pane offers it for a machine voice: the
// first segment its cues' ids share, with how many lines its cues switched on hold (FR-546, FR-746).
// SwitchedOff says every cue in it is switched off, so it has lines yet none can be heard (FR-747).
type Group struct {
	Key         string
	Lines       int
	SwitchedOff bool
}

// Groups returns every group of the script's cues, sorted by key, each counting the lines of its cues
// heard. A group is every cue sharing the first segment of its id, as FR-216 groups a recorded voice's
// takes. A group none of whose cues is heard is still returned, counting nothing and marked switched
// off, so a caller can tell it from a group the script does not hold.
func (s Script) Groups(heard cue.Heard) []Group {
	counted := make(map[string]int)
	held := make(map[string]bool)
	for id, lines := range s.byCue {
		held[id.Group()] = true
		if heard(id) {
			counted[id.Group()] += len(lines)
		}
	}
	groups := make([]Group, 0, len(held))
	for _, key := range slices.Sorted(maps.Keys(held)) {
		groups = append(groups, Group{Key: key, Lines: counted[key], SwitchedOff: counted[key] == 0})
	}
	return groups
}
