//go:build windows

package setup

// processIDs is measured against the one process a test can be certain is running:
// the test binary itself. Nothing is started or ended, so the tests change nothing on
// the machine and give the same answer whatever else it happens to be running.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The snapshot finds the running test by its own executable name. The name is asked
// for in upper case, so the match is proved to ignore case the way Windows does.
func TestProcessIDsFindsTheRunningTestByItsOwnName(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("finding the test binary: %v", err)
	}
	name := strings.ToUpper(filepath.Base(executable))

	found := processIDs(name)

	if own := uint32(os.Getpid()); !slices.Contains(found, own) {
		t.Fatalf("processIDs(%q) = %v, want it to include this process, %d", name, found, own)
	}
}

// A name no process carries yields no ids, which is what lets an install go ahead.
func TestProcessIDsFindsNothingForANameNoProcessCarries(t *testing.T) {
	if found := processIDs("no-process-is-named-this.exe"); len(found) != 0 {
		t.Fatalf("found %v for a name nothing carries, want none", found)
	}
}
