// The machine voice half of the audition surface: a machine voice auditioned on the script's groups,
// its line made on the press where it is not yet made (FR-545 to FR-548).
//
// It sits beside audition.go for the reason machine.go sits beside cast.go: a kind of voice's own
// questions are a slice that comes out whole.

package main

import (
	"errors"
	"sync"

	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/take"
)

// auditionGate holds the machine voice audition whose line is being made, one at a time. A press while
// it is made is ignored; Stop lets it go, so its line is kept unplayed (FR-547).
type auditionGate struct {
	mu      sync.Mutex
	making  bool
	stopped int
}

// begin records an audition's line as being made, answering how many stops came before it; false
// where another audition's line is already being made.
func (g *auditionGate) begin() (int, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.making {
		return 0, false
	}
	g.making = true
	return g.stopped, true
}

// end records the line begun after stops as made, answering whether it may still be played: false
// where Stop was pressed since, which has already let it go.
func (g *auditionGate) end(stops int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if stops != g.stopped {
		return false
	}
	g.making = false
	return true
}

// stop lets go of the line being made, answering whether one was.
func (g *auditionGate) stop() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	was := g.making
	g.stopped++
	g.making = false
	return was
}

// busy reports whether an audition's line is being made.
func (g *auditionGate) busy() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.making
}

// MachineAuditionGroups lists what a machine voice is auditioned on: the script's groups, each counting
// its lines (FR-546). Every machine voice speaks the one script, so the list holds for each of them.
func (a *App) MachineAuditionGroups() []GroupDTO {
	groups := a.session.making.AuditionGroups()
	out := make([]GroupDTO, 0, len(groups))
	for _, group := range groups {
		out = append(out, GroupDTO{Key: group.Key, Label: label(group.Key), Clips: group.Lines})
	}
	return out
}

// AuditionMachineVoice plays one line of a group, drawn at random, for the machine voice with the id
// given, cast or not; it plays while muted as every audition does (FR-546). A line not yet made is made
// first. While it is made, Playing says so and a further press is ignored; Stop lets it go unplayed
// (FR-547). A line that cannot be made answers why (FR-548).
func (a *App) AuditionMachineVoice(id, group string) (AuditionDTO, error) {
	voice, err := machinevoice.Parse(id)
	if err != nil {
		return AuditionDTO{}, err
	}
	stops, free := a.session.auditions.begin()
	if !free {
		return AuditionDTO{}, nil
	}
	a.announcePlayback()
	clip, err := a.session.making.Audition(voice, group, a.session.chooser)
	playable := a.session.auditions.end(stops)
	if err != nil {
		a.announcePlayback()
		if errors.Is(err, services.ErrNothingToAudition) {
			return AuditionDTO{}, nothingFor(voice.Name(), group)
		}
		return AuditionDTO{}, err
	}
	if !playable {
		return AuditionDTO{}, nil
	}
	// A made line is one file, so a machine voice auditions a take of one part (FR-573).
	return a.play(group, take.Of(clip))
}
