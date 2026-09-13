package cue

import (
	"sort"

	"github.com/oernster/bridge-talk/internal/domain/event"
)

// Table is the whole cue vocabulary, indexed so that resolving an event does not
// scan every cue.
type Table struct {
	byKey map[key][]Cue
	all   []Cue
}

// key is the lookup pair every event carries.
type key struct {
	source event.Source
	name   string
}

// NewTable indexes cues for lookup. Cues sharing a key are held most-specific
// first, so Resolve can return the first match it finds.
func NewTable(cues []Cue) Table {
	byKey := make(map[key][]Cue, len(cues))
	all := make([]Cue, len(cues))
	copy(all, cues)
	for _, item := range all {
		index := key{source: item.source, name: item.name}
		byKey[index] = append(byKey[index], item)
	}
	for index := range byKey {
		bucket := byKey[index]
		sort.SliceStable(bucket, func(a, b int) bool {
			return bucket[a].Specificity() > bucket[b].Specificity()
		})
		byKey[index] = bucket
	}
	return Table{byKey: byKey, all: all}
}

// Resolve returns the most specific cue an event satisfies.
//
// A cue constraining more payload fields describes a narrower situation, so a
// very-valuable-salvage drop wins over a plain supercruise exit even though both
// listen to the same journal event.
func (t Table) Resolve(candidate event.Event) (Cue, bool) {
	bucket, ok := t.byKey[key{source: candidate.Source(), name: candidate.Name()}]
	if !ok {
		return Cue{}, false
	}
	for _, item := range bucket {
		if item.Matches(candidate) {
			return item, true
		}
	}
	return Cue{}, false
}

// All returns every cue in the table, in the order it was defined.
func (t Table) All() []Cue {
	out := make([]Cue, len(t.all))
	copy(out, t.all)
	return out
}

// Len reports how many cues the table holds.
func (t Table) Len() int { return len(t.all) }

// Names returns the distinct source and name pairs the table listens for, which is
// what an event source uses to avoid parsing payloads nothing will ever match.
func (t Table) Names(source event.Source) []string {
	// No de-duplication is needed. The index is keyed on the source and name pair, so
	// every key is distinct; filtering to one source then leaves names that are
	// pairwise distinct by construction.
	var out []string
	for index := range t.byKey {
		if index.source != source {
			continue
		}
		out = append(out, index.name)
	}
	sort.Strings(out)
	return out
}
