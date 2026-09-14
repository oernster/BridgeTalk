package config_test

// FR-503: the script embedded beside the cue table, read strictly.

import (
	"errors"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/speech"
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

// FR-549: the shipped script gives commander its British and American speech sounds.
func TestTheShippedScriptGivesCommandersSounds(t *testing.T) {
	t.Parallel()
	loaded, err := config.LoadScript(shippedTable(t))
	if err != nil {
		t.Fatalf("loading the shipped script: %v", err)
	}
	for accent, want := range map[machinevoice.Accent]string{machinevoice.British: "kəmˈɑndə", machinevoice.American: "kəmˈændəɹ"} {
		if got, ok := loaded.Words().Sounds("commander", accent); !ok || got != want {
			t.Errorf("%s commander = %q, %v; want %q", accent, got, ok, want)
		}
	}
}

// FR-549 and FR-550: a broken line, a broken word and a joined word without sounds are refused
// together, each named.
func TestAScriptsBrokenLinesWordsAndJoinsAreRefusedTogether(t *testing.T) {
	t.Parallel()
	raw := []byte("[lines]\n\"Docked\" = [\"a\", \"b\"]\n" +
		"[words]\ncommander = [\"kʘm\"]\n" +
		"[joins]\nafter_comma = [\"commander\"]\n")
	_, err := config.ParseScript(raw, shippedTable(t))
	if !errors.Is(err, script.ErrInvalidScript) || !errors.Is(err, speech.ErrBrokenSpelling) {
		t.Fatalf("got %v, want ErrInvalidScript and ErrBrokenSpelling", err)
	}
	for _, fragment := range []string{`"Docked"`, `"commander"`} {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("error %q does not name %s", err, fragment)
		}
	}
	raw = []byte("[lines]\n[words]\ncommander = [\"kəmˈɑndə\"]\n[joins]\nafter_comma = [\"pilot\"]\n")
	if _, err := config.ParseScript(raw, shippedTable(t)); !errors.Is(err, script.ErrInvalidScript) ||
		!strings.Contains(err.Error(), `"pilot"`) {
		t.Errorf("got %v, want ErrInvalidScript naming pilot", err)
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
