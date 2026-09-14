package library

import "sort"

// Group is one auditionable set of takes: everything the voice would draw on for the
// cues belonging to one area of the game.
//
// The grouping is the cue vocabulary's own. A cue id reads from the widest part of the
// game to the narrowest, so its first segment already names the area and no second
// taxonomy has to be invented or kept in step.
type Group struct {
	// Key is the cue id's first segment, for example "combat".
	Key string
	// Clips are the distinct files the group can play, sorted so the order does not
	// depend on map iteration.
	Clips []string
}

// Groups returns every auditionable group in the chosen voice, sorted by key.
//
// A group holds the union of its cues' takes, deduplicated, because one take can
// answer several cues. A group with nothing behind it is left out rather than offered
// as a button that plays silence.
func (c *Catalogue) Groups() []Group {
	gathered := make(map[string]map[string]struct{})
	for _, item := range c.table.All() {
		clips, ok := c.source.Lookup(item.ID())
		if !ok {
			continue
		}
		key := item.ID().Group()
		if gathered[key] == nil {
			gathered[key] = make(map[string]struct{})
		}
		for _, clip := range clips {
			gathered[key][clip] = struct{}{}
		}
	}

	out := make([]Group, 0, len(gathered))
	for key, set := range gathered {
		clips := make([]string, 0, len(set))
		for clip := range set {
			clips = append(clips, clip)
		}
		sort.Strings(clips)
		out = append(out, Group{Key: key, Clips: clips})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Key < out[b].Key })
	return out
}

// Audition returns one take drawn at random from a group; false when the voice has no
// such group. It is the audition pane's whole job: the caller plays what it gets back.
func (c *Catalogue) Audition(key string) (string, bool) {
	for _, group := range c.Groups() {
		if group.Key != key || len(group.Clips) == 0 {
			continue
		}
		return group.Clips[c.chooser.Intn(len(group.Clips))], true
	}
	return "", false
}
