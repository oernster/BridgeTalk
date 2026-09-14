package script

import (
	"maps"
	"slices"
)

// Group is one group of the script's cues as the Audition pane offers it for a machine voice: the
// first segment its cues' ids share, with how many lines those cues hold (FR-546).
type Group struct {
	Key   string
	Lines int
}

// Groups returns every group of the script's cues, sorted by key, each counting the lines its cues
// hold. A group is every cue sharing the first segment of its id, as FR-216 groups a recorded voice's
// takes.
func (s Script) Groups() []Group {
	counted := make(map[string]int)
	for id, lines := range s.byCue {
		counted[id.Group()] += len(lines)
	}
	groups := make([]Group, 0, len(counted))
	for _, key := range slices.Sorted(maps.Keys(counted)) {
		groups = append(groups, Group{Key: key, Lines: counted[key]})
	}
	return groups
}
