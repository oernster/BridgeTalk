package library

// The checklist behind the Missing takes pane, over real temporary directories. Each test names
// the requirement in REQUIREMENTS.md section 4 it holds.

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

// makeDir creates a directory a test needs to exist.
func makeDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("creating %s: %v", path, err)
	}
}

// FR-316: a folder holding nothing yet is listed beside one with takes; a file is not.
func TestVoiceDirsListsEveryFolderRecordedOrNot(t *testing.T) {
	root := t.TempDir()
	audiotest.WriteTake(t, filepath.Join(root, "Grace", "Docked", "a.wav"))
	makeDir(t, filepath.Join(root, "Oliver"))
	audiotest.WriteFile(t, filepath.Join(root, "notes.txt"), audiotest.NotARecording)

	got, err := VoiceDirs(root)
	if err != nil || !reflect.DeepEqual(got, []string{"Grace", "Oliver"}) {
		t.Fatalf("got %v, %v; want both folders and not the file", got, err)
	}
}

// FR-229: a moment's folder writes each dot of its id as an underscore.
func TestAMomentFolderWritesEachDotAsAnUnderscore(t *testing.T) {
	root := t.TempDir()
	table := journalTable(t, "StartJump.JumpType.Hyperspace")
	makeDir(t, filepath.Join(root, "Oliver"))

	dir, err := MomentFolder(root, "Oliver", "StartJump.JumpType.Hyperspace", table)
	if err != nil {
		t.Fatalf("making the moment's folder: %v", err)
	}
	if want := filepath.Join(root, "Oliver", "StartJump_JumpType_Hyperspace"); dir != want {
		t.Fatalf("got %q, want %q", dir, want)
	}
}

func TestVoiceDirsNeedsARootThatCanBeRead(t *testing.T) {
	if _, err := VoiceDirs(""); !errors.Is(err, ErrNoRoot) {
		t.Errorf("no root: got %v", err)
	}
	if _, err := VoiceDirs(filepath.Join(t.TempDir(), "gone")); err == nil {
		t.Error("a root that is not there was read")
	}
}

// FR-311 and FR-313: both forms of take count as recorded; an empty cue folder does not.
func TestMissingListsWhatAVoiceHasNoTakeFor(t *testing.T) {
	root := t.TempDir()
	table := journalTable(t, "Docked", "Undocked", "Liftoff")
	audiotest.WriteTake(t, filepath.Join(root, "Oliver", "Docked", "a.wav"))
	audiotest.WriteTake(t, filepath.Join(root, "Oliver", "Liftoff.wav"))
	makeDir(t, filepath.Join(root, "Oliver", "Undocked"))

	missing, recorded, err := Missing(root, "Oliver", table)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if recorded != 2 {
		t.Fatalf("recorded %d, want Docked and Liftoff", recorded)
	}
	if len(missing) != 1 || missing[0].ID() != "Undocked" {
		t.Fatalf("missing %v, want Undocked alone", missing)
	}
}

// A voice made with Make folders and nothing saved yet is missing every moment.
func TestAVoiceWithNothingRecordedIsMissingEverything(t *testing.T) {
	root := t.TempDir()
	table := journalTable(t, "Docked", "Undocked")
	makeDir(t, filepath.Join(root, "Oliver"))

	missing, recorded, err := Missing(root, "Oliver", table)
	if err != nil || recorded != 0 || len(missing) != table.Len() {
		t.Fatalf("got %d missing, %d recorded, %v; want everything missing", len(missing), recorded, err)
	}
}

// FR-314: the folder is made where missing; a second press finds it, take and all.
func TestAMomentFolderIsMadeWhereMissingAndKeptWhereNot(t *testing.T) {
	root := t.TempDir()
	table := journalTable(t, "Docked")
	makeDir(t, filepath.Join(root, "Oliver"))

	dir, err := MomentFolder(root, "Oliver", "Docked", table)
	if err != nil || dir != filepath.Join(root, "Oliver", "Docked") {
		t.Fatalf("got %q, %v; want the Docked folder inside Oliver", dir, err)
	}
	take := filepath.Join(dir, "a.wav")
	audiotest.WriteTake(t, take)

	if _, err := MomentFolder(root, "Oliver", "Docked", table); err != nil {
		t.Fatalf("a folder already there was refused: %v", err)
	}
	if held, err := os.ReadFile(take); err != nil || !bytes.Equal(held, audiotest.Recording(t, ".wav")) {
		t.Fatalf("the take already there was changed: %v", err)
	}
}

// FR-315: every way a moment's folder cannot be reached is reported.
func TestAMomentFolderThatCannotBeMadeIsReported(t *testing.T) {
	root := t.TempDir()
	table := journalTable(t, "Docked")

	if _, err := MomentFolder(root, "Oliver", "Docked", table); err == nil {
		t.Error("a voice folder that is not there was used")
	}
	makeDir(t, filepath.Join(root, "Oliver"))
	if _, err := MomentFolder(root, "Oliver", "Nope", table); !errors.Is(err, ErrUnknownCue) {
		t.Errorf("an unknown moment: got %v", err)
	}
	audiotest.WriteFile(t, filepath.Join(root, "Oliver", "Docked"), audiotest.NotARecording)
	if _, err := MomentFolder(root, "Oliver", "Docked", table); err == nil {
		t.Error("a folder was reported made where a file stands")
	}

	audiotest.WriteFile(t, filepath.Join(root, "Plain"), audiotest.NotARecording)
	if _, _, err := Missing(root, "Plain", table); err == nil {
		t.Error("a file was read as a voice folder")
	}
	if _, _, err := Missing("", "Oliver", table); !errors.Is(err, ErrNoRoot) {
		t.Errorf("no root: got %v", err)
	}
	if _, _, err := Missing(root, "a/b", table); !errors.Is(err, ErrVoiceName) {
		t.Errorf("a name that cannot be a folder: got %v", err)
	}
}
