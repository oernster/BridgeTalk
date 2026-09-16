package library

import (
	"sort"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/take"
)

// Group is one auditionable set of takes: everything the voice would draw on for the
// cues belonging to one area of the game.
//
// The grouping is the cue vocabulary's own. A cue id reads from the widest part of the
// game to the narrowest, so its first segment already names the area and no second
// taxonomy has to be invented or kept in step.
type Group struct {
	// Key is the cue id's first segment, for example "combat".
	Key string
	// Takes are the distinct takes the group can play from its cues heard, sorted by their
	// keys so the order does not depend on map iteration.
	Takes []take.Take
	// SwitchedOff says the voice has takes for the group yet every one of its cues is
	// switched off, so none can be heard (FR-747).
	SwitchedOff bool
}

// Groups returns every auditionable group in the chosen voice, sorted by key, each
// holding the takes of its cues heard (FR-745, FR-746).
//
// A group holds the union of its cues' takes, deduplicated, because one take can
// answer several cues. A group with nothing behind it is left out rather than offered
// as a button that plays silence. A group with takes behind it none of which is heard
// is kept, holding nothing and marked switched off, so a caller can tell the two apart.
func (c *Catalogue) Groups(heard cue.Heard) []Group {
	// A take is gathered under its key, which is how one take answering several cues is
	// counted once (take.Take.Key).
	gathered := make(map[string]map[string]take.Take)
	for _, item := range c.table.All() {
		takes, ok := c.source.Lookup(item.ID())
		if !ok || len(takes) == 0 {
			continue
		}
		key := item.ID().Group()
		if gathered[key] == nil {
			gathered[key] = make(map[string]take.Take)
		}
		if !heard(item.ID()) {
			continue
		}
		for _, each := range takes {
			gathered[key][each.Key()] = each
		}
	}

	out := make([]Group, 0, len(gathered))
	for key, set := range gathered {
		takes := make([]take.Take, 0, len(set))
		for _, each := range set {
			takes = append(takes, each)
		}
		sort.Slice(takes, func(a, b int) bool { return takes[a].Key() < takes[b].Key() })
		out = append(out, Group{Key: key, Takes: takes, SwitchedOff: len(takes) == 0})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Key < out[b].Key })
	return out
}

// Audition returns one take drawn at random from a group's cues heard; false when the
// voice has no such group or none of its takes is heard (FR-745). It is the audition
// pane's whole job: the caller plays what it gets back.
func (c *Catalogue) Audition(key string, heard cue.Heard) (take.Take, bool) {
	for _, group := range c.Groups(heard) {
		if group.Key != key || len(group.Takes) == 0 {
			continue
		}
		return group.Takes[c.chooser.Intn(len(group.Takes))], true
	}
	return nil, false
}
