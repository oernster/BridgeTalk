//go:build windows

package setup

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

// shellProgramID names the Windows shell automation object that reads and writes shortcuts.
const shellProgramID = "WScript.Shell"

// The shortcut properties setup writes and a test reads back.
const (
	shortcutTarget     = "TargetPath"
	shortcutIcon       = "IconLocation"
	shortcutWorkingDir = "WorkingDirectory"
)

// withShell runs fn against the shell automation object with COM started, then releases
// everything it took. COM belongs to a thread, so the goroutine stays on one for the call.
func withShell(fn func(shell *ole.IDispatch) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	stop, err := startCOM()
	if err != nil {
		return fmt.Errorf("starting COM: %w", err)
	}
	defer stop()

	unknown, err := oleutil.CreateObject(shellProgramID)
	if err != nil {
		return fmt.Errorf("creating %s: %w", shellProgramID, err)
	}
	defer unknown.Release()
	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("reaching %s: %w", shellProgramID, err)
	}
	defer shell.Release()
	return fn(shell)
}

// startCOM starts COM on this thread, answering with what undoes it.
//
// A thread with COM already running under the same model answers S_FALSE, which still has to
// be balanced by an uninitialise. One running under the other model answers
// RPC_E_CHANGED_MODE: COM is usable there but was not started here, so it is not stopped here.
func startCOM() (func(), error) {
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	if err == nil {
		return ole.CoUninitialize, nil
	}
	var failure *ole.OleError
	if !errors.As(err, &failure) {
		return nil, err
	}
	switch windows.Handle(failure.Code()) {
	case windows.S_FALSE:
		return ole.CoUninitialize, nil
	case windows.RPC_E_CHANGED_MODE:
		return func() {}, nil
	}
	return nil, err
}

// openShortcut answers with the shortcut object for path: the one on disk when there is one,
// a new one to fill in otherwise. The caller releases it.
func openShortcut(shell *ole.IDispatch, path string) (*ole.IDispatch, error) {
	result, err := oleutil.CallMethod(shell, "CreateShortcut", path)
	if err != nil {
		return nil, fmt.Errorf("opening the shortcut %s: %w", path, err)
	}
	return result.ToIDispatch(), nil
}

// createShortcut writes a .lnk through the shell automation object, handing each path over
// as a value. Nothing is typed into a script, so no character in a path can be read as
// anything but itself; a dollar sign once was, as a PowerShell variable.
func createShortcut(linkPath, target, workDir string) error {
	return withShell(func(shell *ole.IDispatch) error {
		link, err := openShortcut(shell, linkPath)
		if err != nil {
			return err
		}
		defer link.Release()
		properties := []struct{ name, value string }{
			{shortcutTarget, target}, {shortcutIcon, target}, {shortcutWorkingDir, workDir},
		}
		for _, property := range properties {
			if _, err := oleutil.PutProperty(link, property.name, property.value); err != nil {
				return fmt.Errorf("setting %s on the shortcut %s: %w", property.name, linkPath, err)
			}
		}
		if _, err := oleutil.CallMethod(link, "Save"); err != nil {
			return fmt.Errorf("saving the shortcut %s: %w", linkPath, err)
		}
		return nil
	})
}
