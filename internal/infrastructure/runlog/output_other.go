//go:build !windows

package runlog

import (
	"errors"
	"os"
)

// hasErrorOutput answers yes off Windows, so Keep copies the crash report to the log and leaves the
// error output where it is: finding a run without one is not built for this platform yet.
func hasErrorOutput() bool { return true }

// sendAll is not built for this platform yet, so it says so rather than doing nothing.
func sendAll(*os.File) error {
	return errors.New("sending error output to a file is not built for this platform yet")
}
