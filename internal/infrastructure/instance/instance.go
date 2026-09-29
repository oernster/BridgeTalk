// Package instance keeps Bridge Talk to one copy at a time (FR-760).
//
// The first start takes a claim and keeps it for the whole run; a start that finds the claim held asks
// the holder to bring its window back, unless told not to, then answers ErrRunning so that it can end
// before it builds anything. The claim is taken from the platform itself: a named mutex with a named
// event on Windows; a locked file with a named pipe beside it on Linux.
package instance

import (
	"errors"
	"sync"
)

// ErrRunning answers a start that found another copy already running.
var ErrRunning = errors.New("another copy of the application is already running")

// Claim is this run's hold on being the one copy. Summons fires each time a later start asks for the
// window back; Release gives the claim up.
type Claim struct {
	summons  chan struct{}
	release  func()
	released sync.Once
}

// Summons yields once for each later start that asked for the window back. Two asks arriving before
// the first is read are one: the window comes back either way.
func (c *Claim) Summons() <-chan struct{} { return c.summons }

// Release gives the claim up, so that a later start can take it. It is safe to call more than once.
func (c *Claim) Release() {
	c.released.Do(c.release)
}

// newClaim builds a claim whose summons channel holds one ask and whose release runs release once.
func newClaim(release func()) *Claim {
	return &Claim{summons: make(chan struct{}, 1), release: release}
}

// summon passes one ask to the claim's reader without waiting, since an ask already waiting says the
// same thing.
func (c *Claim) summon() {
	select {
	case c.summons <- struct{}{}:
	default:
	}
}

// inert answers a claim that holds nothing and is never summoned, for a start whose claim could not be
// made: that start runs regardless (FR-760), so it is handed a claim that does nothing rather than
// none.
func inert() *Claim { return newClaim(func() {}) }

// Take claims the one copy named name. dir is a folder the claim may keep its files in, which only
// Linux uses. summon says whether a start that finds the claim held asks the holder for its window.
//
// It answers the claim and nil for the first start; nil and ErrRunning for a later one. A claim that
// cannot be made answers an inert claim with the reason, so the caller can say why and carry on.
func Take(name, dir string, summon bool) (*Claim, error) {
	return take(name, dir, summon)
}
