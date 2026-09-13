package library

// FR-215's second figure measures every recording under a voice, not only the ones the scan
// could use, so a file nothing reaches shows as a shortfall rather than vanishing.

import (
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

func TestPresentCountsEveryRecognisedRecordingUnderTheVoice(t *testing.T) {
	root := t.TempDir()
	lena := filepath.Join(root, "Lena")
	audiotest.WriteTake(t, filepath.Join(lena, "DockingGranted.wav"))
	audiotest.WriteTake(t, filepath.Join(lena, "typo.wav"))
	audiotest.WriteTake(t, filepath.Join(lena, "not a cue", "x.mp3"))
	audiotest.WriteTake(t, filepath.Join(lena, "DockingGranted", "older", "c.wav"))
	audiotest.WriteFile(t, filepath.Join(lena, "StartJump.mp3"), audiotest.NotARecording)
	audiotest.WriteFile(t, filepath.Join(lena, "notes.txt"), audiotest.NotARecording)

	voices, _ := scanned(t, root, journalTable(t, "DockingGranted", "StartJump"))

	found := only(t, voices)
	if found.Takes != 1 || found.Present != 5 {
		t.Fatalf("got %d takes of %d present, want the one that plays of the five recordings there",
			found.Takes, found.Present)
	}
}
