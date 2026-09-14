package speechmodel

// FR-511, FR-516 and FR-518 as the model sees them: it is loaded when the first line is made; a
// stopped making neither loads nor runs it; a failure says why without leaving it unusable. A
// hand-written session stands in for ONNX Runtime; the real one is tested beside it on Windows.

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// fakeSession answers every line with samples of its own, recording what it was handed.
type fakeSession struct {
	samples  []float32
	failure  error
	runs     int
	released int
	tokens   []int64
	style    []float32
}

func (s *fakeSession) run(tokens []int64, style []float32) ([]float32, error) {
	s.runs++
	s.tokens, s.style = tokens, style
	return s.samples, s.failure
}

func (s *fakeSession) release() { s.released++ }

// opener hands out the session given, failing the first failures times with loadFailure.
type opener struct {
	session     *fakeSession
	failures    int
	loadFailure error
	opened      []string
}

func (o *opener) open(dir string) (session, error) {
	o.opened = append(o.opened, dir)
	if len(o.opened) <= o.failures {
		return nil, o.loadFailure
	}
	return o.session, nil
}

// makerOver is a maker over the folder given whose sessions come from the opener.
func makerOver(dir string, source *opener) *Maker {
	maker := New(dir)
	maker.open = source.open
	return maker
}

// A line and its style row, as the making service hands them over.
var (
	aLine  = []int64{0, 50, 83, 0}
	aStyle = []float32{0.25, 0.5}
)

// Nothing is loaded until a line is made; then the model is loaded once from the folder and every
// line is handed to it unchanged.
func TestTheModelIsLoadedOnceWhenTheFirstLineIsMade(t *testing.T) {
	t.Parallel()
	source := &opener{session: &fakeSession{samples: []float32{0.1, -0.1}}}
	maker := makerOver("installed", source)
	if len(source.opened) != 0 {
		t.Fatalf("loaded %v before any line was made", source.opened)
	}

	for range 2 {
		samples, err := maker.Make(context.Background(), aLine, aStyle)
		if err != nil || !slices.Equal(samples, []float32{0.1, -0.1}) {
			t.Fatalf("Make = %v, %v; want the model's samples", samples, err)
		}
	}

	if !slices.Equal(source.opened, []string{"installed"}) {
		t.Errorf("loaded from %v, want once from installed", source.opened)
	}
	session := source.session
	if session.runs != 2 || !slices.Equal(session.tokens, aLine) || !slices.Equal(session.style, aStyle) {
		t.Errorf("the model ran %d times with %v and %v; want twice with %v and %v",
			session.runs, session.tokens, session.style, aLine, aStyle)
	}
}

// A making already stopped (FR-516) answers why, loading and running nothing.
func TestAStoppedMakingNeitherLoadsNorRuns(t *testing.T) {
	t.Parallel()
	source := &opener{session: &fakeSession{}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := makerOver("installed", source).Make(ctx, aLine, aStyle)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Make = %v, want %v", err, context.Canceled)
	}
	if len(source.opened) != 0 || source.session.runs != 0 {
		t.Errorf("loaded %v and ran %d lines for a stopped making", source.opened, source.session.runs)
	}
}

// A model that cannot be loaded fails the line saying why (FR-518); the next line tries again, so a
// folder Repair has put right is used.
func TestAModelThatCannotBeLoadedIsTriedAgain(t *testing.T) {
	t.Parallel()
	missing := errors.New("the model is missing")
	source := &opener{session: &fakeSession{samples: []float32{0.5}}, failures: 1, loadFailure: missing}
	maker := makerOver("installed", source)

	if _, err := maker.Make(context.Background(), aLine, aStyle); !errors.Is(err, missing) {
		t.Errorf("the first line = %v, want %v", err, missing)
	}
	if samples, err := maker.Make(context.Background(), aLine, aStyle); err != nil || len(samples) != 1 {
		t.Errorf("the second line = %v, %v; want it made", samples, err)
	}
	if len(source.opened) != 2 {
		t.Errorf("loaded %d times, want twice", len(source.opened))
	}
}

// A line the model refuses fails with the model's reason (FR-518), leaving it loaded for the next.
func TestALineTheModelRefusesLeavesItLoaded(t *testing.T) {
	t.Parallel()
	refused := errors.New("the model refused the line")
	source := &opener{session: &fakeSession{failure: refused}}
	maker := makerOver("installed", source)

	for range 2 {
		if _, err := maker.Make(context.Background(), aLine, aStyle); !errors.Is(err, refused) {
			t.Errorf("Make = %v, want %v", err, refused)
		}
	}
	if len(source.opened) != 1 || source.session.released != 0 {
		t.Errorf("loaded %d times and released %d; want loaded once and kept", len(source.opened), source.session.released)
	}
}

