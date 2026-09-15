// Package runlog keeps the log a run leaves (FR-715): a line naming when the run started, then
// what the run reports as it fails, in the file Log.txt inside the product's local data folder.
//
// Go's own crash file, runtime/debug.SetCrashOutput, carries a panic's report whole but leaves out
// the first line of a fatal error's, since the runtime prints that line before it copies anything to
// the file (runtime.throw, read in Go 1.26.3 on 2026-09-14). Pointing the error output itself at the
// log carries every line. That is done wherever the run has no error output to lose, which is how a
// windowed program started from a shortcut runs; a run with one keeps it and adds the crash file.
//
// It also finds the terminal a windowed run was started from, for the reports the command line asks
// for (FR-703): both decide where what a run writes goes.
package runlog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/oernster/bridge-talk/internal/infrastructure/appdata"
	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

const (
	// FileName names the log inside the product's local data folder.
	FileName = "Log.txt"
	// MaxBytes is the size past which a run starts the log afresh: 1 MB (Oliver, 2026-09-14). A crash
	// report measured 0.4 to 24 KB, so the limit only stops the file growing without end.
	MaxBytes = 1 << 20
)

const (
	// startedLayout writes the time a run started, to the second.
	startedLayout = "2006-01-02 15:04:05"
	folderPerm    = 0o755
	filePerm      = 0o644
)

// Path answers with the log inside the product's local data folder, without touching the disk.
func Path() (string, error) {
	base, err := appdata.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, FileName), nil
}

// Open opens the log at path for a run started at started, making its folder where there is none,
// then adds the run's start line. What the log holds is kept unless it is over MaxBytes, when the
// log is started afresh.
func Open(path string, started time.Time) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), folderPerm); err != nil {
		return nil, fmt.Errorf("making the folder for %s: %w", path, refusal.Reason(err))
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if info, err := os.Stat(path); err == nil && info.Size() > MaxBytes {
		flags |= os.O_TRUNC
	}
	log, err := os.OpenFile(path, flags, filePerm)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, refusal.Reason(err))
	}
	if _, err := fmt.Fprintf(log, "%s started %s\n", product.Name, started.Format(startedLayout)); err != nil {
		_ = log.Close()
		return nil, fmt.Errorf("writing to %s: %w", path, refusal.Reason(err))
	}
	return log, nil
}

// Keep sends what the run reports as it fails to log, which stays open for the rest of the run.
// Where the run has no error output, all of it goes to the log; otherwise it stays where it is and
// the crash report is copied to the log as well.
func Keep(log *os.File) error {
	if !hasErrorOutput() {
		return sendAll(log)
	}
	if err := debug.SetCrashOutput(log, debug.CrashOptions{}); err != nil {
		return fmt.Errorf("copying crash reports to %s: %w", log.Name(), err)
	}
	return nil
}

// ReportToTerminal sends what the run prints to the terminal it was started from, where it was given
// no standard output of its own: a windowed program started from a terminal is given none (FR-703). A
// run given one keeps it.
func ReportToTerminal() error { return toTerminal() }
