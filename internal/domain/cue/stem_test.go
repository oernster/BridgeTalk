package cue_test

// Comms moments (FR-617 to FR-619): a message the game sends is matched by its key stem, whatever
// its variant number, its values or the words the game generated for it.

import (
	"errors"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
)

func TestAKeyStemIsTheKeyWithoutItsVariantItsValuesOrItsMarks(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		stem  string
		keyed bool
	}{
		"$Pirate_OnDeclarePiracyAttack07;":                                      {"Pirate_OnDeclarePiracyAttack", true},
		"$Pirate_ThreatenSpecific01:#units=20:#CommodityName=$aluminium_Name;;": {"Pirate_ThreatenSpecific", true},
		"$STATION_docking_granted;":                                             {"STATION_docking_granted", true},
		"$Pirate_Arrival":                                                       {"Pirate_Arrival", true},
		"o7":                                                                    {"", false},
		"$;":                                                                    {"", false},
		"$12;":                                                                  {"", false},
		"":                                                                      {"", false},
	}
	for key, want := range cases {
		stem, keyed := cue.KeyStem(key)
		if stem != want.stem || keyed != want.keyed {
			t.Errorf("KeyStem(%q) = %q, %t; want %q, %t", key, stem, keyed, want.stem, want.keyed)
		}
	}
}

// messageTable holds the channel cue before the comms moment, so what resolves is decided by the
// key stem rather than by the order the table was written in.
func messageTable(t *testing.T) cue.Table {
	t.Helper()
	return cue.NewTable([]cue.Cue{
		mustCue(t, cue.Definition{
			ID: "ReceiveText.Channel.npc", Source: "journal", Event: "ReceiveText",
			Match: map[string]string{"Channel": "npc"},
		}),
		mustCue(t, cue.Definition{
			ID: "ReceiveText.Pirate.OnDeclarePiracyAttack", Source: "journal", Event: "ReceiveText",
			Stem: map[string]string{"Message": "Pirate_OnDeclarePiracyAttack"},
		}),
	})
}

// message is a ReceiveText event on the npc channel carrying message as its key.
func message(message any) event.Event {
	return event.New(event.SourceJournal, "ReceiveText", event.EdgeNone,
		map[string]any{"Channel": "npc", "Message": message}, at)
}

// resolvedID names the cue an event resolves to; empty where none does.
func resolvedID(table cue.Table, candidate event.Event) cue.ID {
	resolved, _ := table.Resolve(candidate)
	return resolved.ID()
}

// FR-617's acceptance: two variants of one key reach the comms moment; a longer key with the same
// beginning does not, so the channel cue answers it.
func TestACommsMomentAnswersEveryVariantOfItsKey(t *testing.T) {
	t.Parallel()
	table := messageTable(t)

	for _, key := range []string{"$Pirate_OnDeclarePiracyAttack07;", "$Pirate_OnDeclarePiracyAttack12;"} {
		if got := resolvedID(table, message(key)); got != "ReceiveText.Pirate.OnDeclarePiracyAttack" {
			t.Errorf("%s resolved to %q, want the comms moment", key, got)
		}
	}
	if got := resolvedID(table, message("$Pirate_OnDeclarePiracyAttacker01;")); got != "ReceiveText.Channel.npc" {
		t.Errorf("a longer key resolved to %q, want the channel cue", got)
	}
}

// FR-618: a message with no key, as a player types it, reaches no comms moment; neither does a
// message field that holds no text at all.
func TestAMessageWithNoKeyReachesNoCommsMoment(t *testing.T) {
	t.Parallel()
	table := messageTable(t)

	for name, value := range map[string]any{"typed": "o7", "not text": float64(7)} {
		if got := resolvedID(table, message(value)); got != "ReceiveText.Channel.npc" {
			t.Errorf("a %s message resolved to %q, want the channel cue", name, got)
		}
	}
}

// FR-619: a comms moment's id is its event followed by its stem in dots, it names one stem and no
// match field beside it and the stem is one a key could have. Anything else is refused.
func TestACommsMomentThatIsWrittenWronglyIsRefused(t *testing.T) {
	t.Parallel()
	pirate := map[string]string{"Message": "Pirate_Arrival"}
	cases := map[string]cue.Definition{
		"id spelled otherwise": {ID: "ReceiveText.Pirate", Source: "journal", Event: "ReceiveText", Stem: pirate},
		"two stems": {ID: "ReceiveText.Pirate.Arrival", Source: "journal", Event: "ReceiveText",
			Stem: map[string]string{"Message": "Pirate_Arrival", "From": "Pirate_Arrival"}},
		"a match field beside it": {ID: "ReceiveText.Pirate.Arrival", Source: "journal", Event: "ReceiveText",
			Stem: pirate, Match: map[string]string{"Channel": "npc"}},
		"a variant number": {ID: "ReceiveText.Pirate.Arrival01", Source: "journal", Event: "ReceiveText",
			Stem: map[string]string{"Message": "Pirate_Arrival01"}},
		"no stem at all": {ID: "ReceiveText.", Source: "journal", Event: "ReceiveText",
			Stem: map[string]string{"Message": ""}},
	}
	for name, definition := range cases {
		if _, err := cue.New(definition); !errors.Is(err, cue.ErrInvalidCue) {
			t.Errorf("%s: err = %v, want a refusal", name, err)
		}
	}
}
