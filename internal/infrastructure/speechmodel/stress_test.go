//go:build benchmarks && (windows || linux)

package speechmodel_test

// FR-511 under the conditions that broke a build on 2026-09-14: each line made on a fresh goroutine
// whose stack has just grown, while another goroutine forces collections that shrink stacks. An
// address handed to ONNX Runtime from a stack that moves before the call leaves ONNX Runtime reading
// and writing the old stack: measured that day as panics, empty lines and "NULL input" refusals. It
// carries the benchmarks tag, since it takes the real model half a minute.

import (
	"context"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
	"github.com/oernster/bridge-talk/internal/infrastructure/speechmodel"
)

const (
	// stressLines is how many lines are made. 13 of 300 broke before the fix, so a broken caller
	// survives all 150 about once in 700 runs.
	stressLines = 150
	// stressGCPercent collects after almost every allocation, so stacks shrink as often as they can.
	stressGCPercent = 1
	// growDepth is how many frames grow runs before each line, so the line uses too little of the
	// grown stack to keep it.
	growDepth = 512
	// frameBytes is how much each frame of grow holds.
	frameBytes = 1 << 10
)

// grow recurses depth frames of frameBytes each, growing the goroutine's stack.
//
//go:noinline
func grow(depth int) byte {
	var pad [frameBytes]byte
	if depth == 0 {
		return pad[0]
	}
	return grow(depth-1) + pad[1]
}

func TestLinesSurviveTheirStackMovingWhileTheModelIsCalled(t *testing.T) {
	dir := modelfilestest.Require(t)
	tokens, row := lineFor(t, dir, "bf_emma")
	maker := speechmodel.New(dir)
	defer maker.Close()

	defer debug.SetGCPercent(debug.SetGCPercent(stressGCPercent))
	var stop atomic.Bool
	collecting := make(chan struct{})
	go func() {
		defer close(collecting)
		for !stop.Load() {
			runtime.GC()
		}
	}()

	var panics, failures, empty int
	for range stressLines {
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() {
				if recovered := recover(); recovered != nil {
					panics++
					t.Logf("panic: %v", recovered)
				}
			}()
			_ = grow(growDepth)
			samples, err := maker.Make(context.Background(), tokens, row)
			switch {
			case err != nil:
				failures++
				t.Logf("error: %v", err)
			case len(samples) == 0:
				empty++
			}
		}()
		<-done
	}
	stop.Store(true)
	<-collecting

	if broken := panics + failures + empty; broken > 0 {
		t.Errorf("%d of %d lines broke: %d panics, %d errors, %d empty", broken, stressLines, panics, failures, empty)
	}
}
