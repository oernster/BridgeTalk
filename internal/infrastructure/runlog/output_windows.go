//go:build windows

package runlog

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// attachParentProcess asks AttachConsole for the console of the program that started the run. It is
// the value Windows names ATTACH_PARENT_PROCESS, which golang.org/x/sys/windows does not name.
const attachParentProcess = ^uint32(0)

// consoleOutput names the screen buffer of the console a program is attached to.
const consoleOutput = "CONOUT$"

// attachConsole is AttachConsole in kernel32, which golang.org/x/sys/windows does not wrap.
var attachConsole = windows.NewLazySystemDLL("kernel32.dll").NewProc("AttachConsole")

// hasErrorOutput reports whether the run was given an error output. A windowed program started
// without one reads a handle of 0 (measured 2026-09-14), so everything it writes there is lost.
func hasErrorOutput() bool { return given(windows.STD_ERROR_HANDLE) }

// given reports whether the run was given the standard handle named.
func given(which uint32) bool {
	handle, err := windows.GetStdHandle(which)
	return err == nil && handle != 0 && handle != windows.InvalidHandle
}

// sendAll points the run's error output at log: first the handle the Go runtime looks up for each
// report it writes (runtime.write1, read in Go 1.26.3 on 2026-09-14), then os.Stderr, which was
// fixed from that handle when the program started.
func sendAll(log *os.File) error {
	if err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(log.Fd())); err != nil {
		return fmt.Errorf("sending error output to %s: %w", log.Name(), err)
	}
	os.Stderr = log
	return nil
}

// toTerminal points standard output and error output at the console of the program that started
// the run, where the run was given no standard output. A windowed program started from PowerShell
// reads a handle of 0 for both, so a line it prints is lost; attached to that console, the line
// shows there (measured 2026-09-15).
func toTerminal() error {
	if given(windows.STD_OUTPUT_HANDLE) {
		return nil
	}
	if attached, _, err := attachConsole.Call(uintptr(attachParentProcess)); attached == 0 {
		return fmt.Errorf("finding the terminal the run was started from: %w", err)
	}
	console, err := os.OpenFile(consoleOutput, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("opening the terminal: %w", err)
	}
	for _, which := range []uint32{windows.STD_OUTPUT_HANDLE, windows.STD_ERROR_HANDLE} {
		if err := windows.SetStdHandle(which, windows.Handle(console.Fd())); err != nil {
			return fmt.Errorf("sending output to the terminal: %w", err)
		}
	}
	os.Stdout, os.Stderr = console, console
	return nil
}
