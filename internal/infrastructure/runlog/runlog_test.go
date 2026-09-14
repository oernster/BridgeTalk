package runlog

// FR-715: a run leaves a log. A crash ends the process that has it, so the crash tests start this
// test binary again as a child; TestMain turns the child into the crash asked for.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

const (
	// childEnv names what the child does; logEnv names the log it keeps.
	childEnv = "BRIDGE_TALK_RUNLOG_CHILD"
	logEnv   = "BRIDGE_TALK_RUNLOG_PATH"

	// panicAct panics on another goroutine; fatalAct sends all error output to the log, writes a
	// warning to it, then fails with a fatal error rather than a panic.
	panicAct = "panic"
	fatalAct = "fatal"

	plantedPanic   = "a planted panic on another goroutine"
	plantedWarning = "warning: a planted warning"
	// fatalHeadline is the line the Go runtime starts a fatal error's report with; SetCrashOutput
	// alone leaves it out of the file (measured 2026-09-14).
	fatalHeadline = "fatal error: sync: unlock of unlocked mutex"

	// goCrashExitCode is the exit code the Go runtime ends a crashed program with.
	goCrashExitCode = 2
	// childFailedExitCode and childSurvivedExitCode end a child that never reached its crash.
	childFailedExitCode   = 3
	childSurvivedExitCode = 4
	// childWait is how long a child waits for its crash before saying it survived.
	childWait = 10 * time.Second
)

// started is the time every test's run starts at.
var started = time.Date(2026, time.September, 14, 11, 18, 31, 0, time.Local)

func TestMain(m *testing.M) {
	if act := os.Getenv(childEnv); act != "" {
		crash(act, os.Getenv(logEnv))
		return
	}
	os.Exit(m.Run())
}

// crash is the child: it keeps the log as the application does, then fails as act asks.
func crash(act, path string) {
	log, err := Open(path, started)
	if err == nil {
		if act == fatalAct {
			err = sendAll(log)
		} else {
			err = Keep(log)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "the child could not keep its log: %v\n", err)
		os.Exit(childFailedExitCode)
	}
	switch act {
	case panicAct:
		go func() { panic(plantedPanic) }()
	case fatalAct:
		fmt.Fprintln(os.Stderr, plantedWarning)
		var unlocked sync.Mutex
		unlocked.Unlock()
	}
	time.Sleep(childWait)
	os.Exit(childSurvivedExitCode)
}

// runChild starts the child with an error output of its own and answers with its log and what it
// wrote to that output.
func runChild(t *testing.T, act string) (logged, errorOutput string) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("finding the test binary: %v", err)
	}
	path := filepath.Join(t.TempDir(), FileName)
	child := exec.Command(executable, "-test.run=^$")
	child.Env = append(os.Environ(), childEnv+"="+act, logEnv+"="+path)
	var said bytes.Buffer
	child.Stderr = &said

	ran := child.Run()

	var exit *exec.ExitError
	if !errors.As(ran, &exit) || exit.ExitCode() != goCrashExitCode {
		t.Fatalf("the child ended with %v, want a crash; it said %q", ran, said.String())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the child's log: %v", err)
	}
	return string(raw), said.String()
}

// startLine is the line a run started at when adds to the log.
func startLine(when time.Time) string {
	return product.Name + " started " + when.Format("2006-01-02 15:04:05") + "\n"
}

func TestAPanicOnAnotherGoroutineIsInTheLogAndStaysOnTheErrorOutput(t *testing.T) {
	t.Parallel()
	logged, errorOutput := runChild(t, panicAct)

	if !strings.HasPrefix(logged, startLine(started)) {
		t.Errorf("the log does not open with the run's start line: %q", logged)
	}
	if want := "panic: " + plantedPanic; !strings.Contains(logged, want) {
		t.Errorf("the log lacks %q: %q", want, logged)
	}
	if want := "panic: " + plantedPanic; !strings.Contains(errorOutput, want) {
		t.Errorf("the error output lacks %q: %q", want, errorOutput)
	}
}

// plantLog writes content to a log file under a new folder, answering with its path.
func plantLog(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, content, filePerm); err != nil {
		t.Fatalf("planting %s: %v", path, err)
	}
	return path
}

// openAndClose opens the log for a run started at when, then closes it.
func openAndClose(t *testing.T, path string, when time.Time) {
	t.Helper()
	log, err := Open(path, when)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	if err := log.Close(); err != nil {
		t.Fatalf("closing %s: %v", path, err)
	}
}

func TestEveryRunAddsItsStartLineAfterWhatTheLogHolds(t *testing.T) {
	t.Parallel()
	earlier := "an earlier report\n"
	path := plantLog(t, []byte(earlier))
	later := started.Add(time.Hour)

	openAndClose(t, path, started)
	openAndClose(t, path, later)

	raw, err := os.ReadFile(path)
	if want := earlier + startLine(started) + startLine(later); err != nil || string(raw) != want {
		t.Errorf("the log holds %q, %v; want %q", raw, err, want)
	}
}

func TestALogOverTheLimitIsStartedAfresh(t *testing.T) {
	t.Parallel()
	cases := []struct {
		size int
		kept bool
	}{
		{MaxBytes, true},
		{MaxBytes + 1, false},
	}
	for _, each := range cases {
		planted := bytes.Repeat([]byte("x"), each.size)
		path := plantLog(t, planted)

		openAndClose(t, path, started)

		raw, err := os.ReadFile(path)
		want := startLine(started)
		if each.kept {
			want = string(planted) + want
		}
		if err != nil || string(raw) != want {
			t.Errorf("a log of %d bytes: holds %d bytes, %v; want %d", each.size, len(raw), err, len(want))
		}
	}
}

func TestTheLogIsMadeWithItsFolder(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "Local", product.Slug, FileName)

	openAndClose(t, path, started)

	if raw, err := os.ReadFile(path); err != nil || string(raw) != startLine(started) {
		t.Errorf("the log holds %q, %v; want the start line", raw, err)
	}
}

// FR-237: a log that cannot be kept is refused naming it once.
func TestALogThatCannotBeKeptIsRefusedNamingItOnce(t *testing.T) {
	t.Parallel()
	plainFile := plantLog(t, []byte("x"))
	underAFile := filepath.Join(plainFile, "Local", FileName)
	aFolder := t.TempDir()

	for _, path := range []string{underAFile, aFolder} {
		log, err := Open(path, started)
		if log != nil {
			_ = log.Close()
		}
		for _, problem := range refusal.Check(err, path) {
			t.Errorf("opening %s: %s", path, problem)
		}
	}
}
