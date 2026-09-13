package setup

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// zipOf builds an in-memory archive from a name-to-content map.
func zipOf(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range entries {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("creating entry %q: %v", name, err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatalf("writing entry %q: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing archive: %v", err)
	}
	return buffer.Bytes()
}

func TestExtractZipWritesEveryEntry(t *testing.T) {
	t.Parallel()
	dest := t.TempDir()
	payload := zipOf(t, map[string]string{
		ExeName:             "binary",
		"assets/readme.txt": "a readme",
	})

	if err := ExtractZip(payload, dest); err != nil {
		t.Fatalf("extracting: %v", err)
	}
	for name, want := range map[string]string{
		ExeName:             "binary",
		"assets/readme.txt": "a readme",
	} {
		raw, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if string(raw) != want {
			t.Errorf("%s holds %q, want %q", name, string(raw), want)
		}
	}
}

// TestExtractZipRejectsAPathThatEscapes proves the fence bites. An archive that
// climbs out of the install directory is the one thing extraction must refuse.
func TestExtractZipRejectsAPathThatEscapes(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "install")
	payload := zipOf(t, map[string]string{"../escaped.txt": "no"})

	if err := ExtractZip(payload, dest); err == nil {
		t.Fatal("extraction accepted a path that climbs out of the destination")
	}
}

func TestExtractZipRejectsAnArchiveItCannotRead(t *testing.T) {
	t.Parallel()
	if err := ExtractZip([]byte("not a zip"), t.TempDir()); err == nil {
		t.Fatal("extraction accepted something that is not an archive")
	}
}

func TestDirSizeKBTotalsTheTree(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	nested := filepath.Join(dir, "assets")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("creating %s: %v", nested, err)
	}
	// Two kilobytes plus a byte, so the division to whole kilobytes is exercised
	// rather than landing on a round number by luck.
	if err := os.WriteFile(filepath.Join(nested, "big"), make([]byte, 2049), 0o644); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	size, err := DirSizeKB(dir)
	if err != nil {
		t.Fatalf("sizing: %v", err)
	}
	if size != 2 {
		t.Errorf("size is %d KB, want 2", size)
	}
}

func TestDirSizeKBFailsOnAMissingTree(t *testing.T) {
	t.Parallel()
	if _, err := DirSizeKB(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("sizing accepted a directory that is not there")
	}
}

func TestCopyFileReproducesTheContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "setup.exe")
	dst := filepath.Join(dir, "uninstall.exe")
	if err := os.WriteFile(src, []byte("payload"), 0o755); err != nil {
		t.Fatalf("writing the source: %v", err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("copying: %v", err)
	}
	raw, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading the copy: %v", err)
	}
	if string(raw) != "payload" {
		t.Errorf("the copy holds %q, want %q", string(raw), "payload")
	}
}

func TestCopyFileFailsWhenTheSourceIsAbsent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := CopyFile(filepath.Join(dir, "absent"), filepath.Join(dir, "out")); err == nil {
		t.Fatal("the copy accepted a source that is not there")
	}
}

func TestRemoveTreeDeletesTheWholeTree(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "state")
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("creating the tree: %v", err)
	}

	if err := RemoveTree(dir); err != nil {
		t.Fatalf("removing: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the tree is still there: %v", err)
	}
}

// TestTheLoginEntryNamesTheRealPath guards the value written into the Run key.
//
// It was fmt.Sprintf("%q", path), which is Go's quoting rather than the shell's: it
// escapes the separators inside the string, so the entry reached the registry reading
// "C:\Users\..." and Windows looked for a place that does not exist. The application
// and the setup program both write this entry, so both were broken and neither said
// so: the box read as on while nothing launched at sign-in.
func TestTheLoginEntryNamesTheRealPath(t *testing.T) {
	// Built from the package's own names rather than written out, since the product is
	// named in one place and read from there.
	path := filepath.Join(`C:\Users\Someone\AppData\Local\Programs`, InstallFolder, ExeName)
	written := runValue(path)

	if strings.Contains(written, `\\`) {
		t.Errorf("run value = %s, want single separators: a doubled path is not a place", written)
	}
	if written != `"`+path+`" `+HiddenFlag {
		t.Errorf("run value = %s, want the path in plain quotes then the hidden flag", written)
	}
	if back := runTarget(written); back != path {
		t.Errorf("read back = %s, want the path alone, with the argument left off", back)
	}
}

// TestALoginEntryReadsBackFromAnyQuoting keeps the reader tolerant of what is already
// on disk, since an entry written by an older build is still out there.
func TestALoginEntryReadsBackFromAnyQuoting(t *testing.T) {
	// A path with a space in it, because that is where splitting on the first space
	// rather than reading to the closing quote would go wrong.
	path := `C:\Program Files\Thing\thing.exe`
	stored := []string{
		path,
		`"` + path + `"`,
		` "` + path + `" `,
		`"` + path + `" ` + HiddenFlag,
	}
	for _, stored := range stored {
		if back := runTarget(stored); back != path {
			t.Errorf("runTarget(%q) = %q, want %q", stored, back, path)
		}
	}
}

// TestTheUninstallEntryNamesTheRealPath guards the two values the Apps list runs.
//
// UninstallString and ModifyPath name the uninstaller copy the way the login entry names
// the application, so they carry the same risk: Go's %q escapes the separators inside
// the path, which then names a place that does not exist.
func TestTheUninstallEntryNamesTheRealPath(t *testing.T) {
	path := filepath.Join(`C:\Users\Someone\AppData\Local\Programs`, InstallFolder, "uninstall.exe")
	values := uninstallValues(UninstallInfo{UninstallExe: path})

	want := map[string]string{
		"UninstallString": `"` + path + `" ` + UninstallFlag,
		"ModifyPath":      `"` + path + `"`,
	}
	for name, expected := range want {
		if got := values[name]; got != expected {
			t.Errorf("%s = %s, want %s", name, got, expected)
		}
		if back := runTarget(values[name]); back != path {
			t.Errorf("%s reads back as %s, want the path alone", name, back)
		}
	}
}
