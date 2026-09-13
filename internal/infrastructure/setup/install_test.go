package setup

// These tests are inside the package so they can reach the unexported helpers the
// leftover scan is built from. Nothing here writes to the registry, creates a
// shortcut or starts a process: those live in the Windows files beside this one and
// act on the machine itself, which a test must not.

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// archiveOf builds a zip in memory from a name-to-contents map, with a name ending in
// a separator taken as a directory entry.
func archiveOf(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range entries {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("adding %q: %v", name, err)
		}
		if _, err := file.Write([]byte(body)); err != nil {
			t.Fatalf("writing %q: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing the archive: %v", err)
	}
	return buffer.Bytes()
}

// This is what decides that a shortcut is ours, so it answers false rather than
// guessing. The comparison folds case because the paths come from Windows.
func TestOnlyRealChildrenOfADirectoryCountAsUnderIt(t *testing.T) {
	t.Parallel()
	base := filepath.Join("C:", "Users", "Someone", "Desktop")
	cases := []struct {
		path, dir string
		want      bool
	}{
		{filepath.Join(base, "Shortcut.lnk"), base, true},
		{filepath.Join(base, "nested", "Shortcut.lnk"), base, true},
		{strings.ToUpper(filepath.Join(base, "Shortcut.lnk")), base, true},
		{base, base, true},
		{filepath.Join("C:", "Users", "Someone", "Documents", "x.lnk"), base, false},
		{base + "Extra", base, false},
		{"", base, false},
		{filepath.Join(base, "x.lnk"), "", false},
		// One path absolute and the other relative cannot be compared at all.
		{filepath.Join("relative", "x.lnk"), base, false},
	}
	for _, each := range cases {
		if got := under(each.path, each.dir); got != each.want {
			t.Errorf("under(%q, %q) = %v, want %v", each.path, each.dir, got, each.want)
		}
	}
}

// Installing under the local application data is what keeps the whole flow free of an
// administrator prompt, so the directory is derived from the environment rather than
// written down as a path.
func TestTheInstallAndStateDirectoriesComeFromTheEnvironment(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	t.Setenv("APPDATA", base)

	install, err := InstallDir()
	if err != nil {
		t.Fatalf("install dir: %v", err)
	}
	if install != filepath.Join(base, installSubdir, InstallFolder) {
		t.Fatalf("install dir = %q", install)
	}

	// WebView2 falls back to the executable's own file name, the suffix included.
	state, err := StateDir()
	if err != nil {
		t.Fatalf("state dir: %v", err)
	}
	if state != filepath.Join(base, ExeName) {
		t.Fatalf("state dir = %q", state)
	}
}

func TestAMachineWithNoApplicationDataIsReportedRatherThanGuessedAt(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("APPDATA", "")

	if _, err := InstallDir(); err == nil {
		t.Error("an install directory was invented with no LOCALAPPDATA")
	}
	if _, err := StateDir(); err == nil {
		t.Error("a state directory was invented with no APPDATA")
	}
}

func TestAPayloadIsExtractedWithItsDirectoriesMade(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "install")
	payload := archiveOf(t, map[string]string{
		"app.exe":            "the program",
		"assets/readme.txt":  "a readme",
		"assets/nested/a.md": "a note",
	})

	if err := ExtractZip(payload, dest); err != nil {
		t.Fatalf("extracting: %v", err)
	}
	for name, want := range map[string]string{
		"app.exe":            "the program",
		"assets/readme.txt":  "a readme",
		"assets/nested/a.md": "a note",
	} {
		got, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("reading %q: %v", name, err)
		}
		if string(got) != want {
			t.Fatalf("%q holds %q, want %q", name, got, want)
		}
	}
}

// An entry whose name climbs out of the destination would write anywhere on the
// machine the setup program can reach. It is refused rather than sanitised, so a
// payload that tries it is a payload that fails loudly.
func TestAnEntryThatClimbsOutOfTheDestinationIsRefused(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "install")
	payload := archiveOf(t, map[string]string{"../escaped.txt": "somewhere else"})

	err := ExtractZip(payload, dest)
	if err == nil {
		t.Fatal("an entry climbing out of the destination was written")
	}
	if !strings.Contains(err.Error(), "unsafe path") {
		t.Errorf("refusal = %q, want it to say what was wrong", err)
	}
}

