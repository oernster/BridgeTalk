package runlog

import (
	"fmt"
	"io"
)

// Lines logs the lines a run reports that need no answer, such as a made line written without its
// pause (FR-553), each on a line of its own. The run hands it the error output Keep points at the log.
type Lines struct {
	out io.Writer
}

// NewLines logs to out.
func NewLines(out io.Writer) Lines { return Lines{out: out} }

// Log writes line to the output. A line that cannot be written is lost: the log is the one place a
// run could say so.
func (l Lines) Log(line string) { _, _ = fmt.Fprintln(l.out, line) }
