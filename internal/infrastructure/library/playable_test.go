package library

// FR-204 over a real disk: a take named for a cue that will not play is left out of its
// voice and named in the scan report, whichever form it takes; the scan goes on.

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

func TestATakeThatWillNotPlayIsLeftOutAndReported(t *testing.T) {
	root := t.TempDir()
	ivy := filepath.Join(root, "Ivy")
	audiotest.WriteTake(t, filepath.Join(ivy, "DockingGranted.wav"))
	audiotest.WriteFile(t, filepath.Join(ivy, "DockingGranted.2.wav"), audiotest.NotARecording)
	audiotest.WriteTake(t, filepath.Join(ivy, "StartJump", "a.mp3"))
	audiotest.WriteFile(t, filepath.Join(ivy, "StartJump", "b.mp3"), audiotest.NotARecording)
	audiotest.WriteFile(t, filepath.Join(root, "Jack", "DockingGranted.ogg"), audiotest.NotARecording)

	voices, report := scanned(t, root, journalTable(t, "DockingGranted", "StartJump"))

	found := only(t, voices)
	if found.Name != "Ivy" || found.Takes != 2 || found.Cues() != 2 {
		t.Fatalf("got %s with %d takes over %d cues, want Ivy with the two that play",
			found.Name, found.Takes, found.Cues())
	}
	want := []string{
		filepath.Join("Ivy", "DockingGranted.2.wav"),
		filepath.Join("Ivy", "StartJump", "b.mp3"),
		filepath.Join("Jack", "DockingGranted.ogg"),
	}
	if got := paths(report.Undecodable); !reflect.DeepEqual(got, want) {
		t.Errorf("undecodable = %v, want %v", got, want)
	}
	for _, reason := range report.Undecodable {
		if !strings.Contains(reason.Why, "will not play") || strings.Contains(reason.Why, root) {
			t.Errorf("%s: reason %q, want why it will not play without the path", reason.Path, reason.Why)
		}
	}
	// A voice whose only take will not play is no voice (FR-209).
	if got := paths(report.Empty); !reflect.DeepEqual(got, []string{"Jack"}) {
		t.Errorf("empty = %v, want Jack", got)
	}
}

// ScanVoice asks the same question, so the Missing takes pane counts only what plays.
func TestAVoiceScannedAloneLeavesOutWhatWillNotPlay(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Kim")
	audiotest.WriteTake(t, filepath.Join(dir, "DockingGranted.mp3"))
	audiotest.WriteFile(t, filepath.Join(dir, "StartJump.wav"), audiotest.NotARecording)

	kim, report := ScanVoice(dir, journalTable(t, "DockingGranted", "StartJump"))
	if kim.Takes != 1 || len(report.Undecodable) != 1 || !report.Any() {
		t.Fatalf("got %d takes and %+v, want one take with StartJump.wav reported", kim.Takes, report)
	}
}
