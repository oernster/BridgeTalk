package modelfiles_test

// FR-538: checking the model files downloads nothing and names each file missing or different.

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// A file that matches says nothing; a missing one is listed as missing; one of another size or
// another digest is refused naming it with what differs. Nothing is asked of any address.
func TestCheckNamesWhatIsMissingAndWhatDiffersAskingForNothing(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := t.TempDir()
	altered, resized := []byte("adam"), []byte("echo")
	files := []modelfiles.File{
		server.Entry("model.onnx", []byte("a model")),
		server.Entry("bf_alice.bin", []byte("alice")),
		server.Entry("am_adam.bin", altered),
		server.Entry("am_echo.bin", resized),
	}
	write(t, dir, "model.onnx", []byte("a model"))
	write(t, dir, "am_adam.bin", []byte("ADAM"))
	write(t, dir, "am_echo.bin", []byte("echo, longer"))

	missing, err := modelfiles.Check(dir, files)

	if !slices.Equal(missing, []string{"bf_alice.bin"}) {
		t.Errorf("missing = %v, want [bf_alice.bin]", missing)
	}
	if !errors.Is(err, modelfiles.ErrDiffers) {
		t.Fatalf("Check = %v, want %v", err, modelfiles.ErrDiffers)
	}
	said := err.Error()
	for _, want := range []string{"am_adam.bin", modelfilestest.Digest(altered), modelfilestest.Digest([]byte("ADAM")), "am_echo.bin"} {
		if !strings.Contains(said, want) {
			t.Errorf("Check said %q, which does not name %s", said, want)
		}
	}
	for _, unwanted := range []string{"model.onnx", "bf_alice.bin"} {
		if strings.Contains(said, unwanted) {
			t.Errorf("Check said %q, naming %s, which is not different", said, unwanted)
		}
	}
	if asked := server.Asked(); len(asked) != 0 {
		t.Errorf("Check asked for %v, want nothing", asked)
	}
}

// A folder standing where a file should be is neither missing nor matching: it is refused, named
// once (FR-237).
func TestAFolderWhereAFileShouldBeIsRefusedNamingItOnce(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "model.onnx")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("making %s: %v", path, err)
	}

	missing, err := modelfiles.Check(dir, []modelfiles.File{server.Entry("model.onnx", []byte("a model"))})

	if len(missing) != 0 {
		t.Errorf("missing = %v, want none", missing)
	}
	for _, problem := range refusal.Check(err, path) {
		t.Error(problem)
	}
}

// Verify says in one error what Check finds, each missing file and each different one by name; a
// folder that matches says nothing.
func TestVerifyNamesEveryFileMissingOrDifferent(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := t.TempDir()
	files := []modelfiles.File{
		server.Entry("model.onnx", []byte("a model")),
		server.Entry("bf_alice.bin", []byte("alice")),
		server.Entry("am_adam.bin", []byte("adam")),
	}
	write(t, dir, "model.onnx", []byte("a model"))
	write(t, dir, "am_adam.bin", []byte("ADAM"))

	err := modelfiles.Verify(dir, files)

	if !errors.Is(err, modelfiles.ErrDiffers) {
		t.Fatalf("Verify = %v, want %v", err, modelfiles.ErrDiffers)
	}
	for _, want := range []string{"bf_alice.bin", "am_adam.bin"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Verify said %q, which does not name %s", err, want)
		}
	}
	if strings.Contains(err.Error(), "model.onnx") {
		t.Errorf("Verify said %q, naming model.onnx, which matches", err)
	}
	write(t, dir, "bf_alice.bin", []byte("alice"))
	write(t, dir, "am_adam.bin", []byte("adam"))
	if err := modelfiles.Verify(dir, files); err != nil {
		t.Errorf("Verify over a matching folder = %v, want nothing", err)
	}
}
