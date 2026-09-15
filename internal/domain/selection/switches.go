package selection

import (
	"sort"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// Switches is which cues are switched off (FR-622); every cue it does not name is switched on
// (FR-628), so the zero value has everything on.
//
// It is a value: a change answers a new set and leaves the one it was made from as it was. The
// engine reads a set on the poll loop's goroutine while the window changes it from its own, so a
// set is swapped in whole rather than edited where it is being read.
type Switches struct {
	off map[cue.ID]struct{}
}

// NewSwitches builds the set with each id named switched off.
func NewSwitches(off []cue.ID) Switches {
	held := make(map[cue.ID]struct{}, len(off))
	for _, id := range off {
		held[id] = struct{}{}
	}
	return Switches{off: held}
}

// Off reports whether a cue is switched off.
func (s Switches) Off(id cue.ID) bool {
	_, off := s.off[id]
	return off
}

// With answers a set in which every id named is switched to the state given; every other id is as
// it was in this one.
func (s Switches) With(on bool, ids ...cue.ID) Switches {
	held := make(map[cue.ID]struct{}, len(s.off)+len(ids))
	for id := range s.off {
		held[id] = struct{}{}
	}
	for _, id := range ids {
		if on {
			delete(held, id)
			continue
		}
		held[id] = struct{}{}
	}
	return Switches{off: held}
}

// Kept answers the ids to keep for the next run: those switched off that the table holds as cues
// with a switch, in id order so what they are written to changes only when a switch does (FR-629,
// FR-631).
func (s Switches) Kept(table cue.Table) []cue.ID {
	var kept []cue.ID
	for _, item := range table.All() {
		if item.Switchable() && s.Off(item.ID()) {
			kept = append(kept, item.ID())
		}
	}
	sort.Slice(kept, func(a, b int) bool { return kept[a] < kept[b] })
	return kept
}
