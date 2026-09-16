package plugin

import (
	"runtime"
	"sync"
)

// runner owns the one operating system thread every call into every plugin is made from.
//
// Two things are bought with one goroutine. A plugin author needs no locking of their own,
// since calls arrive one at a time. And a plugin that initialises something belonging to a
// thread, a COM apartment being the usual one, finds that same thread on the next call: Go
// moves a goroutine between threads freely, so without this a plugin could be initialised
// on one thread and called on another.
//
// No plugin exists to measure, so this is a precaution rather than a finding. It is taken
// now because it cannot be retrofitted once plugins are in the wild (Oliver, 2026-09-16).
type runner struct {
	work chan func()
	stop sync.Once
}

// newRunner starts the thread and returns a runner that sends work to it.
func newRunner() *runner {
	r := &runner{work: make(chan func())}
	ready := make(chan struct{})
	go func() {
		// The thread is locked and never unlocked, so it ends when this goroutine does and
		// is used for nothing else in the meantime.
		runtime.LockOSThread()
		close(ready)
		for job := range r.work {
			job()
		}
		// The loop runs to the end of the channel whatever any one job did, because each job
		// carries its own guard below.
	}()
	<-ready
	return r
}

// do runs fn on the plugin thread and waits for it to finish.
//
// A closed runner runs nothing rather than panicking on a send to a closed channel: a call
// arriving after the application has begun shutting down is a race nobody should have to
// hear about; answering it with the zero value is what a refused call already means.
//
// The job carries a guard of its own, on the plugin thread, because the recover above runs on
// the caller's goroutine and can do nothing for a panic raised on another. A plugin is somebody
// else's code reached through a raw call, so a panic on the way into it or out of it is a thing
// that can happen; without this it would end the whole application, taking the window with it
// over one voice that misbehaved. The call then answers whatever it had, which for every caller
// here is the zero value; that is already what a refused call means.
func (r *runner) do(fn func()) {
	defer func() { _ = recover() }()
	done := make(chan struct{})
	r.work <- func() {
		defer close(done)
		defer func() { _ = recover() }()
		fn()
	}
	<-done
}

// close ends the thread. It is safe to call more than once, since the application closes on
// paths that can overlap.
func (r *runner) close() {
	r.stop.Do(func() { close(r.work) })
}
