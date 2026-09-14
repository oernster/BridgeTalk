package library

// FR-215's second figure measures every recording under a voice, not only the ones the scan
// could use, so a file nothing reaches shows as a shortfall rather than vanishing.

import (
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
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

// FR-215, second figure: distinct files used against the recordings present. One file
// answering two cues is one file used; a recording present that answers nothing is not used.
func TestFilesCountDistinctFilesUsedAgainstRecordingsPresent(t *testing.T) {
	voice := voiceOf("Ivy", map[cue.ID][]string{
		"DockingGranted":              {"shared.wav"},
		"ShieldState.ShieldsUp.false": {"shared.wav", "own.wav"},
	})
	voice.Present = 4

	used, present := voice.Files()

	if used != 2 || present != 4 {
		t.Errorf("files = %d used of %d present, want 2 of 4", used, present)
	}
}
