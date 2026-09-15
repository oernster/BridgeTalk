// Package selection holds the arithmetic that decides whether a cue speaks now and
// which take it uses. It is pure: the clock and the randomness are both injected,
// so every decision here is reproducible in a test.
package selection

import (
	"time"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// Chooser supplies the randomness a take selection needs. The domain defines it so
// that nothing here has to import math/rand.
type Chooser interface {
	// Intn returns a value in [0, n). Callers guarantee n is positive.
	Intn(n int) int
}

// Picker chooses a take, avoiding the clip last played for the same cue (FR-610).
//
// Avoiding only the immediately previous clip is deliberate. Remembering more would
// make a two-clip cue silent on alternate firings; a long memory makes a large
// folder feel ordered rather than varied.
type Picker struct {
	chooser Chooser
	last    map[cue.ID]string
}

// NewPicker builds a Picker over an injected source of randomness.
func NewPicker(chooser Chooser) *Picker {
	return &Picker{chooser: chooser, last: make(map[cue.ID]string)}
}

// Pick returns one clip for a cue; false when there is nothing to play. It records nothing;
// Played does, for the reason Open records nothing: a take picked for a firing that is then
// let go was never heard, so it is not the take to avoid next time.
func (p *Picker) Pick(id cue.ID, clips []string) (string, bool) {
	switch len(clips) {
	case 0:
		return "", false
	case 1:
		return clips[0], true
	}

	previous, seen := p.last[id]
	if !seen {
		return clips[p.chooser.Intn(len(clips))], true
	}

	// Choose from the clips that are not the previous one by picking an index into
	// the shortened list, then stepping over the excluded entry. This draws evenly
	// across the remaining clips with a single call to the chooser.
	candidates := make([]string, 0, len(clips)-1)
	for _, clip := range clips {
		if clip != previous {
			candidates = append(candidates, clip)
		}
	}
	if len(candidates) == 0 {
		return clips[0], true
	}
	return candidates[p.chooser.Intn(len(candidates))], true
}

// Played notes that a clip was handed over to be spoken for a cue, making it the clip the
// next Pick for that cue avoids.
func (p *Picker) Played(id cue.ID, clip string) {
	p.last[id] = clip
}

// CooldownGate rate-limits a cue to at most one firing per its cooldown.
type CooldownGate struct {
	lastFired map[cue.ID]time.Time
}

// NewCooldownGate builds an empty gate.
func NewCooldownGate() *CooldownGate {
	return &CooldownGate{lastFired: make(map[cue.ID]time.Time)}
}

// Open reports whether a cue may fire at now. It records nothing: a firing counts against
// the cooldown only once Record is told it was handed over to be spoken, so a firing nobody
// heard holds nothing back. A cue with no cooldown is always open.
func (g *CooldownGate) Open(item cue.Cue, now time.Time) bool {
	previous, seen := g.lastFired[item.ID()]
	return !seen || item.Cooldown() <= 0 || now.Sub(previous) >= item.Cooldown()
}

// Record notes that a cue fired at now. A cue with no cooldown is recorded too, so that
// adding a cooldown later needs no special first-run case.
func (g *CooldownGate) Record(id cue.ID, now time.Time) {
	g.lastFired[id] = now
}

// Reset clears every recorded firing.
func (g *CooldownGate) Reset() {
	g.lastFired = make(map[cue.ID]time.Time)
}

// DedupeWindow collapses repeats of the same cue arriving within a short window.
//
// This is not the cooldown. The cooldown is a deliberate per-cue rate limit
// measured in seconds; the dedupe window is much shorter and exists because the
// journal sometimes emits the same situation more than once in quick succession.
type DedupeWindow struct {
	window time.Duration
	seen   map[cue.ID]time.Time
}

// NewDedupeWindow builds a window of the given width.
func NewDedupeWindow(window time.Duration) *DedupeWindow {
	return &DedupeWindow{window: window, seen: make(map[cue.ID]time.Time)}
}

// Fresh reports whether a cue is not a repeat of one marked inside the window. It records
// nothing; Mark does, for the reason Open records nothing.
func (d *DedupeWindow) Fresh(id cue.ID, now time.Time) bool {
	previous, ok := d.seen[id]
	return !ok || now.Sub(previous) >= d.window
}

// Mark notes that a cue was handed over to be spoken at now, opening the window against
// its repeats.
func (d *DedupeWindow) Mark(id cue.ID, now time.Time) {
	d.seen[id] = now
}
