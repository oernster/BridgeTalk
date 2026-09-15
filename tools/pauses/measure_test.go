package main

// FR-551 and FR-555: a full run finds and writes the pauses, then the endings; a run with -endings-only
// finds and writes the endings alone, never asking the break finder or writing the pauses.

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

// measureInto measures bf_emma over the joined script, writing both books into dir as the options
// say; it answers the break finder and the ending finder it was handed.
func measureInto(t *testing.T, dir string, endingsOnly bool) (*fakeFinder, *fakeEndingFinder) {
	t.Helper()
	finder, ends := &fakeFinder{answer: sure}, &fakeEndingFinder{answer: startingAt}
	run := finding{
		files: voicefiles.New(voiceFolder(t, "bf_emma")), maker: &fakeMaker{}, finder: finder, ends: ends,
		work: t.TempDir(), out: io.Discard,
	}
	chosen := options{
		voices: voicesNamed(t, "bf_emma"), out: filepath.Join(dir, "pauses.toml"),
		endings: filepath.Join(dir, "endings.toml"), endingsOnly: endingsOnly,
	}
	if err := run.measure(context.Background(), joinedScript(t), chosen); err != nil {
		t.Fatalf("measure: %v", err)
	}
	return finder, ends
}

// A full run asks each finder once for the voice and writes both books.
func TestAFullRunWritesThePausesAndTheEndings(t *testing.T) {
	dir := t.TempDir()
	finder, ends := measureInto(t, dir, false)
	if len(finder.asked) != 1 || len(ends.asked) != 1 {
		t.Errorf("break finder asked %d times, ending finder %d; want once each", len(finder.asked), len(ends.asked))
	}
	for _, name := range []string{"pauses.toml", "endings.toml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s not written: %v", name, err)
		}
	}
}

// FR-555: a run with -endings-only asks the ending finder alone and writes the endings, leaving the
// pauses file unwritten.
func TestARunWithEndingsOnlyWritesTheEndingsAloneAskingNoBreakFinder(t *testing.T) {
	dir := t.TempDir()
	finder, ends := measureInto(t, dir, true)
	if len(finder.asked) != 0 || len(ends.asked) != 1 {
		t.Errorf("break finder asked %d times, ending finder %d; want never and once", len(finder.asked), len(ends.asked))
	}
	if _, err := os.Stat(filepath.Join(dir, "pauses.toml")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the pauses file was touched: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "endings.toml")); err != nil {
		t.Errorf("endings.toml not written: %v", err)
	}
}
