package wholefile_test

// A file is put in place whole or not at all: written beside its place, then renamed into it. Made
// lines (FR-517), the stored settings and the model files (FR-537) are all written this way.

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/wholefile"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// suffix follows a file's name while it is written, in every test below.
const suffix = ".part"

// filePerm is how the files below are written.
const filePerm = 0o644

// writing answers with a write that puts body in the file and fails with nothing.
func writing(body string) func(io.Writer) error {
	return func(file io.Writer) error {
		_, err := io.WriteString(file, body)
		return err
	}
}

// holds fails the test unless path holds body.
func holds(t *testing.T, path, body string) {
	t.Helper()
	if got, err := os.ReadFile(path); err != nil || string(got) != body {
		t.Errorf("%s holds %q, %v; want %q", path, got, err, body)
	}
}

// gone fails the test unless nothing stands at path.
func gone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s is still there: %v", path, err)
	}
}

// A write puts the file in place, replacing what was there and anything an interrupted write left
// beside it; nothing is left beside it after.
func TestAFileIsPutInPlaceWhole(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path+suffix, []byte("half of an earlier write, longer than the next"), filePerm); err != nil {
		t.Fatalf("leaving a part: %v", err)
	}

	for _, body := range []string{"first", "second"} {
		if err := wholefile.Write(path, suffix, filePerm, writing(body)); err != nil {
			t.Fatalf("Write(%q): %v", body, err)
		}
		holds(t, path, body)
		gone(t, path+suffix)
	}
}

// A write that fails leaves the file as it was and nothing beside it, answering with the write's own
// error.
func TestAFailedWriteChangesNothing(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := wholefile.Write(path, suffix, filePerm, writing("kept")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	failure := errors.New("the write failed")

	err := wholefile.Write(path, suffix, filePerm, func(file io.Writer) error {
		_, _ = io.WriteString(file, "half")
		return failure
	})

	if !errors.Is(err, failure) {
		t.Errorf("Write = %v, want %v", err, failure)
	}
	holds(t, path, "kept")
	gone(t, path+suffix)
}

// A file beside its place that cannot be made is refused naming it once (FR-237), asking nothing of
// the write; a place that cannot take the rename is refused naming the place once, leaving nothing
// beside it.
func TestAFileThatCannotBeMadeOrPutInPlaceIsRefusedNamingItOnce(t *testing.T) {
	t.Parallel()
	blocked := filepath.Join(t.TempDir(), "settings.json")
	if err := os.Mkdir(blocked+suffix, 0o755); err != nil {
		t.Fatalf("making %s: %v", blocked+suffix, err)
	}
	asked := false
	err := wholefile.Write(blocked, suffix, filePerm, func(io.Writer) error { asked = true; return nil })
	for _, problem := range refusal.Check(err, blocked+suffix) {
		t.Errorf("a folder beside the place: %s", problem)
	}
	if asked {
		t.Error("the write was asked for with nowhere to write")
	}

	held := filepath.Join(t.TempDir(), "settings.json")
	if err := os.MkdirAll(filepath.Join(held, "inside"), 0o755); err != nil {
		t.Fatalf("making %s: %v", held, err)
	}
	err = wholefile.Write(held, suffix, filePerm, writing("never placed"))
	for _, problem := range refusal.Check(err, held) {
		t.Errorf("a folder at the place: %s", problem)
	}
	gone(t, held+suffix)
}
