// Package makingtest holds hand-written fakes of the ports a machine voice is made through, so every
// suite that makes lines over fakes builds them the same way. Each is safe to call from the making
// goroutine and the test at once.
package makingtest

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// The failures the fakes give when told to.
var (
	// ErrModel is the failure Maker gives on call FailOn.
	ErrModel = errors.New("the model failed")
	// ErrDisk is the failure Store gives on write FailWrite.
	ErrDisk = errors.New("the disk is full")
)

// MadeFrom are the digests Material answers with, so every made line over these fakes is keyed by
// them.
var MadeFrom = making.Files{Style: "style-1", Model: "model-1"}

// Key is the key a line with these sounds is made under from MadeFrom.
func Key(sounds string) string { return making.Key(sounds, MadeFrom) }

// Material is a style of zeros under MadeFrom.
func Material() ports.Material {
	// NewStyle refuses only a count other than MaxSymbols rows of StyleWidth, which this is not.
	style, _ := speech.NewStyle(make([]float32, speech.MaxSymbols*speech.StyleWidth))
	return ports.Material{Style: style, Files: MadeFrom}
}

// Files answers every voice with Material, refusing a voice whose id Refused gives a reason for.
type Files struct {
	Material ports.Material
	Refused  map[string]error
}

// Open answers the material; where the voice is refused, the reason.
func (f Files) Open(voice machinevoice.Voice) (ports.Material, error) {
	if err := f.Refused[voice.ID()]; err != nil {
		return ports.Material{}, err
	}
	return f.Material, nil
}

// Maker answers each line with one sample: the count of numbers it was handed. Counting calls from
// one, call FailOn fails with ErrModel and call BlockOn closes Started then waits for its context to
// end. Call PauseOn closes Started then waits for Resume to be closed, making its line; where its
// context ends first, it answers why. It counts the times it is closed.
type Maker struct {
	FailOn  int
	BlockOn int
	PauseOn int
	Started chan struct{}
	Resume  chan struct{}

	mu     sync.Mutex
	calls  int
	closes int
}

// NewMaker makes a Maker that neither fails nor blocks until told to.
func NewMaker() *Maker { return &Maker{Started: make(chan struct{}), Resume: make(chan struct{})} }

// Make answers one line's samples.
func (f *Maker) Make(ctx context.Context, tokens []int64, _ []float32) ([]float32, error) {
	f.mu.Lock()
	f.calls++
	call := f.calls
	f.mu.Unlock()
	switch call {
	case f.BlockOn:
		close(f.Started)
		<-ctx.Done()
		return nil, ctx.Err()
	case f.PauseOn:
		close(f.Started)
		select {
		case <-f.Resume:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case f.FailOn:
		return nil, ErrModel
	}
	return []float32{float32(len(tokens))}, nil
}

// Made counts the lines the maker was asked for.
func (f *Maker) Made() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// Close records that the model was released.
func (f *Maker) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closes++
}

// Closed counts the times Close was called.
func (f *Maker) Closed() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closes
}

// Store keeps made lines in memory by voice id, logging every write and delete in order.
// Counting writes from one, write FailWrite fails with ErrDisk; every delete fails with DeleteErr,
// deleting nothing, where one is given.
type Store struct {
	FailWrite int
	DeleteErr error

	mu     sync.Mutex
	keys   map[string][]string
	log    []string
	writes int
}

// NewStore makes an empty Store.
func NewStore() *Store { return &Store{keys: map[string][]string{}} }

// Hold puts made lines on disk for the voice with this id before anything runs.
func (f *Store) Hold(id string, keys ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys[id] = append(f.keys[id], keys...)
}

// Keys lists a voice's made lines.
func (f *Store) Keys(voice machinevoice.Voice) []string { return f.Held(voice.ID()) }

// Write keeps a made line, logging it.
func (f *Store) Write(voice machinevoice.Voice, key string, _ []float32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	if f.writes == f.FailWrite {
		return ErrDisk
	}
	f.keys[voice.ID()] = append(f.keys[voice.ID()], key)
	f.log = append(f.log, "write "+voice.ID())
	return nil
}

// Path is where a made line is played from (PathOf).
func (f *Store) Path(voice machinevoice.Voice, key string) string { return PathOf(voice.ID(), key) }

// PathOf is where Store plays a made line from: the voice's id, then the key.
func PathOf(id, key string) string { return id + "/" + key + ".flac" }

// Delete deletes a voice's made lines under the keys given, logging it.
func (f *Store) Delete(voice machinevoice.Voice, keys []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, "delete "+voice.ID())
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	f.keys[voice.ID()] = slices.DeleteFunc(f.keys[voice.ID()], func(key string) bool { return slices.Contains(keys, key) })
	return nil
}

// Held returns the made lines on disk for the voice with this id.
func (f *Store) Held(id string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.keys[id])
}

// Entries returns the writes and deletes so far, in order.
func (f *Store) Entries() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.log)
}

// AwaitLimit is how long a test waits on the fakes. They answer in microseconds, so reaching it means
// what was waited for is never going to happen.
const AwaitLimit = 2 * time.Second

// Await waits for ch to close, failing the test with what never happened once AwaitLimit passes, so a
// test whose making never starts fails at once rather than blocking until the whole run times out.
func Await(t testing.TB, ch <-chan struct{}, what string) { awaitWithin(t, ch, what, AwaitLimit) }

// awaitWithin is Await with the limit given.
func awaitWithin(t testing.TB, ch <-chan struct{}, what string, limit time.Duration) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(limit):
		t.Fatalf("%s did not happen within %v", what, limit)
	}
}
