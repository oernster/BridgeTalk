package services

import (
	"fmt"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/domain/selection"
)

// dedupeWindow is the width of the repeat collapse. It is much shorter than any
// cue cooldown because it exists only to absorb the journal restating a situation,
// not to rate-limit anything.
const dedupeWindow = 900 * time.Millisecond

// ReactionService turns events into playback requests.
//
// It owns the whole decision for one event: find the cue, check it is not a repeat,
// check its cooldown, ask the catalogue what the active voice can play, choose a
// take and hand the result to the scheduler. Every step that ends in silence is
// reported, so nothing fails invisibly.
type ReactionService struct {
	table     cue.Table
	catalogue ports.VoiceCatalogue
	scheduler *Scheduler
	picker    *selection.Picker
	cooldown  *selection.CooldownGate
	dedupe    *selection.DedupeWindow
	reporter  ports.Reporter
	clock     ports.Clock
	muted     bool
}

// NewReactionService wires the decision path.
func NewReactionService(
	table cue.Table,
	catalogue ports.VoiceCatalogue,
	scheduler *Scheduler,
	chooser selection.Chooser,
	reporter ports.Reporter,
	clock ports.Clock,
) *ReactionService {
	return &ReactionService{
		table:     table,
		catalogue: catalogue,
		scheduler: scheduler,
		picker:    selection.NewPicker(chooser),
		cooldown:  selection.NewCooldownGate(),
		dedupe:    selection.NewDedupeWindow(dedupeWindow),
		reporter:  reporter,
		clock:     clock,
	}
}

// SetMuted silences playback without stopping the watchers, so the reaction log
// keeps showing what would have been said.
func (r *ReactionService) SetMuted(muted bool) { r.muted = muted }

// Muted reports whether playback is currently silenced.
func (r *ReactionService) Muted() bool { return r.muted }

// Handle processes one event to completion.
func (r *ReactionService) Handle(candidate event.Event) {
	matched, ok := r.table.Resolve(candidate)
	if !ok {
		return
	}

	now := r.clock.Now()
	if !r.dedupe.Fresh(matched.ID(), now) {
		r.report(matched, candidate, "", ports.OutcomeDuplicate)
		return
	}
	if !r.cooldown.Allow(matched, now) {
		r.report(matched, candidate, "", ports.OutcomeCooldown)
		return
	}

	performance, served := r.catalogue.Clips(matched.ID())
	if !served || len(performance.Clips) == 0 {
		r.report(matched, candidate, "", ports.OutcomeUnbound)
		return
	}

	if r.muted {
		r.report(matched, candidate, "", ports.OutcomeDropped)
		return
	}

	// The picker declines only an empty list; an empty list has already been answered
	// above, so there is nothing left here to decline.
	chosen, _ := r.picker.Pick(matched.ID(), performance.Clips)
	r.scheduler.Submit(Request{Cue: matched, Clips: []string{chosen}})
}

// HandleAll processes a batch in arrival order, then lets the scheduler start
// whatever should now be speaking.
func (r *ReactionService) HandleAll(events []event.Event) {
	for _, candidate := range events {
		r.Handle(candidate)
	}
	r.scheduler.Advance()
}

// Describe renders a one-line summary of the active voice's coverage.
func (r *ReactionService) Describe() string {
	served, total := r.catalogue.Coverage()
	return fmt.Sprintf("%s serves %d of %d cues", r.catalogue.ActiveVoice(), served, total)
}

// report records a decision that produced no playback.
func (r *ReactionService) report(matched cue.Cue, candidate event.Event, clip, outcome string) {
	if r.reporter == nil {
		return
	}
	r.reporter.Report(ports.Reaction{
		At:      candidate.At(),
		Cue:     matched.ID(),
		Event:   candidate.Name(),
		Clip:    clip,
		Outcome: outcome,
	})
}
