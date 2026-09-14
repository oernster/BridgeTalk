// Package speechmodel makes a line's samples with the model through ONNX Runtime, loaded with cgo
// disabled (FR-511, CON-8).
//
// The model is loaded from the folder setup fills when a machine voice is cast (FR-544) or when the
// first line is made, whichever comes first: a player who casts only recorded voices never pays for
// loading 310 MB. It then stays loaded until Close. A model that cannot be loaded fails the line with the reason (FR-518); the next line tries
// again, so a folder Repair has put right is used without a restart.
package speechmodel

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrClosed means the maker was closed, so it makes nothing more.
	ErrClosed = errors.New("the speech model is closed")
	// ErrMisshapenInput means a line came with no numbers or no style row. The model is handed the
	// address of each, which an empty one does not have.
	ErrMisshapenInput = errors.New("a line needs its numbers and a style row")
)

// session is the model loaded from one folder. It makes one line at a time until it is released.
type session interface {
	run(tokens []int64, style []float32) ([]float32, error)
	release()
}

// Maker makes lines with the model in one folder. It is safe for use from more than one goroutine;
// lines are made one at a time.
type Maker struct {
	dir  string
	open func(dir string) (session, error)

	mu     sync.Mutex
	loaded session
	closed bool
}

// New makes lines with the model and ONNX Runtime in dir, laid out as voicefiles reads them.
func New(dir string) *Maker { return &Maker{dir: dir, open: openSession} }

// Make answers the samples the model makes for a line's numbers with its style row (FR-511). A
// making already stopped answers why before anything is loaded or run (FR-516); a line takes a few
// hundred milliseconds, so it is not interrupted once started.
func (m *Maker) Make(ctx context.Context, tokens []int64, style []float32) ([]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(tokens) == 0 || len(style) == 0 {
		return nil, fmt.Errorf("%w: %d numbers and %d style values", ErrMisshapenInput, len(tokens), len(style))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return nil, err
	}
	return m.loaded.run(tokens, style)
}

// Load loads the model ahead of the first line, so casting a machine voice spares that line the load
// (FR-544). A model already loaded is left as it is; a making already stopped loads nothing and answers
// why. A load that fails answers the reason, as the line after it would (FR-518).
func (m *Maker) Load(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loadLocked()
}

// loadLocked loads the model where it is not loaded yet; a closed maker loads nothing. The caller
// holds mu.
func (m *Maker) loadLocked() error {
	if m.closed {
		return ErrClosed
	}
	if m.loaded != nil {
		return nil
	}
	loaded, err := m.open(m.dir)
	if err != nil {
		return err
	}
	m.loaded = loaded
	return nil
}

// Close releases the model where it was loaded. Nothing more is made after it.
func (m *Maker) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loaded != nil {
		m.loaded.release()
		m.loaded = nil
	}
	m.closed = true
}
