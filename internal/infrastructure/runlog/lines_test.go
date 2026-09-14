package runlog

// FR-553 and FR-715: a line the run logs reaches the output handed in, one line each.

import (
	"strings"
	"testing"
)

func TestEachLineLoggedIsWrittenOnALineOfItsOwn(t *testing.T) {
	var out strings.Builder
	lines := NewLines(&out)

	lines.Log(`bf_emma "Docked" line 1 was written without its pause`)
	lines.Log("second")

	if got, want := out.String(), "bf_emma \"Docked\" line 1 was written without its pause\nsecond\n"; got != want {
		t.Errorf("wrote %q, want %q", got, want)
	}
}
