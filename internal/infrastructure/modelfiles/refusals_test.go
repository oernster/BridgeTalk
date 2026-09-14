package modelfiles_test

// FR-537 and FR-237: a download that cannot be made, cannot be written, arrives cut short or cannot be
// put in place is refused, leaving nothing under its name.

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// An address that is no address, a server that is gone and an answer cut short are each a failed
// download.
func TestADownloadThatNeverArrivesWholeIsRefused(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := t.TempDir()
	unaddressed := server.Entry("bf_alice.bin", []byte("alice"))
	unaddressed.Address = "::no address"
	gone := modelfilestest.Serve(t)
	unreached := gone.Entry("bf_lily.bin", []byte("lily"))
	cut := server.Entry("bf_emma.bin", []byte("emma"))
	server.Cut("/bf_emma.bin", []byte("emma"))
	unarchived := unreached
	unarchived.Name, unarchived.Inside = "onnxruntime.dll", "release/lib/onnxruntime.dll"
	gone.Close()

	for _, file := range []modelfiles.File{unaddressed, unreached, cut, unarchived} {
		err := modelfiles.Fetch(context.Background(), server.Client(), dir, []modelfiles.File{file}, io.Discard)
		if !errors.Is(err, modelfiles.ErrDownload) {
			t.Errorf("%s: Fetch = %v, want %v", file.Name, err, modelfiles.ErrDownload)
		}
	}
	holds(t, dir, map[string][]byte{})
}

// A part that cannot be written, whether downloaded or taken from an archive, is refused naming the
// path once; so is a place held by a folder.
func TestAFileThatCannotBeWrittenOrPutInPlaceIsRefusedNamingItOnce(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	alice := []byte("alice")
	whole := server.Entry("bf_alice.bin", alice)
	packed := whole
	packed.Address = server.Offer("/release.zip", modelfilestest.Archive(t, map[string][]byte{"bf_alice.bin": alice}))
	packed.Inside = "bf_alice.bin"

	for _, file := range []modelfiles.File{whole, packed} {
		dir := t.TempDir()
		part := filepath.Join(dir, file.Name+".part")
		mkdir(t, part)
		err := modelfiles.Fetch(context.Background(), server.Client(), dir, []modelfiles.File{file}, io.Discard)
		for _, problem := range refusal.Check(err, part) {
			t.Errorf("a folder at the part of %s: %s", file.Address, problem)
		}
	}

	dir := t.TempDir()
	place := filepath.Join(dir, whole.Name)
	mkdir(t, filepath.Join(place, "inside"))
	err := modelfiles.Fetch(context.Background(), server.Client(), dir, []modelfiles.File{whole}, io.Discard)
	for _, problem := range refusal.Check(err, place) {
		t.Errorf("a folder at the place: %s", problem)
	}
	if _, err := os.Stat(place + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a part was left behind: %v", err)
	}
}

// mkdir makes a folder and any above it, failing the test where it cannot.
func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("making %s: %v", path, err)
	}
}
