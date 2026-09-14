package main

// The payload tool: the application, then every model file setup installs in the folder beside it,
// checked against the list first (FR-543).

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
	"github.com/oernster/bridge-talk/internal/infrastructure/setup"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

// put writes body at name under dir.
func put(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// fixture lays out a built application beside a models folder holding every file of a list naming a
// style file, the model and the tokenizer file; it answers with the two folders and the list.
func fixture(t *testing.T) (string, string, []modelfiles.File) {
	t.Helper()
	app, models := t.TempDir(), t.TempDir()
	put(t, app, setup.ExeName, "the program")
	var files []modelfiles.File
	for _, name := range []string{"bf_alice.bin", "model.onnx", modelfiles.TokenizerFile} {
		put(t, models, name, name)
		files = append(files, modelfiles.File{Name: name, Size: int64(len(name)), SHA256: modelfilestest.Digest([]byte(name))})
	}
	return app, models, files
}

func TestThePayloadHoldsTheApplicationThenEveryModelFileSetupInstalls(t *testing.T) {
	t.Parallel()
	app, models, files := fixture(t)
	archive := filepath.Join(t.TempDir(), "payload.zip")

	if err := run([]string{"-app", app, "-out", archive}, models, files, io.Discard); err != nil {
		t.Fatalf("packing: %v", err)
	}

	reader, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatalf("reading the archive: %v", err)
	}
	defer reader.Close()
	var names []string
	for _, member := range reader.File {
		names = append(names, member.Name)
	}
	slices.Sort(names)
	want := []string{setup.ExeName, path.Join(voicefiles.Folder, "bf_alice.bin"), path.Join(voicefiles.Folder, "model.onnx")}
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Errorf("the archive holds %v, want %v", names, want)
	}
}

// A models folder that does not match the list is named file by file and the archive is left as it
// was.
func TestAModelsFolderThatDoesNotMatchLeavesTheArchiveAsItWas(t *testing.T) {
	t.Parallel()
	app, models, files := fixture(t)
	put(t, models, "bf_alice.bin", "ALICE")
	if err := os.Remove(filepath.Join(models, "model.onnx")); err != nil {
		t.Fatalf("removing model.onnx: %v", err)
	}
	archive := filepath.Join(t.TempDir(), "payload.zip")
	put(t, filepath.Dir(archive), filepath.Base(archive), "the archive before")

	err := run([]string{"-app", app, "-out", archive}, models, files, io.Discard)

	for _, want := range []string{"bf_alice.bin", "model.onnx"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("packing = %v, want a failure naming %s", err, want)
		}
	}
	assertUntouched(t, archive)
}

// An application folder without the application is refused and the archive is left as it was.
func TestAnApplicationFolderWithoutTheApplicationLeavesTheArchiveAsItWas(t *testing.T) {
	t.Parallel()
	_, models, files := fixture(t)
	archive := filepath.Join(t.TempDir(), "payload.zip")
	put(t, filepath.Dir(archive), filepath.Base(archive), "the archive before")

	err := run([]string{"-app", t.TempDir(), "-out", archive}, models, files, io.Discard)

	if err == nil || !strings.Contains(err.Error(), setup.ExeName) {
		t.Errorf("packing = %v, want a failure naming %s", err, setup.ExeName)
	}
	assertUntouched(t, archive)
}

// assertUntouched fails the test unless the archive still holds what it held before the tool ran.
func assertUntouched(t *testing.T, archive string) {
	t.Helper()
	if got, err := os.ReadFile(archive); err != nil || string(got) != "the archive before" {
		t.Errorf("the archive holds %q (%v), want it as it was", got, err)
	}
}

// The tool is told both folders or packs nothing.
func TestTheToolIsToldBothFolders(t *testing.T) {
	t.Parallel()
	app, models, files := fixture(t)

	err := run([]string{"-app", app}, models, files, io.Discard)

	if !errors.Is(err, errNoFolders) {
		t.Errorf("packing without -out = %v, want %v", err, errNoFolders)
	}
}
