package plugin

// One rule cannot be reached from outside the package, because Windows will not let a test
// produce it: a folder that exists and cannot be read. Reading a file as a directory
// answers that the path is not there, measured on 2026-09-16, which is the other branch
// entirely. So the reader is handed in here.

import (
	"errors"
	"os"
	"testing"
)

// refusingLister is a directory that exists and will not be read.
func refusingLister(string) ([]os.DirEntry, error) {
	return nil, errors.New("access is denied")
}

func TestAFolderThatCannotBeReadIsNamedWithTheReason(t *testing.T) {
	t.Parallel()

	set := load(`C:\somewhere\plugins`, nil, refusingLister)
	defer set.Close()

	if len(set.Plugins) != 0 {
		t.Fatalf("loaded %+v from a folder that would not be read", set.Plugins)
	}
	if len(set.Refusals) != 1 {
		t.Fatalf("refusals = %+v, want the folder named once", set.Refusals)
	}
	if set.Refusals[0].File != "plugins" {
		t.Errorf("refusal names %q, want the folder", set.Refusals[0].File)
	}
	if set.Refusals[0].Why != "access is denied" {
		t.Errorf("refusal says %q, want what the system said", set.Refusals[0].Why)
	}
}
