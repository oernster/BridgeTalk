package main

import (
	"fmt"
	"os"
	"time"

	"github.com/oernster/bridge-talk/internal/infrastructure/runlog"
)

// keepLog starts this run's log before anything else can fail (FR-715), so a refusal to start and a
// crash both reach it however the run was started. The log stays open until the process ends.
//
// A log that cannot be kept stops nothing: the run starts with a warning, which reaches whatever
// error output the run has.
func keepLog() {
	path, err := runlog.Path()
	if err == nil {
		var log *os.File
		if log, err = runlog.Open(path, time.Now()); err == nil {
			err = runlog.Keep(log)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v (keeping no log)\n", err)
	}
}

// runLog answers what logs the lines a run reports that need no answer (FR-553): the error output
// keepLog pointed at the log, so it is asked for after keepLog.
func runLog() runlog.Lines { return runlog.NewLines(os.Stderr) }
