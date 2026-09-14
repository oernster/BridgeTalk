package config

import (
	_ "embed"
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
)

//go:embed script.toml
var embeddedScript []byte

// scriptFile is the shape of script.toml: one [lines] table whose keys are cue ids, each
// holding that cue's lines (FR-503). An id holding a dot is quoted, since TOML would otherwise
// read it as a table inside a table.
type scriptFile struct {
	Lines map[string][]string `toml:"lines"`
}

// LoadScript reads the shipped script against the cue table (FR-503).
func LoadScript(table cue.Table) (script.Script, error) {
	return parseScript(embeddedScript, table)
}

// parseScript reads a script's bytes against the cue table, refusing a key its shape does not
// hold.
func parseScript(raw []byte, table cue.Table) (script.Script, error) {
	var parsed scriptFile
	if err := tomlfile.Decode(raw, &parsed); err != nil {
		return script.Script{}, fmt.Errorf("parsing script: %w", err)
	}
	return script.New(parsed.Lines, table)
}
