//go:build windows

package setup

// The install directory is deleted by a process that outlives setup (FR-805). These tests
// start that process for real against a temporary directory, so what is measured is the
// delete itself: nothing here reaches the install directory or the registry. The only
// programs started are the delete and a copy of the test binary playing setup.

import (
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

// standInVariable turns the test binary into a stand-in for setup: a process that runs
// until its input is closed, which is how a test says setup has closed.
const standInVariable = "SETUP_STAND_IN"

// deletionDeadline bounds the wait for the directory to go once the stand-in has closed.
// It is generous because the process doing the delete starts cold.
const deletionDeadline = 20 * time.Second

// TestStandInForSetup is the stand-in's body. Run by the suite, it does nothing.
func TestStandInForSetup(t *testing.T) {
	if os.Getenv(standInVariable) == "" {
		return
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
}

// startStandIn starts the stand-in, answering with what closes it.
func startStandIn(t *testing.T) (pid int, closeIt func()) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("finding the test binary: %v", err)
	}
	cmd := exec.Command(executable, "-test.run=^TestStandInForSetup$")
	cmd.Env = append(os.Environ(), standInVariable+"=1")
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("piping to the stand-in: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting the stand-in: %v", err)
	}
	closeIt = func() {
		_ = input.Close()
		_ = cmd.Wait()
	}
	t.Cleanup(closeIt)
	return cmd.Process.Pid, closeIt
}

// gone reports whether dir has been deleted before the deadline passes.
func gone(dir string, deadline time.Duration) bool {
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return true
		}
		time.Sleep(closePollStep)
	}
	return false
}

// The directory is deleted once setup has closed. Setup is played by a stand-in process,
// so the test can close it and still be there to look.
func TestTheInstallDirectoryGoesOnceSetupHasClosed(t *testing.T) {
	dir := standingDir(t)
	pid, closeSetup := startStandIn(t)

	scheduleDirDeletionAfter(pid, dir)
	closeSetup()

	if !gone(dir, deletionDeadline) {
		t.Fatalf("the directory was still there %v after setup closed", deletionDeadline)
	}
}

// A process standing in a directory holds it, so a delete that inherited a working directory
// inside the install directory would empty it and leave it standing. Setup started from inside
// its own directory must still see that directory go once it has closed.
func TestTheInstallDirectoryGoesWhenSetupWasStartedInsideIt(t *testing.T) {
	dir := standingDir(t)
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("reading the working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("entering %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	pid, closeSetup := startStandIn(t)

	scheduleDirDeletionAfter(pid, dir)
	closeSetup()
	_ = os.Chdir(previous)

	if !gone(dir, deletionDeadline) {
		t.Fatalf("the directory was still there %v after setup closed, with setup started inside it", deletionDeadline)
	}
}

// pastTheOldWait is longer than the fixed pause the delete once took before removing the
// directory whether or not setup had closed, so a directory still standing after it was
// kept for a reason other than the clock.
const pastTheOldWait = 4 * time.Second

// standingFile is written into each directory so an empty one cannot pass for a kept one.
const standingFile = "standing.txt"

// standingPrefix names each directory a delete is aimed at, so a leftover one is recognisable.
const standingPrefix = "bridgetalk-setup-delete-"

// standingDir makes a directory with a file in it for a delete to aim at. It is made straight
// in the system temporary directory rather than by t.TempDir: the delete starts beside its
// directory, so one still waiting on this test binary would hold t.TempDir's own parent past
// the test's cleanup.
func standingDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", standingPrefix)
	if err != nil {
		t.Fatalf("making a directory to delete: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if err := os.WriteFile(dir+string(os.PathSeparator)+standingFile, []byte("kept"), dirPerm); err != nil {
		t.Fatalf("writing into %s: %v", dir, err)
	}
	return dir
}

// The directory is not deleted while setup is still running. The running setup here is
// the test binary itself, which is the process ScheduleDirDeletion is called from.
func TestTheInstallDirectoryOutlivesTheRunningSetup(t *testing.T) {
	dir := standingDir(t)

	ScheduleDirDeletion(dir)
	time.Sleep(pastTheOldWait)

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the directory went while setup was still running: %v", err)
	}
}
