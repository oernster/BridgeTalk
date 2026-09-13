package library

import (
	"sort"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/domain/selection"
)

// acknowledgement finds the cue played once when a voice is chosen, so the choice is
// confirmed by hearing it rather than by text changing on screen.
//
// It is found by its source rather than by its id, because the id is a word in the cue
// table and the table is the one place such words live; writing it here would be a
// second home for it that has to be kept in step. The game emits nothing with an
// application source, so the only cues carrying one are the application's own moments.
//
// Exactly one such cue exists today. A second would need something to tell them apart,
// and this is where that would go.
func acknowledgement(table cue.Table) (cue.ID, bool) {
	for _, item := range table.All() {
		if item.Source() == event.SourceApplication {
			return item.ID(), true
		}
	}
	return "", false
}

// Catalogue answers, for a cue, what the chosen voice can play for it.
//
// There is one resolution rule and no chain: the voice either recorded that cue or it
// did not. Nothing is substituted from another cue, because a wrong line delivered
// confidently is worse than silence. A voice is expected to be complete anyway;
// a gap is a thing to go and record, not a thing to paper over.
type Catalogue struct {
	voice   Voice
	table   cue.Table
	chooser selection.Chooser
}

// NewCatalogue builds a catalogue over one voice.
func NewCatalogue(voice Voice, table cue.Table, chooser selection.Chooser) *Catalogue {
	return &Catalogue{voice: voice, table: table, chooser: chooser}
}

// ActiveVoice returns the display name of the voice in use.
func (c *Catalogue) ActiveVoice() string { return c.voice.Name }

// Clips returns the takes recorded for a cue; false when this voice has none.
func (c *Catalogue) Clips(id cue.ID) (ports.Performance, bool) {
	clips, ok := c.voice.Lookup(id)
	if !ok {
		return ports.Performance{}, false
	}
	return ports.Performance{Clips: clips}, true
}

// Acknowledgement returns one take of the acknowledgement cue, chosen at random.
//
// Nothing here is worth an error. A voice with no acknowledgement recorded is silent
// when it is chosen rather than broken.
func (c *Catalogue) Acknowledgement() (string, bool) {
	id, named := acknowledgement(c.table)
	if !named {
		return "", false
	}
	clips, ok := c.voice.Lookup(id)
	if !ok {
		return "", false
	}
	return clips[c.chooser.Intn(len(clips))], true
}

// Coverage counts the cues this voice can serve, out of the whole table.
func (c *Catalogue) Coverage() (int, int) {
	served := 0
	for _, item := range c.table.All() {
		if _, ok := c.voice.Lookup(item.ID()); ok {
			served++
		}
	}
	return served, c.table.Len()
}

// Files reports the takes this voice holds and how many of them are reachable.
//
// The two numbers are expected to be equal, since a voice is recorded against the
// vocabulary and every file it holds should answer a cue. They are reported as a pair
// because they fail differently: a shortfall in Coverage means lines were never
// recorded, while a gap here means files are present that nothing can reach, which is
// a naming mistake rather than a missing performance.
func (c *Catalogue) Files() (reachable int, held int) {
	seen := map[string]struct{}{}
	for _, clips := range c.voice.byCue {
		for _, clip := range clips {
			seen[clip] = struct{}{}
		}
	}
	return len(seen), c.voice.Takes
}

// Served lists the cues this voice can serve, sorted by id.
//
// It is the exact complement of Unbound, which is why both ask the voice rather than
// one deriving itself from the other: a cue is in one list or the other, never in
// neither and never in both; one predicate is what keeps that true.
func (c *Catalogue) Served() []cue.Cue { return c.partition(true) }

// Unbound lists the cues this voice has nothing for, sorted by id, which is what the
// chooser shows so a quiet voice explains itself.
func (c *Catalogue) Unbound() []cue.Cue { return c.partition(false) }

func (c *Catalogue) partition(served bool) []cue.Cue {
	var out []cue.Cue
	for _, item := range c.table.All() {
		if _, ok := c.voice.Lookup(item.ID()); ok == served {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID() < out[b].ID() })
	return out
}
