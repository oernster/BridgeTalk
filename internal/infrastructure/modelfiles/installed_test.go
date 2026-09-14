package modelfiles_test

// FR-524 and FR-543: setup installs every listed file the application reads, which is every one but
// the tokenizer file.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
)

func TestSetupInstallsEveryListedFileButTheTokenizerFile(t *testing.T) {
	t.Parallel()
	files, err := modelfiles.Listed()
	if err != nil {
		t.Fatalf("reading the list: %v", err)
	}

	installed := modelfiles.Installed(files)

	if len(installed) != len(files)-1 {
		t.Errorf("installed %d of %d listed files, want every one but the tokenizer file", len(installed), len(files))
	}
	if slices.Contains(installed, modelfiles.TokenizerFile) {
		t.Errorf("installed %v, which holds %s", installed, modelfiles.TokenizerFile)
	}
	for _, want := range []string{"model.onnx", "onnxruntime.dll", "bf_alice.bin"} {
		if !slices.Contains(installed, want) {
			t.Errorf("installed %v, which does not hold %s", installed, want)
		}
	}
}
