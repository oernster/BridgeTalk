package script_test

// FR-549 and FR-550: the script's table of words and the words it joins after a final comma,
// refused naming the word, joined in the saved speech sounds and listed for what comes after.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// commanderSounds is the table of words the shipped script gives: commander, British then American.
var commanderSounds = map[string][]string{"commander": {"kəmˈɑndə", "kəmˈændəɹ"}}

// joining builds a script for Docked and Undocked that gives commander's sounds and joins it.
func joining(t *testing.T, entries map[string][]string) script.Script {
	t.Helper()
	built, err := script.New(entries, tableOf(t, "Docked", "Undocked"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	spoken, err := built.WithWords(commanderSounds, []string{"commander"})
	if err != nil {
		t.Fatalf("WithWords: %v", err)
	}
	return spoken
}

// line reads a line the test expects to be well formed.
func line(t *testing.T, text string) speech.Line {
	t.Helper()
	read, err := speech.Read(text)
	if err != nil {
		t.Fatalf("Read(%q): %v", text, err)
	}
	return read
}

// FR-549: a script gives the table of words it was built with; the lines stay as they were.
func TestAScriptGivesItsTableOfWords(t *testing.T) {
	spoken := joining(t, map[string][]string{"Docked": three})
	if got, ok := spoken.Words().Sounds("commander", machinevoice.American); !ok || got != "kəmˈændəɹ" {
		t.Errorf("American commander = %q, %v; want kəmˈændəɹ", got, ok)
	}
	if lines, ok := spoken.Lines("Docked"); !ok || len(lines) != script.LinesPerCue {
		t.Errorf("Docked holds %d lines, want %d", len(lines), script.LinesPerCue)
	}
}

// FR-549: a table of words that cannot be read refuses the script, naming the word.
func TestABrokenTableOfWordsIsRefusedNamingTheWord(t *testing.T) {
	built := docked(t, three)
	_, err := built.WithWords(map[string][]string{"commander": {"kʘm"}}, nil)
	if !errors.Is(err, script.ErrInvalidScript) || !errors.Is(err, speech.ErrBrokenSpelling) ||
		!strings.Contains(err.Error(), `"commander"`) {
		t.Errorf("got %v, want ErrInvalidScript and ErrBrokenSpelling naming commander", err)
	}
}

// FR-550: every word joined after a comma must have its speech sounds in the table, since they are
// how it is found in the saved sounds; each one missing is refused by name.
func TestAWordJoinedWithoutSoundsInTheTableIsRefusedNamingIt(t *testing.T) {
	_, err := docked(t, three).WithWords(commanderSounds, []string{"commander", "pilot", "captain"})
	if !errors.Is(err, script.ErrInvalidScript) {
		t.Fatalf("got %v, want ErrInvalidScript", err)
	}
	for _, word := range []string{`"pilot"`, `"captain"`} {
		if !strings.Contains(err.Error(), word) {
			t.Errorf("error %q does not name %s", err, word)
		}
	}
	if strings.Contains(err.Error(), `"commander"`) {
		t.Errorf("error %q names commander, which the table holds", err)
	}
}

// FR-550's acceptance: a line ending with a comma, a joined word and one final mark has the comma
// and its space left out of its sounds in each accent, the word joined to the one before.
func TestAFinalWordAfterACommaIsJoinedToTheWordBefore(t *testing.T) {
	spoken := joining(t, map[string][]string{"Docked": three})
	for _, each := range []struct {
		text   string
		accent machinevoice.Accent
		sounds string
		want   string
	}{
		{"Breathable atmosphere, commander.", machinevoice.British, "bɹˈiːðəbᵊl ˈatməsfɪə, kəmˈɑndə.", "bɹˈiːðəbᵊl ˈatməsfɪəkəmˈɑndə."},
		{"Breathable atmosphere, commander.", machinevoice.American, "bɹˈiðəbᵊl ˈætməsfˌɪɹ, kəmˈændəɹ.", "bɹˈiðəbᵊl ˈætməsfˌɪɹkəmˈændəɹ."},
		{"Hard landing, commander!", machinevoice.British, "hˈɑːd lˈandɪŋ, kəmˈɑndə!", "hˈɑːd lˈandɪŋkəmˈɑndə!"},
		{"Ready, commander?", machinevoice.American, "ɹˈɛdi, kəmˈændəɹ?", "ɹˈɛdikəmˈændəɹ?"},
	} {
		got, err := spoken.Join(line(t, each.text), each.accent, each.sounds)
		if err != nil || got != each.want {
			t.Errorf("Join(%q, %s) = %q, %v; want %q", each.text, each.accent, got, err, each.want)
		}
	}
}

// FR-550: a line where the word follows a comma but does not end it, is not the joined word, has
// no comma before it or ends other than with one mark keeps its sounds as they are.
func TestALineThatDoesNotEndWithACommaAndAJoinedWordKeepsItsSounds(t *testing.T) {
	built := docked(t, three)
	spoken, err := built.WithWords(map[string][]string{"commander": {"kəmˈɑndə"}, "pilot": {"pˈaɪlət"}}, []string{"commander"})
	if err != nil {
		t.Fatalf("WithWords: %v", err)
	}
	sounds := "sˈQld, kəmˈɑndə."
	for _, text := range []string{
		"Sold, commander. Credits are in.", "Sold, pilot.", "Sold commander.", "Sold, commander",
		"Sold, commander..", "Sold, decommander.",
	} {
		if got, err := spoken.Join(line(t, text), machinevoice.British, sounds); err != nil || got != sounds {
			t.Errorf("Join(%q) = %q, %v; want the sounds unchanged", text, got, err)
		}
	}
}

// FR-550: sounds that do not end with the comma, the word's spelling and the mark cannot be
// joined, naming the sounds and the ending looked for, rather than being saved unjoined.
func TestSoundsThatDoNotEndAsTheTableSpellsThemCannotBeJoined(t *testing.T) {
	spoken := joining(t, map[string][]string{"Docked": three})
	sounds := "bɹˈiːðəbᵊl ˈatməsfɪə, kəmˈɑːndə."
	_, err := spoken.Join(line(t, "Breathable atmosphere, commander."), machinevoice.British, sounds)
	if !errors.Is(err, script.ErrUnjoinable) || !strings.Contains(err.Error(), sounds) ||
		!strings.Contains(err.Error(), ", kəmˈɑndə.") {
		t.Errorf("got %v, want ErrUnjoinable naming the sounds and the ending", err)
	}
}

// FR-550: every joining line is listed with its cue, its place and its saved sounds in the accent
// asked for, cue by cue then line by line.
func TestJoinedListsEveryJoiningLineInCueThenLineOrder(t *testing.T) {
	docked := []string{"Docked, commander.", "We're down safely.", "Down safe, commander!"}
	undocked := []string{"Clear, commander?", "Undocked.", "Leaving, commander. Out."}
	spoken := joining(t, map[string][]string{"Undocked": undocked, "Docked": docked})
	saved := map[string]script.Saved{
		"Docked":   {Lines: docked, British: []string{"dˈɒktkəmˈɑndə.", "wɪə dˈWn", "dˈWn sˈAfkəmˈɑndə!"}, American: []string{"dˈɑktkəmˈændəɹ.", "wɪɹ dˈWn", "dˈWn sˈAfkəmˈændəɹ!"}},
		"Undocked": {Lines: undocked, British: []string{"klˈɪəkəmˈɑndə?", "ʌndˈɒkt.", "lˈiːvɪŋ, kəmˈɑndə. ˈWt."}, American: []string{"klˈɪɹkəmˈændəɹ?", "ʌndˈɑkt.", "lˈivɪŋ, kəmˈændəɹ. ˈWt."}},
	}
	voiced, err := script.Voice(spoken, saved)
	if err != nil {
		t.Fatalf("Voice: %v", err)
	}
	want := []script.SavedLine{
		{Cue: "Docked", Index: 0, Sounds: "dˈɑktkəmˈændəɹ."},
		{Cue: "Docked", Index: 2, Sounds: "dˈWn sˈAfkəmˈændəɹ!"},
		{Cue: "Undocked", Index: 0, Sounds: "klˈɪɹkəmˈændəɹ?"},
	}
	if got := voiced.Joined(machinevoice.American); !slices.Equal(got, want) {
		t.Errorf("Joined(American) = %+v, want %+v", got, want)
	}
	if got := voiced.Joined(machinevoice.British); len(got) != len(want) || got[1].Sounds != "dˈWn sˈAfkəmˈɑndə!" {
		t.Errorf("Joined(British) = %+v, want British sounds for the same lines", got)
	}
}

// A script given no table of words joins nothing.
func TestAScriptWithoutATableOfWordsJoinsNothing(t *testing.T) {
	lines := []string{"Docked, commander.", three[1], three[2]}
	voiced, err := script.Voice(docked(t, lines), map[string]script.Saved{"Docked": savedFor(lines)})
	if err != nil {
		t.Fatalf("Voice: %v", err)
	}
	if got := voiced.Joined(machinevoice.British); len(got) != 0 {
		t.Errorf("Joined = %+v, want none", got)
	}
}
