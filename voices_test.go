package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/library"
)

// captureStderr runs work with the process's standard error redirected into a pipe and
// returns what it wrote.
func captureStderr(t *testing.T, work func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("opening a pipe: %v", err)
	}
	original := os.Stderr
	os.Stderr = writer
	defer func() { os.Stderr = original }()

	work()

	if err := writer.Close(); err != nil {
		t.Fatalf("closing the pipe: %v", err)
	}
	written, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading the pipe: %v", err)
	}
	return string(written)
}

// Everything a scan passed over is named at startup, whichever way it was passed over,
// so a misspelled folder and an empty directory never look like the same silence.
func TestEverythingAScanPassedOverIsNamed(t *testing.T) {
	report := library.Report{
		Empty:       []library.Reason{{Path: "Nobody", Why: "nothing to play"}},
		Unmatched:   []library.Reason{{Path: "Alpha/holiday.mp3", Why: "no cue's name"}},
		Duplicated:  []library.Reason{{Path: "Alpha/DOCKED", Why: "differs only in case"}},
		Undecodable: []library.Reason{{Path: "Alpha/Docked/broken.mp3", Why: "will not play"}},
	}

	written := captureStderr(t, func() { warnAbout(report) })

	for _, want := range []string{
		"note: Nobody: nothing to play",
		"note: Alpha/holiday.mp3: no cue's name",
		"note: Alpha/DOCKED: differs only in case",
		"note: Alpha/Docked/broken.mp3: will not play",
	} {
		if !strings.Contains(written, want) {
			t.Errorf("stderr = %q, want it to carry %q", written, want)
		}
	}
}

// A scan that passed nothing over says nothing, so a clean start stays quiet.
func TestACleanScanPrintsNothing(t *testing.T) {
	if written := captureStderr(t, func() { warnAbout(library.Report{}) }); written != "" {
		t.Errorf("stderr = %q, want nothing", written)
	}
}

// An application nobody has pointed at their recordings yet is not in trouble, so an
// unset root finds nothing and says nothing. A chosen root that cannot be read still
// warns, because somebody chose it; a readable one finds its voices in silence.
func TestOnlyAChosenLibraryRootThatCannotBeReadWarns(t *testing.T) {
	table := fixtureTable(t)
	cases := []struct {
		name   string
		root   string
		voices int
		warns  bool
	}{
		{"an unset root is silent", "", 0, false},
		{"a chosen root that is missing warns", filepath.Join(t.TempDir(), "gone"), 0, true},
		{"a readable root finds its voices", libraryRootFixture(t), 2, false},
	}
	for _, each := range cases {
		var found []library.Voice
		written := captureStderr(t, func() { found, _ = scanLibrary(each.root, table) })
		if len(found) != each.voices {
			t.Errorf("%s: found %d voices, want %d", each.name, len(found), each.voices)
		}
		if warned := strings.Contains(written, "warning:"); warned != each.warns {
			t.Errorf("%s: stderr = %q, want a warning: %v", each.name, written, each.warns)
		}
	}
}
