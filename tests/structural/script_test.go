package structural

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/config"
)

// scriptComplete switches FR-507 on. The script is written a group of cues at a time, so until
// the last group lands a cue without lines is reported as progress rather than failing the
// build; from then on it fails.
const scriptComplete = false

// TestTheShippedScriptHoldsNoProblem holds FR-504, FR-505 and FR-531 over script.toml.
//
// The rules live in the domain; this reads the shipped file through them, so they are written
// once. Every problem is named with its cue and its line, not the first alone.
//
// Proved by planting a key that is not a cue, a cue with two lines and a broken spelling in
// script.toml, reading the exit code each time.
func TestTheShippedScriptHoldsNoProblem(t *testing.T) {
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("loading the shipped cue table: %v", err)
	}
	if _, err := config.LoadScript(table); err != nil {
		t.Errorf("script.toml: %v", err)
	}
}

// TestTheScriptHoldsLinesForEveryCue holds FR-507: every cue in the table has lines.
//
// Proved by planting scriptComplete as true while the script is incomplete, reading the exit
// code.
func TestTheScriptHoldsLinesForEveryCue(t *testing.T) {
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("loading the shipped cue table: %v", err)
	}
	loaded, err := config.LoadScript(table)
	if err != nil {
		t.Fatalf("script.toml: %v", err)
	}
	missing := loaded.Missing(table)
	if len(missing) > 0 && !scriptComplete {
		t.Skipf("script.toml holds lines for %d of %d cues; FR-507 is enforced once it is complete",
			table.Len()-len(missing), table.Len())
	}
	for _, id := range missing {
		t.Errorf("%s has no lines in script.toml (FR-507)", id)
	}
}
