package config_test

// FR-503: the script embedded beside the cue table, read strictly.

import (
	"errors"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
)

// shippedTable loads the cue table the application ships.
func shippedTable(t *testing.T) cue.Table {
	t.Helper()
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("loading the shipped table: %v", err)
	}
	return table
}

// The shipped script loads against the shipped table and gives Docked three lines.
func TestTheShippedScriptLoadsAgainstTheShippedTable(t *testing.T) {
	t.Parallel()
	loaded, err := config.LoadScript(shippedTable(t))
	if err != nil {
		t.Fatalf("loading the shipped script: %v", err)
	}
	if lines, ok := loaded.Lines("Docked"); !ok || len(lines) != 3 {
		t.Errorf("Docked holds %d lines, want 3", len(lines))
	}
}

// A file that is not TOML is refused.
func TestAScriptThatIsNotTomlIsRefused(t *testing.T) {
	t.Parallel()
	if _, err := config.ParseScript([]byte("[lines\n"), shippedTable(t)); err == nil {
		t.Error("a file that is not TOML was accepted")
	}
}

// A key the script's shape does not hold is refused rather than dropped in silence.
func TestAScriptWithAKeyOutsideItsShapeIsRefused(t *testing.T) {
	t.Parallel()
	raw := []byte("[line]\n\"Docked\" = [\"a\", \"b\", \"c\"]\n")
	if _, err := config.ParseScript(raw, shippedTable(t)); !errors.Is(err, tomlfile.ErrUnknownKey) {
		t.Errorf("got %v, want ErrUnknownKey", err)
	}
}

// An id holding a dot must be quoted: unquoted, TOML reads it as a table inside a table, which
// the script's shape refuses rather than reading as some other cue.
func TestAnUnquotedIdHoldingADotIsRefused(t *testing.T) {
	t.Parallel()
	raw := []byte("[lines]\nCast.Confirmed = [\"a\", \"b\", \"c\"]\n")
	if _, err := config.ParseScript(raw, shippedTable(t)); err == nil {
		t.Error("an unquoted id holding a dot was accepted")
	}
}
