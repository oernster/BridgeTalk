package config

import (
	_ "embed"
	"errors"
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
)

//go:embed script.toml
var embeddedScript []byte

// scriptFile is the shape of script.toml: a [lines] table whose keys are cue ids, each holding
// that cue's lines (FR-503); a [words] table giving words their speech sounds (FR-549); a [joins]
// table naming the words joined after a final comma (FR-550). An id holding a dot is quoted, since
// TOML would otherwise read it as a table inside a table.
type scriptFile struct {
	Lines map[string][]string `toml:"lines"`
	Words map[string][]string `toml:"words"`
	Joins scriptJoins         `toml:"joins"`
}

// scriptJoins is the shape of the [joins] table.
type scriptJoins struct {
	AfterComma []string `toml:"after_comma"`
}

// LoadScript reads the shipped script against the cue table (FR-503).
func LoadScript(table cue.Table) (script.Script, error) {
	return parseScript(embeddedScript, table)
}

// parseScript reads a script's bytes against the cue table, refusing a key its shape does not
// hold. A problem in the lines and one in the words are reported together.
func parseScript(raw []byte, table cue.Table) (script.Script, error) {
	var parsed scriptFile
	if err := tomlfile.Decode(raw, &parsed); err != nil {
		return script.Script{}, fmt.Errorf("parsing script: %w", err)
	}
	built, linesErr := script.New(parsed.Lines, table)
	spoken, wordsErr := built.WithWords(parsed.Words, parsed.Joins.AfterComma)
	if err := errors.Join(linesErr, wordsErr); err != nil {
		return script.Script{}, err
	}
	return spoken, nil
}
