package library

import (
	"sort"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/selection"
	"github.com/oernster/bridge-talk/internal/domain/take"
)

// Catalogue answers, for a cue, what the chosen voice can play for it.
//
// There is one resolution rule and no chain: the voice either has that cue or it does
// not. Nothing is substituted from another cue, because a wrong line delivered
// confidently is worse than silence. A voice is expected to be complete anyway;
// a gap is a thing to go and record, not a thing to paper over.
//
// The takes arrive through the audio source port (FR-501), so the catalogue knows
// nothing of where they came from: a scanned voice is one source among any.
type Catalogue struct {
	source  ports.AudioSource
	shown   string
	table   cue.Table
	chooser selection.Chooser
}

// NewCatalogue builds a catalogue over one audio source, shown by the name given.
func NewCatalogue(source ports.AudioSource, shown string, table cue.Table, chooser selection.Chooser) *Catalogue {
	return &Catalogue{source: source, shown: shown, table: table, chooser: chooser}
}

// ActiveVoice returns the name the voice in use is shown by (FR-210).
func (c *Catalogue) ActiveVoice() string { return c.shown }

// Clips returns the takes the source holds for a cue; false when it has none.
func (c *Catalogue) Clips(id cue.ID) (ports.Performance, bool) {
	takes, ok := c.source.Lookup(id)
	if !ok {
		return ports.Performance{}, false
	}
	return ports.Performance{Takes: takes}, true
}

// Acknowledgement returns one take of the acknowledgement cue, chosen at random.
//
// Nothing here is worth an error. A voice with no acknowledgement recorded is silent
// when it is chosen rather than broken.
func (c *Catalogue) Acknowledgement() (take.Take, bool) {
	id, named := c.table.Confirmation()
	if !named {
		return nil, false
	}
	takes, ok := c.source.Lookup(id)
	if !ok {
		return nil, false
	}
	return takes[c.chooser.Intn(len(takes))], true
}

// Coverage counts the cues this voice can serve, out of the whole table.
func (c *Catalogue) Coverage() (int, int) {
	served := 0
	for _, item := range c.table.All() {
		if _, ok := c.source.Lookup(item.ID()); ok {
			served++
		}
	}
	return served, c.table.Len()
}

// Served lists the cues this voice can serve, sorted by id.
//
// It is the exact complement of Unbound, which is why both ask the source rather than
// one deriving itself from the other: a cue is in one list or the other, never in
// neither and never in both; one predicate is what keeps that true.
func (c *Catalogue) Served() []cue.Cue { return c.partition(true) }

// Unbound lists the cues this voice has nothing for, sorted by id, which is what the
// chooser shows so a quiet voice explains itself.
func (c *Catalogue) Unbound() []cue.Cue { return c.partition(false) }

func (c *Catalogue) partition(served bool) []cue.Cue {
	var out []cue.Cue
	for _, item := range c.table.All() {
		if _, ok := c.source.Lookup(item.ID()); ok == served {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID() < out[b].ID() })
	return out
}
