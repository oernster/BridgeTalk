//go:build windows || linux

package speechmodel

// FR-518 and FR-237 for the model itself: with ONNX Runtime loaded from models/, a model that is
// missing or is not a model fails with a reason naming its path once. ONNX Runtime writes the path
// into its own message, twice for a missing file, measured on 2026-09-14.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
	"github.com/oernster/bridge-talk/internal/infrastructure/nativelib/nativelibtest"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
	"github.com/oernster/bridge-talk/internal/refusal"
)

func TestAModelMissingOrDamagedIsRefusedNamingItOnce(t *testing.T) {
	runtimePath := filepath.Join(modelfilestest.Require(t), voicefiles.RuntimeFile)
	modelPath := filepath.Join(t.TempDir(), voicefiles.ModelFile)

	refusedOver := func(what string) {
		t.Helper()
		loaded, err := openFrom(runtimePath, modelPath)
		if loaded != nil {
			loaded.release()
		}
		for _, problem := range refusal.Check(err, modelPath) {
			t.Errorf("%s: %s", what, problem)
		}
	}

	refusedOver("a missing model")
	if err := os.WriteFile(modelPath, []byte("not a model"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", modelPath, err)
	}
	refusedOver("a damaged model")
}

// A library that is not ONNX Runtime is refused naming it once; so is a model path no file can have.
func TestALibraryOrPathThatCannotBeUsedIsRefusedNamingItOnce(t *testing.T) {
	notTheRuntime := nativelibtest.SystemLibrary(t)
	loaded, err := openFrom(notTheRuntime, filepath.Join(t.TempDir(), voicefiles.ModelFile))
	if loaded != nil {
		loaded.release()
	}
	for _, problem := range refusal.Check(err, notTheRuntime) {
		t.Errorf("a library that is not ONNX Runtime: %s", problem)
	}

	runtimePath := filepath.Join(modelfilestest.Require(t), voicefiles.RuntimeFile)
	impossible := filepath.Join(t.TempDir(), "model\x00.onnx")
	loaded, err = openFrom(runtimePath, impossible)
	if loaded != nil {
		loaded.release()
	}
	for _, problem := range refusal.Check(err, impossible) {
		t.Errorf("a path no file can have: %s", problem)
	}
}
