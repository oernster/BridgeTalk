package services

import (
	"sort"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/take"
)

// takeGap is the silence inserted between the parts of one take. It is zero: a take recorded
// in pieces is one utterance, joined as naturally as the recordings allow (FR-573).
const takeGap = 0

// Request is one thing waiting to be said: one cue answered by one take (FR-221). The
// alternatives were chosen between before it got here, so a request carries the take that
// was chosen rather than the takes it was chosen from.
type Request struct {
	Cue  cue.Cue
	Take take.Take
}

// Scheduler decides what is spoken, in what order and what is interrupted or
// discarded. It holds no goroutine of its own: Submit enqueues and Advance starts
// whatever should be playing, so the caller owns the loop and the tests own time.
type Scheduler struct {
	player   ports.AudioPlayer
	reporter ports.Reporter
	clock    ports.Clock
	// switches says which moments are switched off on Chatter; nil has every moment on (FR-625).
	switches ports.Switchboard

	queue   []Request
	current *Request
}

// NewScheduler builds a scheduler over an injected player.
func NewScheduler(player ports.AudioPlayer, reporter ports.Reporter, clock ports.Clock) *Scheduler {
	return &Scheduler{player: player, reporter: reporter, clock: clock}
}

// SetSwitchboard hands over what says which moments are switched off on Chatter, so a request
// switched off while it waits is let go when its turn comes (FR-625).
func (s *Scheduler) SetSwitchboard(switches ports.Switchboard) { s.switches = switches }

// Submit applies the priority policy to a new request, answering whether it was taken:
// queued or started rather than let go.
//
// The policy is what separates a voice worth listening to from a slot machine:
//   - an alert interrupts whatever less urgent is speaking and waits ahead of everything
//     less urgent, behind any alert that arrived before it;
//   - a notice queues;
//   - an ambient queues only when nothing is already waiting;
//   - a flavour is discarded whenever anything at all is pending.
func (s *Scheduler) Submit(request Request) bool {
	switch request.Cue.Priority() {
	case cue.PriorityAlert:
		if s.current != nil && s.current.Cue.Priority() < cue.PriorityAlert {
			s.player.Stop()
			s.current = nil
		}
		fallthrough
	case cue.PriorityNotice:
		s.queue = append(s.queue, request)
	case cue.PriorityAmbient:
		if len(s.queue) > 0 {
			s.report(request, ports.OutcomeDropped)
			return false
		}
		s.queue = append(s.queue, request)
	default:
		if len(s.queue) > 0 || s.current != nil {
			s.report(request, ports.OutcomeDropped)
			return false
		}
		s.queue = append(s.queue, request)
	}
	s.sortQueue()
	if s.current != nil {
		s.report(request, ports.OutcomeQueued)
	}
	return true
}

// sortQueue keeps the queue in descending priority while preserving the arrival
// order of equal priorities, so nothing waiting is ever starved by its neighbours.
func (s *Scheduler) sortQueue() {
	sort.SliceStable(s.queue, func(a, b int) bool {
		return s.queue[a].Cue.Priority() > s.queue[b].Cue.Priority()
	})
}

// Advance starts the next request when nothing is playing. It is called after a
// submission and again whenever the player reports it has finished. A request whose moment
// was switched off while it waited is let go and recorded rather than started; the one
// behind it is taken instead (FR-625). A take already playing is never stopped by its switch
// (FR-626).
func (s *Scheduler) Advance() {
	if s.player.Playing() {
		return
	}
	for len(s.queue) > 0 {
		next := s.queue[0]
		s.queue = s.queue[1:]
		if s.switches != nil && s.switches.Off(next.Cue.ID()) {
			s.report(next, ports.OutcomeOff)
			continue
		}
		s.start(next)
		return
	}
}

// start plays one request.
func (s *Scheduler) start(next Request) {
	if err := s.player.Play(next.Take, time.Duration(takeGap)); err != nil {
		s.current = nil
		s.report(next, ports.OutcomeUnbound)
		return
	}
	held := next
	s.current = &held
	s.report(next, ports.OutcomePlayed)
}

// Finished tells the scheduler the player has reported the end of a take.
//
// The player reports a take cut short as well as one that ended; it reports the cut once
// the take that replaced it is already sounding. While the player is still playing,
// that report is not the end of what is speaking, so it changes nothing; otherwise a
// flavour line would queue behind an alert rather than being let go (FR-612).
func (s *Scheduler) Finished() {
	if s.player.Playing() {
		return
	}
	s.current = nil
	s.Advance()
}

// Pending reports how many requests are waiting.
func (s *Scheduler) Pending() int { return len(s.queue) }

// Speaking reports whether something is currently being said.
func (s *Scheduler) Speaking() bool { return s.current != nil }

// report forwards a decision to the reaction log when one is wired up.
func (s *Scheduler) report(request Request, outcome string) {
	if s.reporter == nil {
		return
	}
	clip := request.Take.Key()
	s.reporter.Report(ports.Reaction{
		At:      s.clock.Now(),
		Cue:     request.Cue.ID(),
		Clip:    clip,
		Outcome: outcome,
	})
}
