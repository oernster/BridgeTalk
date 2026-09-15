package config_test

// Comms moments in the cue table (FR-619, FR-620).

import (
	"strings"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
)

// FR-619: a table whose comms moment spells its id other than its event and its stem in dots fails
// to load, naming the cue.
func TestACommsMomentWhoseIdDisagreesWithItsStemFailsToLoad(t *testing.T) {
	t.Parallel()
	path := overrideFile(t, "cues.toml", `
[[cue]]
id = "ReceiveText.Pirate"
purpose = "When a pirate arrives."
source = "journal"
event = "ReceiveText"
stem = { Message = "Pirate_Arrival" }
`)
	_, err := config.LoadCueTable(path)
	if err == nil || !strings.Contains(err.Error(), "ReceiveText.Pirate ") {
		t.Fatalf("err = %v, want a refusal naming the cue", err)
	}
}

// FR-620's acceptance: each pirate key stem, arriving in a variant, resolves to its own comms moment
// in the shipped table, carrying the priority FR-620 gives it and a purpose.
func TestTheShippedPirateMomentsAnswerTheirMessages(t *testing.T) {
	t.Parallel()
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("loading the shipped table: %v", err)
	}
	at := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	moments := map[string]string{
		"Pirate_Arrival":               "notice",
		"Pirate_OnStartScanCargo":      "notice",
		"Pirate_NotEnoughCargo":        "notice",
		"Pirate_OnNoCargoFound":        "notice",
		"Pirate_StartInterdiction":     "alert",
		"Pirate_OnDeclarePiracyAttack": "alert",
	}
	for stem, spelled := range moments {
		received := event.New(event.SourceJournal, "ReceiveText", event.EdgeNone,
			map[string]any{"Channel": "npc", "Message": "$" + stem + "03;"}, at)
		resolved, ok := table.Resolve(received)
		want := cue.ID("ReceiveText." + strings.ReplaceAll(stem, "_", "."))
		if !ok || resolved.ID() != want {
			t.Errorf("%s resolved to %q, want %q", stem, resolved.ID(), want)
			continue
		}
		priority, err := cue.ParsePriority(spelled)
		if err != nil {
			t.Fatalf("%s: %v", spelled, err)
		}
		if resolved.Priority() != priority {
			t.Errorf("%s has priority %v, want %s", want, resolved.Priority(), spelled)
		}
		if strings.TrimSpace(resolved.Purpose()) == "" {
			t.Errorf("%s has no purpose", want)
		}
	}
}
