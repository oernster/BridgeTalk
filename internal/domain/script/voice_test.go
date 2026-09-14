package script_test

// FR-506 and FR-533: saved speech sounds checked against the script they were made from.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// docked builds a script giving Docked the given lines, against a table that also holds Undocked.
func docked(t *testing.T, lines []string) script.Script {
	t.Helper()
	built, err := script.New(map[string][]string{"Docked": lines}, tableOf(t, "Docked", "Undocked"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return built
}

// savedFor is well formed saved speech sounds for three lines.
func savedFor(lines []string) script.Saved {
	return script.Saved{
		Lines:    slices.Clone(lines),
		British:  []string{"dˈɒkɪŋ", "wɪə dˈWn", "dˈɒkt"},
		American: []string{"dˈɑkɪŋ", "wɪɹ dˈWn", "dˈɑkt"},
	}
}

// voiceRefused asserts that voicing fails as invalid, naming every fragment given.
func voiceRefused(t *testing.T, built script.Script, saved map[string]script.Saved, fragments ...string) error {
	t.Helper()
	_, err := script.Voice(built, saved)
	if !errors.Is(err, script.ErrInvalidScript) {
		t.Fatalf("Voice = %v, want ErrInvalidScript", err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("error %q does not name %q", err, fragment)
		}
	}
	return err
}

// FR-532: a cue's saved speech sounds are handed out for each accent, in the order of its lines.
func TestSavedSoundsAreGivenForEachAccent(t *testing.T) {
	saved := savedFor(three)
	voiced, err := script.Voice(docked(t, three), map[string]script.Saved{"Docked": saved})
	if err != nil {
		t.Fatalf("Voice: %v", err)
	}
	for accent, want := range map[machinevoice.Accent][]string{
		machinevoice.British:  saved.British,
		machinevoice.American: saved.American,
	} {
		if got, ok := voiced.Sounds("Docked", accent); !ok || !slices.Equal(got, want) {
			t.Errorf("%s sounds = %q, %v; want %q", accent, got, ok, want)
		}
	}
	if _, ok := voiced.Sounds("Undocked", machinevoice.British); ok {
		t.Error("Undocked has sounds the script never gave it")
	}
}

// FR-533: a cue the script gives lines with no saved speech sounds is refused, naming it.
func TestACueWithoutSavedSoundsIsRefused(t *testing.T) {
	voiceRefused(t, docked(t, three), map[string]script.Saved{}, `"Docked"`, "no saved speech sounds")
}

// FR-533: saved speech sounds for a cue the script no longer holds are refused, naming it.
func TestSoundsLeftForAGoneCueAreRefused(t *testing.T) {
	voiceRefused(t, docked(t, three), map[string]script.Saved{"Docked": savedFor(three), "Undocked": savedFor(three)},
		`"Undocked"`, "no longer holds")
}

// FR-533's acceptance: a line edited without making its sounds again is refused, naming the
// cue and the line; the old line's sounds are named as left behind.
func TestSoundsMadeFromOtherTextAreRefusedNamingTheLine(t *testing.T) {
	edited := []string{"Docked.", three[1], three[2]}
	voiceRefused(t, docked(t, edited), map[string]script.Saved{"Docked": savedFor(three)},
		`"Docked"`, `"Docked."`, "made from its text", `"Docking complete."`, "no longer holds")
}

// FR-533: an accent short of sounds for a cue's lines is refused, naming the accent.
func TestSoundsMissingForAnAccentAreRefused(t *testing.T) {
	saved := savedFor(three)
	saved.American = saved.American[:2]
	voiceRefused(t, docked(t, three), map[string]script.Saved{"Docked": saved}, `"Docked"`, "American", "3 lines")
}

// FR-506: saved speech sounds holding a symbol the model cannot read are refused, as are too many
// of them, naming the cue, the line and the accent.
func TestSoundsTheModelCannotTakeAreRefused(t *testing.T) {
	unreadable := savedFor(three)
	unreadable.British[2] = "dʘ"
	err := voiceRefused(t, docked(t, three), map[string]script.Saved{"Docked": unreadable},
		`"Docked"`, `"Docked and secure, commander."`, "British")
	if !errors.Is(err, speech.ErrUnreadableSymbol) {
		t.Errorf("got %v, want ErrUnreadableSymbol as well", err)
	}
	long := savedFor(three)
	long.American[0] = strings.Repeat("ə", 511)
	if err := voiceRefused(t, docked(t, three), map[string]script.Saved{"Docked": long}); !errors.Is(err, speech.ErrTooLong) {
		t.Errorf("got %v, want ErrTooLong as well", err)
	}
}

// The sounds handed out are a copy, so no caller can change the script.
func TestTheSoundsHandedOutAreTheCallersOwn(t *testing.T) {
	voiced, err := script.Voice(docked(t, three), map[string]script.Saved{"Docked": savedFor(three)})
	if err != nil {
		t.Fatalf("Voice: %v", err)
	}
	sounds, _ := voiced.Sounds("Docked", machinevoice.British)
	sounds[0] = ""
	if again, _ := voiced.Sounds("Docked", machinevoice.British); again[0] == "" {
		t.Error("changing the returned sounds changed the script")
	}
}

// The cues a script gives lines, sorted, so the sounds tool writes them in a stable order.
func TestCuesListsTheScriptsCuesSorted(t *testing.T) {
	built, err := script.New(map[string][]string{"Undocked": three, "Docked": three}, tableOf(t, "Docked", "Undocked"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got, want := built.Cues(), []cue.ID{"Docked", "Undocked"}; !slices.Equal(got, want) {
		t.Errorf("Cues = %v, want %v", got, want)
	}
}
