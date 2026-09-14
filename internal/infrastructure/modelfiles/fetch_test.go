package modelfiles_test

// FR-536 and FR-537: filling the folder downloads only what does not match; a download that does
// not match leaves nothing in its place.

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// A file that matches is left alone; a missing one and a different one are downloaded into place.
func TestOnlyWhatDoesNotMatchIsDownloaded(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := t.TempDir()
	model, alice, adam := []byte("a model"), []byte("alice"), []byte("adam")
	files := []modelfiles.File{
		server.Entry("model.onnx", model), server.Entry("bf_alice.bin", alice), server.Entry("am_adam.bin", adam),
	}
	write(t, dir, "model.onnx", model)
	write(t, dir, "am_adam.bin", []byte("ADAM"))

	if err := modelfiles.Fetch(context.Background(), server.Client(), dir, files, io.Discard); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if asked, want := server.Asked(), []string{"/bf_alice.bin", "/am_adam.bin"}; !slices.Equal(asked, want) {
		t.Errorf("asked for %v, want %v", asked, want)
	}
	holds(t, dir, map[string][]byte{"model.onnx": model, "bf_alice.bin": alice, "am_adam.bin": adam})
}

// The folder is made where it is missing.
func TestTheFolderIsMadeWhereItIsMissing(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := filepath.Join(t.TempDir(), voicefiles.Folder)

	err := modelfiles.Fetch(context.Background(), server.Client(), dir,
		[]modelfiles.File{server.Entry("bf_alice.bin", []byte("alice"))}, io.Discard)

	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	holds(t, dir, map[string][]byte{"bf_alice.bin": []byte("alice")})
}

// Another digest, another size and a failed download are each refused naming the file with what was
// wrong, leaving nothing under its name; the files after them are still fetched.
func TestADownloadThatDoesNotMatchLeavesNothingInItsPlace(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := t.TempDir()
	altered := server.Entry("bf_alice.bin", []byte("alice"))
	server.Offer("/bf_alice.bin", []byte("ALICE"))
	resized := server.Entry("bf_lily.bin", []byte("lily"))
	server.Offer("/bf_lily.bin", []byte("lily, longer"))
	gone := server.Entry("bf_isabella.bin", []byte("isabella"))
	gone.Address += "-gone"
	emma := server.Entry("bf_emma.bin", []byte("emma"))

	err := modelfiles.Fetch(context.Background(), server.Client(), dir,
		[]modelfiles.File{altered, resized, gone, emma}, io.Discard)

	if !errors.Is(err, modelfiles.ErrDiffers) || !errors.Is(err, modelfiles.ErrDownload) {
		t.Fatalf("Fetch = %v, want both %v and %v", err, modelfiles.ErrDiffers, modelfiles.ErrDownload)
	}
	said := err.Error()
	for _, want := range []string{
		"bf_alice.bin", altered.SHA256, modelfilestest.Digest([]byte("ALICE")), "bf_lily.bin", "bf_isabella.bin",
	} {
		if !strings.Contains(said, want) {
			t.Errorf("Fetch said %q, which does not name %s", said, want)
		}
	}
	holds(t, dir, map[string][]byte{"bf_emma.bin": []byte("emma")})
}

// ONNX Runtime comes inside a release archive: the file is taken from its listed path and the archive
// is not kept. An archive without that path is refused, as is a download that is no archive at
// all; neither leaves anything behind.
func TestAFileIsTakenFromInsideItsArchive(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := t.TempDir()
	runtime := []byte("a runtime")
	address := server.Offer("/release.zip", modelfilestest.Archive(t, map[string][]byte{
		"release/lib/onnxruntime.dll": runtime, "release/README.md": []byte("words"),
	}))
	wanted := modelfiles.File{
		Name: "onnxruntime.dll", Address: address, Inside: "release/lib/onnxruntime.dll",
		Size: int64(len(runtime)), SHA256: modelfilestest.Digest(runtime),
	}
	absent := wanted
	absent.Name, absent.Inside = "other.dll", "release/lib/other.dll"
	broken := wanted
	broken.Name, broken.Address = "broken.dll", server.Offer("/broken.zip", []byte("not an archive"))

	err := modelfiles.Fetch(context.Background(), server.Client(), dir,
		[]modelfiles.File{absent, broken, wanted}, io.Discard)

	if !errors.Is(err, modelfiles.ErrNotInArchive) {
		t.Fatalf("Fetch = %v, want %v", err, modelfiles.ErrNotInArchive)
	}
	for _, want := range []string{"other.dll", "release/lib/other.dll", "broken.dll"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Fetch said %q, which does not name %s", err, want)
		}
	}
	holds(t, dir, map[string][]byte{"onnxruntime.dll": runtime})
}

// A folder that cannot be made is refused, named once (FR-237), before anything is asked for.
func TestAFolderThatCannotBeMadeIsRefusedNamingItOnce(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("a file"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", blocker, err)
	}
	dir := filepath.Join(blocker, voicefiles.Folder)

	err := modelfiles.Fetch(context.Background(), server.Client(), dir,
		[]modelfiles.File{server.Entry("bf_alice.bin", []byte("alice"))}, io.Discard)

	for _, problem := range refusal.Check(err, dir) {
		t.Error(problem)
	}
	if asked := server.Asked(); len(asked) != 0 {
		t.Errorf("asked for %v before the folder was made", asked)
	}
}
