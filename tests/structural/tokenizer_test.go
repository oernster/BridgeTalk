package structural

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/oernster/bridge-talk/internal/domain/speech"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
)

// boundaryKey is how the tokenizer file writes the marker at each end of a line.
const boundaryKey = "$"

// tokenizerFile is the part of the model's tokenizer file the table is copied from.
type tokenizerFile struct {
	Model struct {
		Vocab map[string]int64 `json:"vocab"`
	} `json:"model"`
}

// TestTheSymbolTableIsTheModelsOwn holds the speech sound table in internal/domain/speech to the
// model's tokenizer file, so a new model cannot leave the table behind (FR-506, FR-531). Every
// symbol the file gives must be in the table with the same number; the boundary must be the file's
// marker; the table must hold nothing the file does not. The model drops a symbol it does not hold
// without complaint, so nothing later would catch a table that drifted.
//
// Proved by planting a symbol's number changed, a symbol removed and a symbol added in
// internal/domain/speech/symbols.go, reading the exit code each time.
func TestTheSymbolTableIsTheModelsOwn(t *testing.T) {
	path := filepath.Join(modelfilestest.Require(t), modelfiles.TokenizerFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var file tokenizerFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	if len(file.Model.Vocab) == 0 {
		t.Fatalf("%s gives no symbols", path)
	}

	table := speech.Symbols()
	boundaryTokens, err := speech.Tokens("")
	if err != nil {
		t.Fatalf("Tokens: %v", err)
	}
	for key, number := range file.Model.Vocab {
		if key == boundaryKey {
			if boundaryTokens[0] != number {
				t.Errorf("the boundary is %d where %s gives %d", boundaryTokens[0], path, number)
			}
			continue
		}
		symbol, size := utf8.DecodeRuneInString(key)
		if size != len(key) {
			t.Errorf("%s gives %q, which is not one symbol", path, key)
			continue
		}
		if held, ok := table[symbol]; !ok {
			t.Errorf("the table lacks %q, which %s gives as %d", symbol, path, number)
		} else if held != number {
			t.Errorf("the table reads %q as %d where %s gives %d", symbol, held, path, number)
		}
	}
	for symbol, number := range table {
		if _, ok := file.Model.Vocab[string(symbol)]; !ok {
			t.Errorf("the table holds %q as %d, which %s does not give", symbol, number, path)
		}
	}
}
