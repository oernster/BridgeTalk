package config_test

// FR-532 and FR-533: the saved speech sounds embedded beside the script, read strictly and
// written back the same.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
)

// The shipped script carries saved speech sounds for every line, in each accent.
func TestTheShippedScriptIsVoiced(t *testing.T) {
	t.Parallel()
	voiced, err := config.LoadVoicedScript(shippedTable(t))
	if err != nil {
		t.Fatalf("loading the shipped script with its sounds: %v", err)
	}
	for _, accent := range machinevoice.Accents() {
		if sounds, ok := voiced.Sounds("Docked", accent); !ok || len(sounds) != 3 {
			t.Errorf("Docked holds %d %s speech sounds, want 3", len(sounds), accent)
		}
	}
}

// FR-550's acceptance over the shipped files: "Breathable atmosphere, commander." is saved joined
// in each accent, while "Sold, commander. Credits are in." keeps its comma and is not joined.
func TestTheShippedScriptJoinsAFinalCommanderAndNoOther(t *testing.T) {
	t.Parallel()
	voiced, err := config.LoadVoicedScript(shippedTable(t))
	if err != nil {
		t.Fatalf("loading the shipped script with its sounds: %v", err)
	}
	for accent, want := range map[machinevoice.Accent]string{
		machinevoice.British:  "bɹˈiːðəbᵊl ˈatməsfɪəkəmˈɑndə.",
		machinevoice.American: "bɹˈiðəbᵊl ˈætməsfˌɪɹkəmˈændəɹ.",
	} {
		found := false
		for _, joined := range voiced.Joined(accent) {
			lines, _ := voiced.Lines(joined.Cue)
			switch lines[joined.Index].Text() {
			case "Breathable atmosphere, commander.":
				found = joined.Cue == "BreathableAtmosphere.Set" && joined.Sounds == want
			case "Sold, commander. Credits are in.":
				t.Errorf("%s joined %q", accent, lines[joined.Index].Text())
			}
		}
		if !found {
			t.Errorf("%s does not join BreathableAtmosphere.Set as %q", accent, want)
		}
	}
	lines, _ := voiced.Lines("ShipyardSell")
	sounds, _ := voiced.Sounds("ShipyardSell", machinevoice.British)
	for index, line := range lines {
		if line.Text() == "Sold, commander. Credits are in." && !strings.Contains(sounds[index], ", kəmˈɑndə.") {
			t.Errorf("ShipyardSell's British sounds %q lost the comma before commander", sounds[index])
		}
	}
}

// Saved speech sounds are written as a file that reads back the same, an id holding a dot
// included. The file says it is the sounds tool's to write.
func TestSavedSoundsAreWrittenAsAFileThatReadsBack(t *testing.T) {
	t.Parallel()
	saved := map[string]script.Saved{"Cast.Confirmed": {
		Lines:    []string{"Voice cast."},
		British:  []string{"vˈYs kˈɑːst."},
		American: []string{"vˈYs kˈæst."},
	}}
	written, err := config.EncodeSounds(saved)
	if err != nil {
		t.Fatalf("EncodeSounds: %v", err)
	}
	if !strings.HasPrefix(string(written), "#") || !strings.Contains(string(written), "go run ./tools/sounds") {
		t.Errorf("the file does not say how it is made:\n%s", written)
	}
	back, err := config.ParseSounds(written)
	if err != nil {
		t.Fatalf("ParseSounds: %v", err)
	}
	want, got := saved["Cast.Confirmed"], back["Cast.Confirmed"]
	if !slices.Equal(got.Lines, want.Lines) || !slices.Equal(got.British, want.British) ||
		!slices.Equal(got.American, want.American) {
		t.Errorf("read back %+v, want %+v", got, want)
	}
}

// A key the saved sounds' shape does not hold is refused rather than dropped in silence.
func TestSavedSoundsWithAKeyOutsideTheirShapeAreRefused(t *testing.T) {
	t.Parallel()
	raw := []byte("[sounds.\"Docked\"]\nline = [\"Docked.\"]\n")
	if _, err := config.ParseSounds(raw); !errors.Is(err, tomlfile.ErrUnknownKey) {
		t.Errorf("got %v, want ErrUnknownKey", err)
	}
}

// A script that cannot be read is refused before its sounds are read.
func TestAVoicedScriptWhoseScriptCannotBeReadIsRefused(t *testing.T) {
	t.Parallel()
	if _, err := config.VoiceScript([]byte("[lines\n"), nil, shippedTable(t)); err == nil {
		t.Error("a script that is not TOML was voiced")
	}
}

// Saved sounds that cannot be read stop the script being voiced.
func TestAVoicedScriptWhoseSoundsCannotBeReadIsRefused(t *testing.T) {
	t.Parallel()
	raw := []byte("[lines]\n\"Docked\" = [\"a\", \"b\", \"c\"]\n")
	if _, err := config.VoiceScript(raw, []byte("[sounds\n"), shippedTable(t)); err == nil {
		t.Error("sounds that are not TOML voiced a script")
	}
}

// Saved sounds that are not TOML are refused.
func TestSavedSoundsThatAreNotTomlAreRefused(t *testing.T) {
	t.Parallel()
	if _, err := config.ParseSounds([]byte("[sounds\n")); err == nil {
		t.Error("a file that is not TOML was accepted")
	}
}
