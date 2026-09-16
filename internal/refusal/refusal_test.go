package refusal_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/refusal"
)

// Everything passed over reads the same way, whatever was passed over (FR-567, FR-574).
func TestWhatWasPassedOverIsWordedTheOneWay(t *testing.T) {
	t.Parallel()

	said := refusal.PassedOver("the plugin crew.dll", "it offers no voice")
	if said != "note: the plugin crew.dll was passed over: it offers no voice" {
		t.Errorf("the line reads %q", said)
	}
}

// A failure over one path keeps its reason and loses the path, so the caller's own words
// can name the path once (FR-237). The reason still answers errors.Is as the original did.
func TestAPathFailureKeepsOnlyItsReason(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "Missing Folder")
	_, err := os.Stat(missing)

	reason := refusal.Reason(err)

	if reason == nil || strings.Contains(reason.Error(), missing) {
		t.Fatalf("reason = %v, want the failure without %s", reason, missing)
	}
	if !errors.Is(reason, fs.ErrNotExist) {
		t.Fatalf("reason = %v, want it still read as not existing", reason)
	}
}

// A failure over two paths, as a rename is, loses both.
func TestALinkFailureKeepsOnlyItsReason(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	from := filepath.Join(dir, "From Here")
	to := filepath.Join(dir, "To There")
	err := os.Rename(from, to)

	reason := refusal.Reason(err)

	if reason == nil || strings.Contains(reason.Error(), from) || strings.Contains(reason.Error(), to) {
		t.Fatalf("reason = %v, want the failure without either path", reason)
	}
	if !errors.Is(reason, fs.ErrNotExist) {
		t.Fatalf("reason = %v, want it still read as not existing", reason)
	}
}

// A failed system call is named by the call; the reason is what a reader can act on.
func TestASystemCallFailureKeepsOnlyItsReason(t *testing.T) {
	t.Parallel()
	err := os.NewSyscallError("GetFileAttributesEx", fs.ErrPermission)

	if reason := refusal.Reason(err); reason != fs.ErrPermission {
		t.Fatalf("reason = %v, want the permission failure alone", reason)
	}
}

// An error already wrapped in words of its own keeps them; one naming no path at all has
// nothing to take away.
func TestAnErrorWithWordsOfItsOwnIsLeftAlone(t *testing.T) {
	t.Parallel()
	_, statErr := os.Stat(filepath.Join(t.TempDir(), "Missing Folder"))
	wrapped := fmt.Errorf("reading the journal directory: %w", statErr)
	plain := errors.New("the device is busy")

	for _, err := range []error{wrapped, plain, nil} {
		if reason := refusal.Reason(err); reason != err {
			t.Errorf("reason = %v, want %v unchanged", reason, err)
		}
	}
}

// Check finds each way a refusal can break FR-237 and nothing in one that reads as it
// should. The path is written literally so the doubled separators are tested on every
// platform rather than only on the one whose paths carry backslashes.
func TestCheckFindsEachWayARefusalGoesWrong(t *testing.T) {
	t.Parallel()
	const path = `C:\Recordings\Oliver`
	cases := map[string]struct {
		refused  error
		problems int
	}{
		"named once, in plain words": {
			errors.New(`reading C:\Recordings\Oliver: The system cannot find the file specified.`), 0,
		},
		"named twice": {
			errors.New(`reading C:\Recordings\Oliver: no journal files in C:\Recordings\Oliver`), 1,
		},
		"separators doubled": {
			errors.New(`no journal files in "C:\\Recordings\\Oliver"`), 2,
		},
		"system call before it": {
			errors.New(`GetFileAttributesEx C:\Recordings\Oliver: The system cannot find the file specified.`), 1,
		},
		"system call before a folder on its way": {
			errors.New(`creating C:\Recordings\Oliver: mkdir C:\Recordings: The system cannot find the path specified.`), 1,
		},
		"nothing refused": {nil, 1},
	}
	for name, each := range cases {
		if got := refusal.Check(each.refused, path); len(got) != each.problems {
			t.Errorf("%s: found %v, want %d problems", name, got, each.problems)
		}
	}
}

// A path with no separator in it is its own root, so a system call before it is still found.
func TestCheckReadsAPathWithNoSeparator(t *testing.T) {
	t.Parallel()
	const path = "Recordings"

	if got := refusal.Check(errors.New("no voices in Recordings"), path); len(got) != 0 {
		t.Errorf("a plain refusal found %v, want nothing", got)
	}
	if got := refusal.Check(errors.New("open Recordings: not found"), path); len(got) != 1 {
		t.Errorf("a system call before it found %v, want one problem", got)
	}
}