func TestAPayloadThatIsNotAnArchiveIsReported(t *testing.T) {
	t.Parallel()
	if err := ExtractZip([]byte("this is not a zip file"), t.TempDir()); err == nil {
		t.Fatal("bytes that are not an archive extracted")
	}
}

func TestAnInstallDirectoryThatCannotBeMadeIsReported(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "in the way")
	if err := os.WriteFile(blocker, []byte("a file, not a directory"), 0o644); err != nil {
		t.Fatalf("planting the blocker: %v", err)
	}

	err := ExtractZip(archiveOf(t, map[string]string{"a.txt": "x"}),
		filepath.Join(blocker, "install"))
	if err == nil {
		t.Fatal("an install directory under a file was created")
	}
}

// An entry that cannot be written has to stop the install rather than leaving a
// half-extracted program that starts and then fails on a missing file.
func TestAnEntryThatCannotBeWrittenStopsTheExtraction(t *testing.T) {
	t.Parallel()
	dest := t.TempDir()
	// A directory standing where a file entry must be written cannot be overwritten.
	if err := os.Mkdir(filepath.Join(dest, "app.exe"), 0o755); err != nil {
		t.Fatalf("planting the blocker: %v", err)
	}

	if err := ExtractZip(archiveOf(t, map[string]string{"app.exe": "x"}), dest); err == nil {
		t.Fatal("an entry was written over a directory")
	}
}

func TestAnEntrysParentDirectoryThatCannotBeMadeStopsTheExtraction(t *testing.T) {
	t.Parallel()
	dest := t.TempDir()
	// A file standing where the entry's parent directory must be.
	if err := os.WriteFile(filepath.Join(dest, "assets"), []byte("x"), 0o644); err != nil {
		t.Fatalf("planting the blocker: %v", err)
	}

	err := ExtractZip(archiveOf(t, map[string]string{"assets/readme.txt": "x"}), dest)
	if err == nil {
		t.Fatal("an entry was written beneath a file")
	}
}

func TestTheSavedStateIsRemovedWhenTheUninstallIsAskedTo(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "state")
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("building the state directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("filling the state directory: %v", err)
	}

	if err := RemoveTree(dir); err != nil {
		t.Fatalf("removing: %v", err)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Fatal("the state directory survived")
	}
	// Removing what is already gone is not a failure; an uninstall run twice must
	// not report an error the second time.
	if err := RemoveTree(dir); err != nil {
		t.Fatalf("removing what was already gone: %v", err)
	}
}

func TestTheSetupProgramIsCopiedBesideTheInstall(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	source := filepath.Join(dir, "setup.exe")
	if err := os.WriteFile(source, []byte("the setup program"), 0o644); err != nil {
		t.Fatalf("planting the source: %v", err)
	}
	target := filepath.Join(dir, "uninstall.exe")

	if err := CopyFile(source, target); err != nil {
		t.Fatalf("copying: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading the copy: %v", err)
	}
	if string(got) != "the setup program" {
		t.Fatalf("the copy holds %q", got)
	}
}

// The Apps list calls the copy to uninstall, so a copy that silently did not happen
// would leave an entry whose remove button does nothing.
func TestACopyThatCannotBeMadeIsReported(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := CopyFile(filepath.Join(dir, "absent.exe"), filepath.Join(dir, "out.exe")); err == nil {
		t.Error("a source that is not there was copied")
	}

	source := filepath.Join(dir, "setup.exe")
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatalf("planting the source: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "blocked.exe"), 0o755); err != nil {
		t.Fatalf("planting the blocker: %v", err)
	}
	if err := CopyFile(source, filepath.Join(dir, "blocked.exe")); err == nil {
		t.Error("a copy was written over a directory")
	}
}

func TestADirectoryTreeIsSizedInKilobytes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.bin"), make([]byte, 2048), 0o644); err != nil {
		t.Fatalf("planting: %v", err)
	}

	size, err := DirSizeKB(dir)
	if err != nil {
		t.Fatalf("sizing: %v", err)
	}
	if size != 2 {
		t.Fatalf("size = %d KB, want 2", size)
	}
}

func TestSizingADirectoryThatIsNotThereIsReported(t *testing.T) {
	t.Parallel()
	if _, err := DirSizeKB(filepath.Join(t.TempDir(), "nowhere")); err == nil {
		t.Fatal("a directory that is not there was sized")
	}
}
