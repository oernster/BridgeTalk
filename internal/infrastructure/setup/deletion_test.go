package setup

import (
	"strconv"
	"strings"
	"testing"
)

// awkwardInstallDir holds what breaks a path typed into a script: a space, a dollar sign
// and both kinds of quote.
const awkwardInstallDir = `C:\Users\A $name's "place"\Programs\App`

// deletionPID is the id of the setup process a delete is told to wait for.
const deletionPID = 4242

// The delete waits for setup's own process to exit before it removes anything (FR-805).
func TestTheDeletionWaitsForSetupBeforeItDeletes(t *testing.T) {
	args, _ := dirDeletion(deletionPID, awkwardInstallDir)
	script := strings.Join(args, " ")

	wait := strings.Index(script, "Wait-Process -Id "+strconv.Itoa(deletionPID)+" ")
	remove := strings.Index(script, "Remove-Item")
	if wait < 0 || remove < 0 || wait > remove {
		t.Fatalf("the script does not wait for process %d before it deletes: %q", deletionPID, script)
	}
}

// The directory reaches the delete as a value, never as text in the script it runs.
func TestTheDeletionCarriesTheDirectoryAsAValue(t *testing.T) {
	args, env := dirDeletion(deletionPID, awkwardInstallDir)

	if script := strings.Join(args, " "); strings.Contains(script, awkwardInstallDir) {
		t.Errorf("the directory is typed into the script: %q", script)
	}
	if want := deletionDirVariable + "=" + awkwardInstallDir; env != want {
		t.Errorf("env = %q, want %q", env, want)
	}
}
