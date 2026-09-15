// The Chatter surface of the facade: every moment the game raises, switched on or off one at a time,
// a category at a time or all at once (section 7.2).
//
// It sits beside app.go for the reason checklist.go does: the facade is first to reach the size cap; a
// pane's own questions are a slice that comes out whole.

package main

import (
	"errors"

	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// errChatterNotReady refuses a switch asked of a session built with nothing holding the switches.
var errChatterNotReady = errors.New("nothing can be switched: Chatter is not ready")

// Chatter answers what the Chatter pane shows (FR-727). A session built without the switches, as a
// test may build one, lists nothing.
func (a *App) Chatter() ChatterDTO { return a.chatterAfter(nil) }

// SetMoment switches one moment on or off and answers the pane as it now stands (FR-729).
func (a *App) SetMoment(id string, on bool) (ChatterDTO, error) {
	return a.switched(func(chatter *services.ChatterService) error {
		return chatter.SetMoment(cue.ID(id), on)
	})
}

// SetCategory switches every moment in one category on or off (FR-731).
func (a *App) SetCategory(name string, on bool) (ChatterDTO, error) {
	return a.switched(func(chatter *services.ChatterService) error {
		return chatter.SetCategory(name, on)
	})
}

// SetAllMoments switches every moment on or off (FR-732).
func (a *App) SetAllMoments(on bool) (ChatterDTO, error) {
	return a.switched(func(chatter *services.ChatterService) error {
		return chatter.SetAll(on)
	})
}

// switched makes one change, then answers the pane after it.
//
// A moment or a category Chatter does not list is refused. A switch that applied but could not be
// kept is not: it applies until the application closes, so the pane is answered as it stands with the
// reason beside it (FR-633), rather than the page being told the press failed when it did not.
func (a *App) switched(change func(*services.ChatterService) error) (ChatterDTO, error) {
	if a.session.chatter == nil {
		return a.chatterAfter(nil), errChatterNotReady
	}
	err := change(a.session.chatter)
	if errors.Is(err, services.ErrNoSuchMoment) {
		return a.chatterAfter(nil), err
	}
	return a.chatterAfter(err), nil
}

// chatterAfter reads the pane, naming why the last switch was not kept where it was not.
func (a *App) chatterAfter(notKept error) ChatterDTO {
	answer := ChatterDTO{Categories: []ChatterCategoryDTO{}}
	if notKept != nil {
		answer.Problem = notKept.Error()
	}
	if a.session.chatter == nil {
		return answer
	}
	for _, category := range a.session.chatter.Categories() {
		moments := make([]ChatterMomentDTO, 0, len(category.Moments))
		for _, each := range category.Moments {
			moments = append(moments, ChatterMomentDTO{
				Cue: cueLine(each.Cue.ID(), each.Cue.Purpose()),
				On:  each.On,
			})
		}
		answer.Categories = append(answer.Categories, ChatterCategoryDTO{Name: category.Name, Moments: moments})
	}
	return answer
}
