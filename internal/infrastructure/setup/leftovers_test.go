package setup

// FR-525 and FR-805: an uninstall deletes the made lines whatever is ticked; the window's state goes
// only when forgetting is asked for. The recordings kept beside the made lines are never touched.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestTheMadeLinesGoWhateverIsTickedWhileTheRecordingsBesideThemStay(t *testing.T) {
	t.Parallel()
	for _, forget := range []bool{false, true} {
		base := t.TempDir()
		data := filepath.Join(base, "data")
		left := Leftovers{MadeLines: filepath.Join(data, "made"), State: filepath.Join(base, "state")}
		plant(t, left.MadeLines, "bf_emma/a.flac", "a made line")
		plant(t, data, "recordings/Hugo/a.flac", "a recording")
		plant(t, left.State, "theme", "dark")

		if err := RemoveLeftovers(left, forget); err != nil {
			t.Fatalf("forgetting %v: %v", forget, err)
		}

		if _, err := os.Stat(left.MadeLines); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("forgetting %v: the made lines survived: %v", forget, err)
		}
		if _, err := os.Stat(filepath.Join(data, "recordings", "Hugo", "a.flac")); err != nil {
			t.Errorf("forgetting %v: a recording beside the made lines went: %v", forget, err)
		}
		if _, err := os.Stat(left.State); (err == nil) == forget {
			t.Errorf("forgetting %v: the window's state is there %v", forget, err == nil)
		}
	}
}

// A folder that could not be found arrives empty and is skipped: nothing is removed and nothing is
// refused.
func TestLeftoversThatCouldNotBeFoundAreSkipped(t *testing.T) {
	t.Parallel()
	if err := RemoveLeftovers(Leftovers{}, true); err != nil {
		t.Errorf("removing leftovers with no folders = %v, want nothing", err)
	}
}

// A folder that cannot be deleted does not stop the other being deleted.
func TestAFolderThatCannotGoDoesNotStopTheOther(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	state := filepath.Join(base, "state")
	plant(t, state, "theme", "dark")
	undeletable := base + string(os.PathSeparator) + "."

	err := RemoveLeftovers(Leftovers{MadeLines: undeletable, State: state}, true)

	if err == nil {
		t.Error("a folder that could not be deleted was not refused")
	}
	if _, statErr := os.Stat(state); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("the window's state survived a refusal over the made lines: %v", statErr)
	}
}
