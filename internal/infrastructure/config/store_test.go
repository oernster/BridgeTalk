package config

// These tests are inside the package for the same reason settings_test.go is: they
// need to point a store at a path chosen by the test rather than at the real user
// configuration directory, which a test must never write to.

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/product"
)

// configVariable names the environment variable the standard library resolves a user
// configuration directory from on this platform.
func configVariable() string {
	if runtime.GOOS == "windows" {
		return "AppData"
	}
	return "XDG_CONFIG_HOME"
}

func TestAStoreKeepsItsFileUnderTheApplicationsOwnDirectory(t *testing.T) {
	base := t.TempDir()
	t.Setenv(configVariable(), base)

	got := NewSettings().Path()
	want := filepath.Join(base, product.Slug, settingsFile)
	if got != want {
		t.Fatalf("path: got %q, want %q", got, want)
	}
}

// A machine with no configuration directory is not a reason to refuse to start. The
// store is still built, with no path; it then loads nothing and a save says why.
func TestAMachineWithNowhereToPutSettingsStillGetsAStore(t *testing.T) {
	t.Setenv(configVariable(), "")

	store := NewSettings()
	if store == nil {
		t.Fatal("no store was returned at all")
	}
	if store.Path() != "" {
		t.Fatalf("path: got %q, want empty", store.Path())
	}
}

// The three ways a write can fail all have to report rather than claim success. A
// save that silently did nothing would leave the user believing a choice was kept.
func TestASaveThatCannotCreateItsDirectoryReports(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "in the way")
	if err := os.WriteFile(blocker, []byte("a file, not a directory"), filePerm); err != nil {
		t.Fatalf("planting the blocker: %v", err)
	}
	store := &Settings{path: filepath.Join(blocker, "nested", settingsFile)}

	if err := store.Save(ports.Settings{LibraryRoot: "anywhere"}); err == nil {
		t.Fatal("a save under a path that cannot be a directory reported success")
	}
}

func TestASaveThatCannotWriteItsWorkingFileReports(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := &Settings{path: filepath.Join(dir, settingsFile)}
	// A directory standing where the working file must be written blocks the write
	// without needing permissions this test cannot portably set.
	if err := os.Mkdir(store.path+writingSuffix, dirPerm); err != nil {
		t.Fatalf("planting the blocker: %v", err)
	}

	if err := store.Save(ports.Settings{LibraryRoot: "anywhere"}); err == nil {
		t.Fatal("a save whose working file could not be written reported success")
	}
}

// The rename is what makes the write atomic. If it fails the working file must not be
// left behind; the next save would otherwise find its own blocker already in place.
func TestASaveThatCannotReplaceTheTargetReportsAndTidiesUp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := &Settings{path: filepath.Join(dir, settingsFile)}
	// A non-empty directory standing where the settings file belongs cannot be
	// renamed over on any platform this application ships to.
	if err := os.Mkdir(store.path, dirPerm); err != nil {
		t.Fatalf("planting the blocker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.path, "occupant"), []byte("x"), filePerm); err != nil {
		t.Fatalf("filling the blocker: %v", err)
	}

	if err := store.Save(ports.Settings{LibraryRoot: "anywhere"}); err == nil {
		t.Fatal("a save that could not replace the target reported success")
	}
	if _, err := os.Stat(store.path + writingSuffix); err == nil {
		t.Fatal("the working file was left behind after a failed rename")
	}
}
