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

// madeOnCallLimit is how long a cue that fired with no line made waits for that line before it is
// let go (FR-514). The measured worst case is about 1.25 s: a line under way, the cue's own line and
// loading the model (Oliver, 2026-09-14).
const madeOnCallLimit = 2 * time.Second

// waiting is a cue that fired with no line made, waiting for its line to be written.
type waiting struct {
	matched   cue.Cue
	candidate event.Event
	fired     time.Time
}

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

	// maker makes a cue's lines when it fires with none; nil for a voice whose lines are all on disk.
	maker ports.CueMaker
	// waiting holds the cues waiting for their lines, in the order they fired.
	waiting []waiting
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

// SetCueMaker hands over what makes a cue's lines when it fires with none made (FR-514). A machine
// voice is given one; a recorded voice has every line it will ever have on disk, so it is not.
func (r *ReactionService) SetCueMaker(maker ports.CueMaker) { r.maker = maker }

// Tick hands over each waiting cue whose line has been written, as though it fired now; lets go of
// each that has waited longer than madeOnCallLimit; then lets the scheduler start whatever should now
// be speaking (FR-514). The poll loop calls it on every tick.
func (r *ReactionService) Tick() {
	now := r.clock.Now()
	var still []waiting
	for _, each := range r.waiting {
		performance, served := r.catalogue.Clips(each.matched.ID())
		switch {
		case served && len(performance.Clips) > 0:
			r.speak(each.matched, each.candidate, performance.Clips, now)
		case now.Sub(each.fired) > madeOnCallLimit:
			r.report(each.matched, each.candidate, "", ports.OutcomeDropped)
		default:
			still = append(still, each)
		}
	}
	r.waiting = still
	r.scheduler.Advance()
}

// Muted reports whether playback is currently silenced.
func (r *ReactionService) Muted() bool { return r.muted }

// Handle processes one event to completion.
func (r *ReactionService) Handle(candidate event.Event) {
	matched, ok := r.table.Resolve(candidate)
	if !ok {
		return
	}
	if r.isWaiting(matched.ID()) {
		r.report(matched, candidate, "", ports.OutcomeDuplicate)
		return
	}

	now := r.clock.Now()
	if !r.dedupe.Fresh(matched.ID(), now) {
		r.report(matched, candidate, "", ports.OutcomeDuplicate)
		return
	}
	if !r.cooldown.Open(matched, now) {
		r.report(matched, candidate, "", ports.OutcomeCooldown)
		return
	}

	performance, served := r.catalogue.Clips(matched.ID())
	if !served || len(performance.Clips) == 0 {
		r.makeOnCall(matched, candidate, now)
		return
	}
	r.speak(matched, candidate, performance.Clips, now)
}

// makeOnCall answers a cue with no take: where a line is on its way for it, it waits for that line,
// or is dropped while muted with the line still made (FR-514, FR-611); otherwise it is unbound.
func (r *ReactionService) makeOnCall(matched cue.Cue, candidate event.Event, now time.Time) {
	if r.maker == nil || !r.maker.MakeNext(matched.ID()) {
		r.report(matched, candidate, "", ports.OutcomeUnbound)
		return
	}
	if r.muted {
		r.report(matched, candidate, "", ports.OutcomeDropped)
		return
	}
	r.waiting = append(r.waiting, waiting{matched: matched, candidate: candidate, fired: now})
	r.report(matched, candidate, "", ports.OutcomeMaking)
}

// speak hands a cue with takes to the scheduler, as though it fired at now.
func (r *ReactionService) speak(matched cue.Cue, candidate event.Event, clips []string, now time.Time) {
	if r.muted {
		r.report(matched, candidate, "", ports.OutcomeDropped)
		return
	}

	// The picker declines only an empty list; an empty list has already been answered
	// before a cue reaches here, so there is nothing left here to decline.
	chosen, _ := r.picker.Pick(matched.ID(), clips)

	// A firing counts against the repeat window and the cooldown only once the scheduler
	// takes it. One that was muted, had no take or was let go was never heard, so it holds
	// back nothing that follows it (Oliver, 2026-09-13).
	if r.scheduler.Submit(Request{Cue: matched, Clips: []string{chosen}}) {
		r.dedupe.Mark(matched.ID(), now)
		r.cooldown.Record(matched.ID(), now)
	}
}

// isWaiting reports whether a cue is waiting for its line.
func (r *ReactionService) isWaiting(id cue.ID) bool {
	for _, each := range r.waiting {
		if each.matched.ID() == id {
			return true
		}
	}
	return false
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
		Clip:    clip,
		Outcome: outcome,
	})
}
