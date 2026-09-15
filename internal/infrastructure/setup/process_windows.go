//go:build windows

package setup

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// ErrAppRunning says the application is open, so an install or an uninstall that
// would overwrite or delete the running executable must not start.
//
//lint:ignore ST1005 setup dialog text shown to the user verbatim
var ErrAppRunning = errors.New(product.Name + " is running. Please close it and try again.")

// ErrAppStillRunning says the application was asked to close but was still there
// after the wait, so setup stops rather than writing over a locked file.
//
//lint:ignore ST1005 setup dialog text shown to the user verbatim
var ErrAppStillRunning = errors.New(product.Name + " could not be closed. Please close it manually and try again.")

const (
	// closeTimeout bounds the wait for the executable lock to release, so a stuck
	// process cannot hang setup for ever.
	closeTimeout = 5 * time.Second
	// closePollStep is how often the wait rechecks.
	closePollStep = 100 * time.Millisecond
	// forcedExitCode is reported for a process ended by setup.
	forcedExitCode = 1
)

// IsAppRunning reports whether the application is currently running.
func IsAppRunning() bool { return len(processIDs(ExeName)) > 0 }

// processIDs returns the ids of every running process with the given executable
// name, compared without regard to case. A failure taking or walking the snapshot
// yields no ids, so a snapshot error never blocks a legitimate install.
func processIDs(exeName string) []uint32 {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil
	}
	target := strings.ToLower(exeName)
	var found []uint32
	for {
		if strings.ToLower(windows.UTF16ToString(entry.ExeFile[:])) == target {
			found = append(found, entry.ProcessID)
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			return found
		}
	}
}

// CloseRunningApp ends every running instance and waits for the executable lock to
// release. Termination is forced rather than a polite window close, because the
// application minimises to the tray instead of exiting when its window is closed, so
// only ending the process reliably frees the file.
func CloseRunningApp() error {
	for _, pid := range processIDs(ExeName) {
		terminate(pid)
	}
	deadline := time.Now().Add(closeTimeout)
	for IsAppRunning() {
		if time.Now().After(deadline) {
			return ErrAppStillRunning
		}
		time.Sleep(closePollStep)
	}
	return nil
}

// terminate forcibly ends one process. Every failure is ignored, because a process
// that has already exited needs no further action and one this user cannot open is
// not this user's to end.
func terminate(pid uint32) {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return
	}
	defer windows.CloseHandle(handle)
	_ = windows.TerminateProcess(handle, forcedExitCode)
}

// LaunchApp starts the installed application detached, with the install directory as
// its working directory, so it outlives the setup program rather than ending with it.
func LaunchApp() error {
	dir, err := InstallDir()
	if err != nil {
		return err
	}
	cmd := exec.Command(filepath.Join(dir, ExeName))
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", AppName, refusal.Reason(err))
	}
	return cmd.Process.Release()
}

// ScheduleDirDeletion starts a hidden process that waits for this one to exit, so setup
// has released its own copy of the executable, then removes the install directory
// (FR-805). Setup lives inside the directory it is deleting, which is why the removal
// has to outlive it.
//
// It once paused two seconds and ran rmdir through cmd with the path in quotes. Go
// escapes those quotes on the command line in a form cmd does not read, so rmdir refused
// the path as bad syntax and nothing was ever deleted; measured on 2026-09-15. The pause
// was never tied to setup closing either.
func ScheduleDirDeletion(dir string) { scheduleDirDeletionAfter(os.Getpid(), dir) }

// scheduleDirDeletionAfter starts the delete waiting on the process pid, so a test can
// play setup with a process it is able to close.
func scheduleDirDeletionAfter(pid int, dir string) {
	args, env := dirDeletion(pid, dir)
	cmd := exec.Command(deletionShell, args...)
	cmd.Dir = deletionWorkDir(dir)
	cmd.Env = append(os.Environ(), env)
	cmd.SysProcAttr = hidden()
	if err := cmd.Start(); err != nil {
		return
	}
	_ = cmd.Process.Release()
}