// A line with no numbers or no style row is refused before anything is loaded: the model is handed
// the address of each, which an empty one does not have.
func TestAnEmptyLineOrStyleIsRefusedLoadingNothing(t *testing.T) {
	t.Parallel()
	source := &opener{session: &fakeSession{}}
	maker := makerOver("installed", source)

	for _, each := range []struct {
		tokens []int64
		style  []float32
	}{{nil, aStyle}, {aLine, nil}} {
		if _, err := maker.Make(context.Background(), each.tokens, each.style); !errors.Is(err, ErrMisshapenInput) {
			t.Errorf("Make(%v, %v) = %v, want %v", each.tokens, each.style, err, ErrMisshapenInput)
		}
	}
	if len(source.opened) != 0 {
		t.Errorf("loaded %v for lines that cannot be made", source.opened)
	}
}

// Closing releases a loaded model once; after it nothing more is made or loaded. Closing a maker
// that never loaded releases nothing.
func TestClosingReleasesTheModelOnceAndRefusesMore(t *testing.T) {
	t.Parallel()
	source := &opener{session: &fakeSession{samples: []float32{0.5}}}
	maker := makerOver("installed", source)
	if _, err := maker.Make(context.Background(), aLine, aStyle); err != nil {
		t.Fatalf("Make: %v", err)
	}

	maker.Close()
	maker.Close()

	if source.session.released != 1 {
		t.Errorf("released %d times, want once", source.session.released)
	}
	if _, err := maker.Make(context.Background(), aLine, aStyle); !errors.Is(err, ErrClosed) {
		t.Errorf("Make after Close = %v, want %v", err, ErrClosed)
	}
	if len(source.opened) != 1 {
		t.Errorf("loaded %d times, want once", len(source.opened))
	}

	unused := &opener{session: &fakeSession{}}
	makerOver("installed", unused).Close()
	if len(unused.opened) != 0 || unused.session.released != 0 {
		t.Errorf("closing an unused maker loaded %v and released %d", unused.opened, unused.session.released)
	}
}

// FR-544: loading ahead loads the model once from the folder; the lines after it use that model.
func TestLoadingAheadLoadsTheModelOnceForTheLinesAfter(t *testing.T) {
	t.Parallel()
	source := &opener{session: &fakeSession{samples: []float32{0.5}}}
	maker := makerOver("installed", source)

	for range 2 {
		if err := maker.Load(context.Background()); err != nil {
			t.Fatalf("Load: %v", err)
		}
	}
	if _, err := maker.Make(context.Background(), aLine, aStyle); err != nil {
		t.Fatalf("Make: %v", err)
	}

	if !slices.Equal(source.opened, []string{"installed"}) || source.session.runs != 1 {
		t.Errorf("loaded from %v and ran %d lines; want loaded once from installed, then one line",
			source.opened, source.session.runs)
	}
}

// A load for a stopped making loads nothing; a load that fails says why, the next line trying again
// (FR-518); after Close nothing is loaded.
func TestALoadStoppedFailingOrClosedLoadsNothingMore(t *testing.T) {
	t.Parallel()
	missing := errors.New("the model is missing")
	source := &opener{session: &fakeSession{samples: []float32{0.5}}, failures: 1, loadFailure: missing}
	maker := makerOver("installed", source)
	stopped, cancel := context.WithCancel(context.Background())
	cancel()

	if err := maker.Load(stopped); !errors.Is(err, context.Canceled) || len(source.opened) != 0 {
		t.Errorf("a stopped Load = %v having loaded %v; want %v with nothing loaded", err, source.opened, context.Canceled)
	}
	if err := maker.Load(context.Background()); !errors.Is(err, missing) {
		t.Errorf("Load = %v, want %v", err, missing)
	}
	if _, err := maker.Make(context.Background(), aLine, aStyle); err != nil {
		t.Errorf("the line after a failed load = %v, want it made", err)
	}
	maker.Close()
	if err := maker.Load(context.Background()); !errors.Is(err, ErrClosed) || len(source.opened) != 2 {
		t.Errorf("Load after Close = %v having loaded %d times; want %v with no third load",
			err, len(source.opened), ErrClosed)
	}
}
