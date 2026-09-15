package setup

// Where an install goes (FR-809). Every folder here is built under t.TempDir(); nothing is
// installed and no registry value is read.

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/refusal"
)

// A location an install recorded is where every later run acts, so it wins over the folder
// setup offers; with nothing recorded, the offered folder answers.
func TestARecordedInstallLocationWinsOverTheOfferedFolder(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	recorded := filepath.Join(base, "Games", InstallFolder)

	if got, err := installDirFrom(recorded); err != nil || got != recorded {
		t.Fatalf("with a recorded location: %q, %v; want %q", got, err, recorded)
	}
	offered := filepath.Join(base, installSubdir, InstallFolder)
	if got, err := installDirFrom(""); err != nil || got != offered {
		t.Fatalf("with nothing recorded: %q, %v; want %q", got, err, offered)
	}
}

// Uninstall deletes the install folder whole, so a picked folder is never installed into
// directly: the install gets a folder of its own inside it, unless the pick already is one.
func TestAnInstallGoesIntoAFolderOfItsOwnInsideThePickedOne(t *testing.T) {
	t.Parallel()
	picked := filepath.Join(t.TempDir(), "Games")

	if got, want := InstallDirWithin(picked), filepath.Join(picked, InstallFolder); got != want {
		t.Errorf("picking %q installs into %q, want %q", picked, got, want)
	}
	own := filepath.Join(picked, strings.ToLower(InstallFolder))
	if got := InstallDirWithin(own); got != own {
		t.Errorf("picking the install folder itself installs into %q, want %q", got, own)
	}
}

// A folder not made yet is taken; checking it makes nothing: the probe goes into the
// nearest folder that exists and leaves it as it found it.
func TestAnInstallFolderThatIsNotThereYetIsTakenWithoutBeingMade(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	dir := filepath.Join(base, "Games", InstallFolder)

	if err := CheckInstallDir(dir); err != nil {
		t.Fatalf("a folder under a writable one was refused: %v", err)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("reading the probed folder: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("checking left %d entries behind in the probed folder", len(entries))
	}
}

// An empty folder holds nothing uninstall could take with it; nor does a folder the
// application is already in, which is how a folder left by an earlier install reads.
func TestAnEmptyInstallFolderAndOneHoldingTheApplicationAreTaken(t *testing.T) {
	t.Parallel()
	empty := filepath.Join(t.TempDir(), InstallFolder)
	if err := os.Mkdir(empty, dirPerm); err != nil {
		t.Fatalf("making the empty folder: %v", err)
	}
	if err := CheckInstallDir(empty); err != nil {
		t.Errorf("an empty folder was refused: %v", err)
	}

	holding := filepath.Join(t.TempDir(), InstallFolder)
	if err := os.MkdirAll(filepath.Join(holding, "models"), dirPerm); err != nil {
		t.Fatalf("making the installed folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(holding, ExeName), []byte("the program"), 0o644); err != nil {
		t.Fatalf("planting the program: %v", err)
	}
	if err := CheckInstallDir(holding); err != nil {
		t.Errorf("a folder holding the application was refused: %v", err)
	}
}

// Picking the folder above the product's own data folder would make that data folder the
// install folder, which holds the recordings. Anything already in a folder is refused unless
// the application is among it.
func TestAnInstallFolderHoldingOtherFilesIsRefused(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), InstallFolder)
	if err := os.MkdirAll(filepath.Join(dir, "Recordings"), dirPerm); err != nil {
		t.Fatalf("planting the recordings: %v", err)
	}

	err := CheckInstallDir(dir)
	for _, problem := range refusal.Check(err, dir) {
		t.Error(problem)
	}
}

// The install folder is always named for the product, a relative path names nowhere in
// particular and a plain file cannot hold a folder: each is refused naming its path once.
func TestAnInstallFolderThatCannotBeUsedIsRefusedNamingIt(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	plain := filepath.Join(base, "in the way")
	if err := os.WriteFile(plain, []byte("a file, not a folder"), 0o644); err != nil {
		t.Fatalf("planting the blocker: %v", err)
	}

	for name, each := range map[string]struct{ dir, named string }{
		"not named for the product": {filepath.Join(base, "Games"), filepath.Join(base, "Games")},
		"relative":                  {InstallFolder, InstallFolder},
		"beneath a file":            {filepath.Join(plain, InstallFolder), plain},
	} {
		err := CheckInstallDir(each.dir)
		for _, problem := range refusal.Check(err, each.named) {
			t.Errorf("%s: %s", name, problem)
		}
	}
}

// Setup never asks for administrator rights, so a folder only an administrator may write to
// would fail part way through the install. It is refused before anything is written, naming
// the folder the install would first write into.
func TestAnInstallFolderThisAccountCannotWriteToIsRefused(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	dir := filepath.Join(base, InstallFolder)
	denied := func(folder string) error {
		return &fs.PathError{Op: "open", Path: filepath.Join(folder, "probe"), Err: errors.New("Access is denied.")}
	}

	err := checkInstallDir(dir, denied)
	for _, problem := range refusal.Check(err, base) {
		t.Error(problem)
	}
}
