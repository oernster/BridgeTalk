package config_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
)

// TestShippedTableResolvesRealSynthesisPayloads guards the cue table as a shipped
// contract. The Name values are the ones real journals carry, so a change that renamed
// or dropped these cues would be caught here rather than in the game. A name the table
// does not narrow falls to the event's own cue.
func TestShippedTableResolvesRealSynthesisPayloads(t *testing.T) {
	t.Parallel()
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("loading the shipped table: %v", err)
	}
	cases := map[string]string{
		"Repair Basic":        "Synthesis.Name.Repair Basic",
		"Fuel Basic":          "Synthesis.Name.Fuel Basic",
		"Ammo Basic":          "Synthesis.Name.Ammo Basic",
		"A recipe never seen": "Synthesis",
	}
	for name, want := range cases {
		candidate := event.New(
			event.SourceJournal, "Synthesis", event.EdgeNone,
			map[string]any{"Name": name}, time.Unix(0, 0),
		)
		resolved, ok := table.Resolve(candidate)
		if !ok {
			t.Errorf("Synthesis %q resolved to nothing", name)
			continue
		}
		if string(resolved.ID()) != want {
			t.Errorf("Synthesis %q resolved to %q, want %q", name, resolved.ID(), want)
		}
	}
}

// gameSpelling is the shape of every shipped id: segments of letters, digits and
// spaces, each beginning with a letter, joined by dots.
var gameSpelling = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9 ]*(\.[A-Za-z][A-Za-z0-9 ]*)*$`)

// Every shipped id is spelled the game's way: its first segment is the journal event or
// status value it listens for, so the group a reader sees is the game's own name for the
// moment. The application's own acknowledgement is the one cue with no such name.
func TestEveryShippedIdBeginsWithWhatItListensFor(t *testing.T) {
	t.Parallel()
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("loading the shipped table: %v", err)
	}
	for _, item := range table.All() {
		id := string(item.ID())
		if !gameSpelling.MatchString(id) {
			t.Errorf("%q is not spelled as segments of the game's words", id)
		}
		if item.Source() == event.SourceApplication {
			continue
		}
		if item.ID().Group() != item.Name() {
			t.Errorf("%q listens for %q, so its first segment should name that", id, item.Name())
		}
	}
}
