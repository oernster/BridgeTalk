package setup

// FR-524: the payload carries the application at the root, then the model files in their folder
// beside it, so extracting it lays the install out as the application reads it (FR-539).

import (
	"archive/zip"
	"bytes"
	"errors"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"
)

// modelsFolder is the folder the tests place the model files in.
const modelsFolder = "models"

// plant writes body at name under dir, making the folders it sits in.
func plant(t *testing.T, dir, name, body string) {
	t.Helper()
	place := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(place), dirPerm); err != nil {
		t.Fatalf("making the folder for %s: %v", place, err)
	}
	if err := os.WriteFile(place, []byte(body), dirPerm); err != nil {
		t.Fatalf("writing %s: %v", place, err)
	}
}

// The archive holds each application file under its path and each named model file in the folder,
// nothing else; extracted, every file holds what was packed.
func TestPackedModelFilesAreExtractedIntoTheFolderBesideTheApplication(t *testing.T) {
	t.Parallel()
	app, models := t.TempDir(), t.TempDir()
	plant(t, app, ExeName, "the program")
	plant(t, app, "assets/readme.txt", "a readme")
	plant(t, models, "bf_alice.bin", "alice")
	plant(t, models, "tokenizer.json", "not named, so not carried")
	var packed bytes.Buffer

	err := Pack(&packed, Payload{App: app, ModelsDir: models, Folder: modelsFolder, Models: []string{"bf_alice.bin"}})

	if err != nil {
		t.Fatalf("packing: %v", err)
	}
	want := map[string]string{
		ExeName:                                 "the program",
		"assets/readme.txt":                     "a readme",
		path.Join(modelsFolder, "bf_alice.bin"): "alice",
	}
	reader, err := zip.NewReader(bytes.NewReader(packed.Bytes()), int64(packed.Len()))
	if err != nil {
		t.Fatalf("reading the archive: %v", err)
	}
	var names []string
	for _, member := range reader.File {
		names = append(names, member.Name)
	}
	slices.Sort(names)
	if wanted := slices.Sorted(maps.Keys(want)); !slices.Equal(names, wanted) {
		t.Errorf("the archive holds %v, want %v", names, wanted)
	}
	dest := t.TempDir()
	if err := ExtractZip(packed.String(), dest); err != nil {
		t.Fatalf("extracting: %v", err)
	}
	for name, body := range want {
		got, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(name)))
		if err != nil || string(got) != body {
			t.Errorf("%s holds %q (%v), want %q", name, got, err, body)
		}
	}
}

// errFull is what a writer with no room answers.
var errFull = errors.New("no room")

// fullWriter refuses every write.
type fullWriter struct{}

func (fullWriter) Write([]byte) (int, error) { return 0, errFull }

// A payload that cannot be written is refused with the writer's reason.
func TestAPayloadThatCannotBeWrittenIsRefused(t *testing.T) {
	t.Parallel()
	app := t.TempDir()
	plant(t, app, ExeName, "the program")

	err := Pack(fullWriter{}, Payload{App: app})

	if !errors.Is(err, errFull) {
		t.Errorf("Pack = %v, want %v", err, errFull)
	}
}
