package structural

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// setupReset matches the fixed choices a reinstall once applied in place of the boxes on
// screen; liveCall matches a box saving at once and liveCaught the same call with its
// failure handled.
var (
	setupReset = regexp.MustCompile(`freshChoices`)
	liveCall   = regexp.MustCompile(`backend\(\)\.Set\w+\(`)
	liveCaught = regexp.MustCompile(`backend\(\)\.Set\w+\([^\n]*\)\.catch\(`)
)

// TestSetupAppliesTheBoxesItShows holds FR-235 on the setup page.
//
// Reinstall once applied sign-in off beneath a ticked box, so the application opened with
// its own box unticked. A box that saved at once also dropped any failure, which would
// leave it ticked over nothing. The page has no test runner, so the rule is held here by
// reading its scripts.
//
// Proved by restoring the fixed reinstall choices, then separately removing the failure
// handling from the sign-in box, reading the exit code each time.
func TestSetupAppliesTheBoxesItShows(t *testing.T) {
	root := repoRoot(t)
	for _, name := range setupScripts {
		raw, err := os.ReadFile(filepath.Join(root, setupFrontendDir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if setupReset.Match(raw) {
			t.Errorf("%s applies fixed choices in place of the boxes on screen", name)
		}
		calls, caught := len(liveCall.FindAll(raw, -1)), len(liveCaught.FindAll(raw, -1))
		if calls != caught {
			t.Errorf("%s saves %d boxes at once but handles the failure of only %d", name, calls, caught)
		}
	}
}
