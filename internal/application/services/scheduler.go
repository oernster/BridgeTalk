package services

import (
	"sort"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// takeGap is the gap used for a single take, which has no second part.
const takeGap = 0

// Request is one thing waiting to be said. One cue plays one file, so a request
// carrying alternatives speaks the first of them and no more.
type Request struct {
	Cue   cue.Cue
	Clips []string
}

// Scheduler decides what is spoken, in what order and what is interrupted or
// discarded. It holds no goroutine of its own: Submit enqueues and Advance starts
// whatever should be playing, so the caller owns the loop and the tests own time.
type Scheduler struct {
	player   ports.AudioPlayer
	reporter ports.Reporter
	clock    ports.Clock

	queue   []Request
	current *Request
}

// NewScheduler builds a scheduler over an injected player.
func NewScheduler(player ports.AudioPlayer, reporter ports.Reporter, clock ports.Clock) *Scheduler {
	return &Scheduler{player: player, reporter: reporter, clock: clock}
}

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
// submission and again whenever the player reports it has finished.
func (s *Scheduler) Advance() {
	if s.player.Playing() || len(s.queue) == 0 {
		return
	}
	next := s.queue[0]
	s.queue = s.queue[1:]

	clips := next.Clips
	if len(clips) > 1 {
		clips = clips[:1]
	}

	if err := s.player.Play(clips, time.Duration(takeGap)); err != nil {
		s.current = nil
		s.report(next, ports.OutcomeUnbound)
		return
	}
	held := next
	s.current = &held
	s.report(next, ports.OutcomePlayed)
}

// Finished tells the scheduler the player has gone idle.
func (s *Scheduler) Finished() {
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
	clip := ""
	if len(request.Clips) > 0 {
		clip = request.Clips[0]
	}
	s.reporter.Report(ports.Reaction{
		At:      s.clock.Now(),
		Cue:     request.Cue.ID(),
		Clip:    clip,
		Outcome: outcome,
	})
}
