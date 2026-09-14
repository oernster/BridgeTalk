package services_test

// Hand-written fakes for the ports a machine voice is made through, each safe to call from the
// making goroutine and the test at once.

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
)

// The failures the fakes are told to give.
var (
	errModel  = errors.New("the model failed")
	errDisk   = errors.New("the disk is full")
	errFiles  = errors.New("am_michael.bin is missing")
	errDelete = errors.New("a made line is in use")
)

// fakeFiles answers every voice with one material, refusing a voice given a reason in refused.
type fakeFiles struct {
	material ports.Material
	refused  map[string]error
}

func (f fakeFiles) Open(voice machinevoice.Voice) (ports.Material, error) {
	if err := f.refused[voice.ID()]; err != nil {
		return ports.Material{}, err
	}
	return f.material, nil
}

// fakeMaker answers each line with one sample: the count of numbers it was handed. Counting calls
// from one, call failOn fails and call blockOn closes started then waits for its context to end.
type fakeMaker struct {
	mu      sync.Mutex
	calls   int
	failOn  int
	blockOn int
	started chan struct{}
}

func newFakeMaker() *fakeMaker { return &fakeMaker{started: make(chan struct{})} }

func (f *fakeMaker) Make(ctx context.Context, tokens []int64, _ []float32) ([]float32, error) {
	f.mu.Lock()
	f.calls++
	call := f.calls
	f.mu.Unlock()
	switch call {
	case f.blockOn:
		close(f.started)
		<-ctx.Done()
		return nil, ctx.Err()
	case f.failOn:
		return nil, errModel
	}
	return []float32{float32(len(tokens))}, nil
}

// made counts the lines the maker was asked for.
func (f *fakeMaker) made() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// fakeStore keeps made lines in memory and logs every write and delete in order. Counting writes
// from one, write failWrite fails; every delete fails with deleteErr where one is given.
type fakeStore struct {
	mu        sync.Mutex
	keys      map[string][]string
	log       []string
	writes    int
	failWrite int
	deleteErr error
}

func newFakeStore() *fakeStore { return &fakeStore{keys: map[string][]string{}} }

// hold puts made lines on disk for a voice before anything runs.
func (f *fakeStore) hold(voice machinevoice.Voice, keys ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys[voice.ID()] = append(f.keys[voice.ID()], keys...)
}

func (f *fakeStore) Keys(voice machinevoice.Voice) []string { return f.held(voice) }

func (f *fakeStore) Write(voice machinevoice.Voice, key string, _ []float32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	if f.writes == f.failWrite {
		return errDisk
	}
	f.keys[voice.ID()] = append(f.keys[voice.ID()], key)
	f.log = append(f.log, "write "+voice.ID())
	return nil
}

func (f *fakeStore) Path(voice machinevoice.Voice, key string) string {
	return voice.ID() + "/" + key + ".flac"
}

func (f *fakeStore) DeleteAllBut(voice machinevoice.Voice) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, "keep "+voice.ID())
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.keys = map[string][]string{voice.ID(): f.keys[voice.ID()]}
	return nil
}

func (f *fakeStore) DeleteAll() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, "delete all")
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.keys = map[string][]string{}
	return nil
}

// held returns a voice's made lines on disk.
func (f *fakeStore) held(voice machinevoice.Voice) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.keys[voice.ID()])
}

// entries returns the writes and deletes so far, in order.
func (f *fakeStore) entries() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.log)
}
