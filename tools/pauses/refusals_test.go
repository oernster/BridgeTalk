package main

// FR-551: a run that cannot find every pause it was asked for stops, naming what stopped it, rather
// than writing pauses that could not be trusted.

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

// A finder answering a different number of files than it was handed is refused, naming the voice.
func TestAFinderAnsweringTheWrongNumberOfLinesIsRefused(t *testing.T) {
	finder := &fakeFinder{answer: func(asked request) ([]found, error) {
		answered, _ := sure(asked)
		return answered[1:], nil
	}}
	_, _, err := runFinding(t, &fakeMaker{}, finder, voiceFolder(t, "bf_emma"), "bf_emma")
	if !errors.Is(err, errAnswerCount) || !strings.Contains(err.Error(), "bf_emma") {
		t.Errorf("got %v, want errAnswerCount naming bf_emma", err)
	}
}

// A cut outside the samples of its line is refused, naming the voice, the cue and the line.
func TestACutOutsideItsLineIsRefused(t *testing.T) {
	maker := &fakeMaker{}
	finder := &fakeFinder{answer: func(asked request) ([]found, error) {
		answered, _ := sure(asked)
		answered[0].Cut = ref(len(maker.made[0]))
		return answered, nil
	}}
	_, _, err := runFinding(t, maker, finder, voiceFolder(t, "bf_emma"), "bf_emma")
	if !errors.Is(err, errCutOutside) || !strings.Contains(err.Error(), `bf_emma "Docked" line 1`) {
		t.Errorf("got %v, want errCutOutside naming bf_emma \"Docked\" line 1", err)
	}
}

// A maker that fails stops the run before the finder is asked, naming the voice, the cue and the line.
func TestAMakerThatFailsStopsTheRunNamingTheVoiceAndTheLine(t *testing.T) {
	broken := errors.New("the model refused")
	finder := &fakeFinder{answer: sure}
	_, _, err := runFinding(t, &fakeMaker{failAt: 2, err: broken}, finder, voiceFolder(t, "bf_emma"), "bf_emma")
	if !errors.Is(err, broken) || !strings.Contains(err.Error(), `bf_emma "Docked" line 3`) {
		t.Errorf("got %v, want the maker's error naming bf_emma \"Docked\" line 3", err)
	}
	if len(finder.asked) != 0 {
		t.Errorf("the finder was asked %d times after the maker failed", len(finder.asked))
	}
}

// A finder that fails stops the run, keeping its reason and naming the voice.
func TestAFinderThatFailsStopsTheRunNamingTheVoice(t *testing.T) {
	broken := errors.New("no venv")
	finder := &fakeFinder{answer: func(request) ([]found, error) { return nil, broken }}
	_, _, err := runFinding(t, &fakeMaker{}, finder, voiceFolder(t, "bf_emma"), "bf_emma")
	if !errors.Is(err, broken) || !strings.Contains(err.Error(), "bf_emma") {
		t.Errorf("got %v, want the finder's error naming bf_emma", err)
	}
}

// A voice whose files cannot be opened stops the run, naming the file.
func TestAVoiceWhoseFilesCannotBeOpenedStopsTheRun(t *testing.T) {
	_, _, err := runFinding(t, &fakeMaker{}, &fakeFinder{answer: sure}, voiceFolder(t, "bf_emma"), "bf_emma", "am_michael")
	if err == nil || !strings.Contains(err.Error(), "am_michael.bin") {
		t.Errorf("got %v, want a failure naming am_michael.bin", err)
	}
}

// Pauses found with two different model files would ship a book no model matches, so it is refused.
func TestAModelFileThatChangesDuringTheRunIsRefused(t *testing.T) {
	dir := voiceFolder(t, "bf_emma", "bm_george")
	finder := &fakeFinder{answer: func(asked request) ([]found, error) {
		write(t, filepath.Join(dir, voicefiles.ModelFile), []byte("another model"))
		return sure(asked)
	}}
	if _, _, err := runFinding(t, &fakeMaker{}, finder, dir, "bf_emma", "bm_george"); !errors.Is(err, errModelChanged) {
		t.Errorf("got %v, want errModelChanged", err)
	}
}
