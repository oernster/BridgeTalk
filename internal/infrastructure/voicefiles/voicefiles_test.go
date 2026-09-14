package voicefiles_test

// FR-519 and FR-513: the files a machine voice is made from, read from the folder setup fills.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/speech"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// styleValue fills every number of the style files written below.
const styleValue float32 = 0.5

// The contents of the model and ONNX Runtime written below; only their digests matter.
var (
	modelBytes   = []byte("a model")
	runtimeBytes = []byte("a runtime")
)

// styleBytes is a style file shaped as the model reads it, every number styleValue.
func styleBytes(count int) []byte {
	raw := make([]byte, 0, count*4)
	for range count {
		raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(styleValue))
	}
	return raw
}

// write puts a file in place, failing the test where it cannot.
func write(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// installed fills a folder as setup would for bf_emma alone.
func installed(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, voicefiles.ModelFile), modelBytes)
	write(t, filepath.Join(dir, voicefiles.RuntimeFile), runtimeBytes)
	write(t, filepath.Join(dir, "bf_emma.bin"), styleBytes(speech.MaxSymbols*speech.StyleWidth))
	return dir
}

func voiceNamed(t *testing.T, id string) machinevoice.Voice {
	t.Helper()
	voice, err := machinevoice.Parse(id)
	if err != nil {
		t.Fatalf("Parse(%q): %v", id, err)
	}
	return voice
}

func digestOf(contents []byte) string {
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:])
}

// FR-513: a voice's material is its style file read as numbers, with the digests of that file and
// the model its made lines are keyed by.
func TestAVoicesMaterialIsReadFromItsFiles(t *testing.T) {
	t.Parallel()
	dir := installed(t)

	material, err := voicefiles.New(dir).Open(voiceNamed(t, "bf_emma"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	tokens, err := speech.Tokens("k")
	if err != nil {
		t.Fatalf("Tokens: %v", err)
	}
	row, err := material.Style.For(tokens)
	if err != nil || slices.ContainsFunc(row, func(value float32) bool { return value != styleValue }) {
		t.Errorf("the style row read %v, %v; want every number %v", row[:2], err, styleValue)
	}
	if want := digestOf(styleBytes(speech.MaxSymbols * speech.StyleWidth)); material.Files.Style != want {
		t.Errorf("style digest = %s, want %s", material.Files.Style, want)
	}
	if want := digestOf(modelBytes); material.Files.Model != want {
		t.Errorf("model digest = %s, want %s", material.Files.Model, want)
	}
}

// FR-519: a file the voice is made from that is missing or cannot be read refuses the voice, naming
// that file once (FR-237). A folder where the file should be cannot be read; nor can a style file
// of the wrong size.
func TestAMissingOrUnreadableFileIsRefusedNamingItOnce(t *testing.T) {
	t.Parallel()
	for _, name := range []string{voicefiles.ModelFile, voicefiles.RuntimeFile, "bf_emma.bin"} {
		missing := installed(t)
		path := filepath.Join(missing, name)
		if err := os.Remove(path); err != nil {
			t.Fatalf("removing %s: %v", path, err)
		}
		_, err := voicefiles.New(missing).Open(voiceNamed(t, "bf_emma"))
		refusedOver(t, "missing "+name, err, path, fs.ErrNotExist)

		folder := installed(t)
		path = filepath.Join(folder, name)
		if err := os.Remove(path); err != nil {
			t.Fatalf("removing %s: %v", path, err)
		}
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatalf("making %s: %v", path, err)
		}
		_, err = voicefiles.New(folder).Open(voiceNamed(t, "bf_emma"))
		refusedOver(t, "a folder named "+name, err, path, nil)
	}

	for _, size := range []int{4, speech.MaxSymbols*speech.StyleWidth*4 + 1} {
		dir := installed(t)
		path := filepath.Join(dir, "bf_emma.bin")
		write(t, path, make([]byte, size))
		_, err := voicefiles.New(dir).Open(voiceNamed(t, "bf_emma"))
		refusedOver(t, "a style file of the wrong size", err, path, speech.ErrMisshapenStyle)
	}
}

// Each voice has a style file of its own: one voice's being there does not stand in for another's.
func TestAnotherVoicesStyleFileIsNoStandIn(t *testing.T) {
	t.Parallel()
	dir := installed(t)
	path := filepath.Join(dir, "am_michael.bin")

	_, err := voicefiles.New(dir).Open(voiceNamed(t, "am_michael"))

	refusedOver(t, "am_michael with only bf_emma's style file", err, path, fs.ErrNotExist)
}

// refusedOver fails the test unless err refuses over path naming it once. Where want is given, err
// must also answer errors.Is for it.
func refusedOver(t *testing.T, what string, err error, path string, want error) {
	t.Helper()
	for _, problem := range refusal.Check(err, path) {
		t.Errorf("%s: %s", what, problem)
	}
	if want != nil && !errors.Is(err, want) {
		t.Errorf("%s: refused with %v, want %v", what, err, want)
	}
}
