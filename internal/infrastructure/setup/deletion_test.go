package setup

import (
	"strconv"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/product"
)

// awkwardInstallDir holds what breaks a path typed into a script: a space, a dollar sign
// and both kinds of quote.
const awkwardInstallDir = `C:\Users\A $name's "place"\Programs\App`

// awkwardKeep is a folder name carrying the same hazards, so the name is proved to travel as a
// value rather than trusted to be the plain word the product uses.
const awkwardKeep = `my $plugins's "folder"`

// deletionPID is the id of the setup process a delete is told to wait for.
const deletionPID = 4242

// The delete waits for setup's own process to exit before it removes anything (FR-805), whether
// it removes the lot or keeps a folder.
func TestTheDeletionWaitsForSetupBeforeItDeletes(t *testing.T) {
	for _, keep := range []string{"", product.PluginsFolder} {
		args, _ := dirDeletion(deletionPID, awkwardInstallDir, keep)
		script := strings.Join(args, " ")

		wait := strings.Index(script, "Wait-Process -Id "+strconv.Itoa(deletionPID)+" ")
		remove := strings.Index(script, "Remove-Item")
		if wait < 0 || remove < 0 || wait > remove {
			t.Fatalf("keeping %q, the script does not wait for process %d before it deletes: %q", keep, deletionPID, script)
		}
	}
}

// The directory reaches the delete as a value, never as text in the script it runs.
func TestTheDeletionCarriesTheDirectoryAsAValue(t *testing.T) {
	args, env := dirDeletion(deletionPID, awkwardInstallDir, "")

	if script := strings.Join(args, " "); strings.Contains(script, awkwardInstallDir) {
		t.Errorf("the directory is typed into the script: %q", script)
	}
	if want := []string{deletionDirVariable + "=" + awkwardInstallDir}; strings.Join(env, "\n") != strings.Join(want, "\n") {
		t.Errorf("env = %q, want %q", env, want)
	}
}

// A folder to keep reaches the delete as a value too. The delete then leaves the directory
// standing rather than removing it whole (FR-578).
func TestAFolderToKeepIsCarriedAsAValue(t *testing.T) {
	args, env := dirDeletion(deletionPID, awkwardInstallDir, awkwardKeep)
	script := strings.Join(args, " ")

	if strings.Contains(script, awkwardKeep) {
		t.Errorf("the folder to keep is typed into the script: %q", script)
	}
	want := []string{deletionDirVariable + "=" + awkwardInstallDir, deletionKeepVariable + "=" + awkwardKeep}
	if strings.Join(env, "\n") != strings.Join(want, "\n") {
		t.Errorf("env = %q, want %q", env, want)
	}
	if strings.Contains(script, "Remove-Item -LiteralPath") {
		t.Errorf("keeping a folder, the script still removes the directory whole: %q", script)
	}
}
