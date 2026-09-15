package measured_test

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/measured"
)

// A line is named by its voice, its cue quoted and its place as the script counts, from one.
func TestALineIsNamedByVoiceCueAndItsPlaceCountedFromOne(t *testing.T) {
	for index, want := range []string{`bf_emma "Docked" line 1`, `bf_emma "Docked" line 2`} {
		if got := measured.LineName("bf_emma", "Docked", index); got != want {
			t.Errorf("LineName at index %d = %q, want %q", index, got, want)
		}
	}
}
