package services

import "sync"

// modelTurn hands the model to one maker at a time: a run making a cast voice's lines or an audition.
// An audition waiting takes the turn ahead of a run's next line, so its line is made next after the
// line under way (FR-546). A turn covers making a line and writing it, so what is written keeps the
// order the turns were taken in.
type modelTurn struct {
	mu      sync.Mutex
	changed *sync.Cond
	busy    bool
	ahead   int
}

// newModelTurn makes a turn nobody holds.
func newModelTurn() *modelTurn {
	turn := &modelTurn{}
	turn.changed = sync.NewCond(&turn.mu)
	return turn
}

// take waits until nobody holds the model and no audition waits for it, then holds it.
func (t *modelTurn) take() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for t.busy || t.ahead > 0 {
		t.changed.Wait()
	}
	t.busy = true
}

// takeAhead waits for the line under way alone, then holds the model.
func (t *modelTurn) takeAhead() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ahead++
	for t.busy {
		t.changed.Wait()
	}
	t.ahead--
	t.busy = true
}

// give lets the model go to whoever waits.
func (t *modelTurn) give() {
	t.mu.Lock()
	t.busy = false
	t.mu.Unlock()
	t.changed.Broadcast()
}

// waiting counts the auditions waiting for the model.
func (t *modelTurn) waiting() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ahead
}
