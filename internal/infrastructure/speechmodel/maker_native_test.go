//go:build windows || linux

package speechmodel_test

// The real model through ONNX Runtime, loaded with cgo disabled: a shipped line made for a shipped
// voice (FR-511); a folder that cannot be loaded failing the line with a reason that names the file
// once (FR-518, FR-237). The tests that need the model files skip where models/ is not filled
// (FR-538).

import (
	"context"
	"math"
	"path/filepath"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/speech"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
	"github.com/oernster/bridge-talk/internal/infrastructure/speechmodel"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// shippedLine is a cue the shipped script voices.
const shippedLine cue.ID = "Docked"

// lineFor answers with the numbers and style row of the shipped line's first take for the voice
// named, read the way the making service reads them.
func lineFor(t *testing.T, dir, id string) ([]int64, []float32) {
	t.Helper()
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("LoadCueTable: %v", err)
	}
	voiced, err := config.LoadVoicedScript(table)
	if err != nil {
		t.Fatalf("LoadVoicedScript: %v", err)
	}
	voice, err := machinevoice.Parse(id)
	if err != nil {
		t.Fatalf("Parse(%q): %v", id, err)
	}
	sounds, ok := voiced.Sounds(shippedLine, voice.Accent())
	if !ok || len(sounds) == 0 {
		t.Fatalf("the shipped script gives %s no sounds", shippedLine)
	}
	tokens, err := speech.Tokens(sounds[0])
	if err != nil {
		t.Fatalf("Tokens: %v", err)
	}
	material, err := voicefiles.New(dir).Open(voice)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	row, err := material.Style.For(tokens)
	if err != nil {
		t.Fatalf("For: %v", err)
	}
	return tokens, row
}

// The model makes a shipped line: samples that are all numbers and not all silence.
func TestAShippedLineIsMadeByTheRealModel(t *testing.T) {
	dir := modelfilestest.Require(t)
	tokens, row := lineFor(t, dir, "bf_emma")
	maker := speechmodel.New(dir)
	defer maker.Close()

	samples, err := maker.Make(context.Background(), tokens, row)

	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	notANumber := func(sample float32) bool {
		return math.IsNaN(float64(sample)) || math.IsInf(float64(sample), 0)
	}
	if len(samples) == 0 || slices.ContainsFunc(samples, notANumber) {
		t.Fatalf("made %d samples, some not a number: want a line", len(samples))
	}
	if !slices.ContainsFunc(samples, func(sample float32) bool { return sample != 0 }) {
		t.Errorf("made %d samples of silence", len(samples))
	}
}

// A folder with no ONNX Runtime in it fails the line naming the runtime once. The model's own
// refusals are tested beside openFrom, over the ONNX Runtime in models/, since a library once loaded
// cannot be deleted from a test's temporary folder.
func TestAFolderWithoutONNXRuntimeFailsTheLineNamingItOnce(t *testing.T) {
	t.Parallel()
	empty := t.TempDir()
	maker := speechmodel.New(empty)
	defer maker.Close()

	_, err := maker.Make(context.Background(), []int64{0, 50, 0}, make([]float32, speech.StyleWidth))

	for _, problem := range refusal.Check(err, filepath.Join(empty, voicefiles.RuntimeFile)) {
		t.Error(problem)
	}
}
