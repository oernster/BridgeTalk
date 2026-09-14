package script_test

// FR-503 to FR-505, FR-507 and FR-531: a script checked against the cue table, naming the cue
// and the line behind every problem.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// tableOf builds a cue table holding a journal cue for each id.
func tableOf(t *testing.T, ids ...string) cue.Table {
	t.Helper()
	cues := make([]cue.Cue, 0, len(ids))
	for _, id := range ids {
		item, err := cue.New(cue.Definition{ID: id, Source: "journal", Event: id, Purpose: "When it happens."})
		if err != nil {
			t.Fatalf("building cue %q: %v", id, err)
		}
		cues = append(cues, item)
	}
	return cue.NewTable(cues)
}

// three is a well formed set of lines for one cue.
var three = []string{"Docking complete.", "We're down safely.", "Docked and secure, commander."}

// refused asserts that building a script fails as invalid, naming every fragment given.
func refused(t *testing.T, entries map[string][]string, table cue.Table, fragments ...string) error {
	t.Helper()
	_, err := script.New(entries, table)
	if !errors.Is(err, script.ErrInvalidScript) {
		t.Fatalf("New(%v) = %v, want ErrInvalidScript", entries, err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("error %q does not name %q", err, fragment)
		}
	}
	return err
}

// FR-503: a cue's lines are the ones the script gives it, in order.
func TestAScriptHoldsTheLinesItIsGiven(t *testing.T) {
	built, err := script.New(map[string][]string{"Docked": three}, tableOf(t, "Docked"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	lines, ok := built.Lines("Docked")
	if !ok {
		t.Fatal("Docked has no lines")
	}
	var words []string
	for _, line := range lines {
		words = append(words, line.Words())
	}
	if !slices.Equal(words, three) {
		t.Errorf("lines = %q, want %q", words, three)
	}
}

// A cue the script does not name has no lines.
func TestACueTheScriptDoesNotNameHasNoLines(t *testing.T) {
	built, err := script.New(map[string][]string{"Docked": three}, tableOf(t, "Docked", "Undocked"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := built.Lines("Undocked"); ok {
		t.Error("Undocked has lines the script never gave it")
	}
}

// FR-504's acceptance: a key that is not a cue id is refused, naming it.
func TestAKeyThatIsNotACueIsRefusedNamingIt(t *testing.T) {
	refused(t, map[string][]string{"Dockd": three}, tableOf(t, "Docked"), `"Dockd"`, "not a cue")
}

// FR-505's acceptance: a cue holding other than three lines is refused, naming it.
func TestACueWithoutThreeLinesIsRefused(t *testing.T) {
	table := tableOf(t, "Docked")
	refused(t, map[string][]string{"Docked": three[:2]}, table, `"Docked"`, "2 lines")
	refused(t, map[string][]string{"Docked": append(slices.Clone(three), "Down.")}, table, "4 lines")
}

// FR-505: none of a cue's lines may be empty, counting a line of spaces as empty.
func TestAnEmptyLineIsRefused(t *testing.T) {
	refused(t, map[string][]string{"Docked": {three[0], "  ", three[2]}}, tableOf(t, "Docked"),
		`"Docked"`, "empty line")
}

// FR-505: a cue's three lines are distinct.
func TestARepeatedLineIsRefused(t *testing.T) {
	refused(t, map[string][]string{"Docked": {three[0], three[0], three[2]}}, tableOf(t, "Docked"),
		`"Docked"`, `"Docking complete."`, "twice")
}

// FR-531: a broken spelling is refused, naming the cue and the line; the spelling's own reason
// stays reachable.
func TestABrokenSpellingIsRefusedNamingTheCueAndTheLine(t *testing.T) {
	broken := "Flight [record](/ˈɹɛkɔːd) saved."
	err := refused(t, map[string][]string{"Docked": {three[0], three[1], broken}}, tableOf(t, "Docked"),
		`"Docked"`, broken)
	if !errors.Is(err, speech.ErrBrokenSpelling) {
		t.Errorf("got %v, want ErrBrokenSpelling as well", err)
	}
}

// Every problem is reported, not just the first, so one run names everything to put right.
func TestEveryProblemIsReported(t *testing.T) {
	refused(t, map[string][]string{"Dockd": three, "Docked": three[:2]}, tableOf(t, "Docked"),
		`"Dockd"`, `"Docked"`)
}

// FR-507: the cues the table holds that the script gives no lines, in the table's order.
func TestMissingNamesTheCuesWithNoLines(t *testing.T) {
	table := tableOf(t, "Undocked", "Docked", "Touchdown")
	built, err := script.New(map[string][]string{"Docked": three}, table)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got, want := built.Missing(table), []cue.ID{"Undocked", "Touchdown"}; !slices.Equal(got, want) {
		t.Errorf("Missing = %v, want %v", got, want)
	}
}

// The lines handed out are a copy, so no caller can change the script.
func TestTheLinesHandedOutAreTheCallersOwn(t *testing.T) {
	built, err := script.New(map[string][]string{"Docked": three}, tableOf(t, "Docked"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	lines, _ := built.Lines("Docked")
	lines[0] = speech.Line{}
	again, _ := built.Lines("Docked")
	if again[0].Words() != three[0] {
		t.Error("changing the returned lines changed the script")
	}
}
