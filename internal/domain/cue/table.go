package cue

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/event"
)

// Table is the whole cue vocabulary, indexed so that resolving an event does not
// scan every cue.
type Table struct {
	byKey      map[key][]Cue
	all        []Cue
	categories []string
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
			return bucket[a].narrower(bucket[b])
		})
		byKey[index] = bucket
	}
	return Table{byKey: byKey, all: all}
}

// NewCategorisedTable indexes cues as NewTable does and holds the categories Chatter lists them under,
// in the order given (FR-634, FR-635).
//
// It refuses a category left blank or named twice, a cue the game raises whose category is missing or
// not listed, a cue from the application that names one and a listed category no cue sits in. Each
// refusal names what is wrong, so a table written by hand says where to look.
func NewCategorisedTable(categories []string, cues []Cue) (Table, error) {
	holding := make(map[string]int, len(categories))
	for _, name := range categories {
		if strings.TrimSpace(name) == "" {
			return Table{}, fmt.Errorf("%w: a category has no name", ErrInvalidCue)
		}
		if _, twice := holding[name]; twice {
			return Table{}, fmt.Errorf("%w: the category %q is named twice", ErrInvalidCue, name)
		}
		holding[name] = 0
	}
	for _, item := range cues {
		if err := categoryOf(item, holding); err != nil {
			return Table{}, err
		}
	}
	for _, name := range categories {
		if holding[name] == 0 {
			return Table{}, fmt.Errorf("%w: the category %q holds no cue", ErrInvalidCue, name)
		}
	}
	table := NewTable(cues)
	table.categories = append([]string(nil), categories...)
	return table, nil
}

// categoryOf checks one cue's category against those listed, counting the cue into it.
func categoryOf(item Cue, holding map[string]int) error {
	if !item.Switchable() {
		if item.category != "" {
			return fmt.Errorf(
				"%w: %s comes from the application, so it sits in no category", ErrInvalidCue, item.id,
			)
		}
		return nil
	}
	if item.category == "" {
		return fmt.Errorf("%w: %s has no category, the set Chatter lists it under", ErrInvalidCue, item.id)
	}
	if _, listed := holding[item.category]; !listed {
		return fmt.Errorf(
			"%w: %s names the category %q, which the table does not list", ErrInvalidCue, item.id, item.category,
		)
	}
	holding[item.category]++
	return nil
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

// Categories returns the categories Chatter lists the cues under, in the table's order; none for a
// table built without them.
func (t Table) Categories() []string {
	return append([]string(nil), t.categories...)
}

// Confirmation returns the cue played when a voice is cast (FR-232, FR-521), so the choice is
// confirmed by hearing it rather than by text changing on screen.
//
// It is found by its source rather than by its id, because the id is a word in the cue table and the
// table is the one place such words live. The game emits nothing with an application source, so the
// only cues carrying one are the application's own moments. Exactly one such cue exists today; a
// second would need something to tell them apart; this is where that would go.
func (t Table) Confirmation() (ID, bool) {
	for _, item := range t.all {
		if item.source == event.SourceApplication {
			return item.id, true
		}
	}
	return "", false
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
