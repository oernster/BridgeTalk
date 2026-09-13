//go:build windows

package setup

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// awkwardFolder holds the characters that break a path typed into a PowerShell script: a
// dollar sign read as a variable, a backtick read as an escape and an apostrophe, which ends
// a single-quoted string. Measured: the first two broke every shortcut setup wrote that way.
const awkwardFolder = "Al$ex `n O'Brien"

// iconIndexSuffix is how the shell reads an icon location back: the path, then the index of
// the icon inside the file. Measured on 2026-09-13.
const iconIndexSuffix = ",0"

// shortcutFields is what a saved shortcut reads back as.
type shortcutFields struct{ target, icon, workDir string }

// readShortcut opens a saved shortcut through the same shell object that writes one.
func readShortcut(t *testing.T, path string) shortcutFields {
	t.Helper()
	var fields shortcutFields
	err := withShell(func(shell *ole.IDispatch) error {
		link, err := openShortcut(shell, path)
		if err != nil {
			return err
		}
		defer link.Release()
		into := map[string]*string{
			shortcutTarget: &fields.target, shortcutIcon: &fields.icon, shortcutWorkingDir: &fields.workDir,
		}
		for name, field := range into {
			value, err := oleutil.GetProperty(link, name)
			if err != nil {
				return err
			}
			*field = value.ToString()
			_ = value.Clear()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("reading the shortcut back: %v", err)
	}
	return fields
}

// writeAwkwardShortcut writes a shortcut to a program inside awkwardFolder, answering with
// where it went and what it should read back as.
func writeAwkwardShortcut(t *testing.T) (string, shortcutFields) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), awkwardFolder)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		t.Fatalf("creating %q: %v", dir, err)
	}
	target := filepath.Join(dir, ExeName)
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatalf("writing %q: %v", target, err)
	}
	link := filepath.Join(dir, shortcutName)
	if err := createShortcut(link, target, dir); err != nil {
		t.Fatalf("creating the shortcut: %v", err)
	}
	return link, shortcutFields{target: target, icon: target + iconIndexSuffix, workDir: dir}
}

// FR-802: a shortcut names the program, its icon and its working directory exactly as they
// are spelled, whatever the path holds.
func TestAShortcutKeepsEveryPathExactlyAsGiven(t *testing.T) {
	link, want := writeAwkwardShortcut(t)
	if got := readShortcut(t, link); got != want {
		t.Fatalf("read back %+v, want %+v", got, want)
	}
}

// COM is started for each call and the thread may already have it running: under the same
// model it answers S_FALSE, under the other RPC_E_CHANGED_MODE. A shortcut is written either way.
func TestAShortcutIsWrittenOnAThreadWithCOMAlreadyRunning(t *testing.T) {
	models := map[string]uint32{
		"same model":  ole.COINIT_APARTMENTTHREADED,
		"other model": ole.COINIT_MULTITHREADED,
	}
	for name, model := range models {
		t.Run(name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			if err := ole.CoInitializeEx(0, model); err != nil {
				t.Fatalf("starting COM ahead of the call: %v", err)
			}
			defer ole.CoUninitialize()

			link, want := writeAwkwardShortcut(t)
			if got := readShortcut(t, link); got != want {
				t.Fatalf("read back %+v, want %+v", got, want)
			}
		})
	}
}
