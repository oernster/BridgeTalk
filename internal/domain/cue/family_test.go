package cue_test

// Station traffic (FR-637, FR-638): a comms moment naming the beginnings of key stems answers every
// stem in that family, is narrower than a cue naming no key and broader than one naming a whole stem.

import (
	"errors"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// stationTraffic is the family moment as the shipped table writes it.
var stationTraffic = cue.Definition{
	ID: "ReceiveText.StationTraffic", Source: "journal", Event: "ReceiveText",
	Begins: map[string][]string{"Message": {"STATION_", "DockingChatter_", "DockingFailed_"}},
}

// FR-638's acceptance: three station traffic keys reach the family moment, a pirate key reaches its own
// moment and every other message, a key from another family or no key at all, reaches the channel cue.
// The family is written last so the order it resolves in is decided by narrowness alone.
func TestStationTrafficResolvesToItsOwnMoment(t *testing.T) {
	t.Parallel()
	table := cue.NewTable(append(messageTable(t).All(), mustCue(t, stationTraffic)))

	for _, key := range []string{"$STATION_docking_granted;", "$DockingChatter_Cordial;", "$STATION_NoFireZone_exited;"} {
		if got := resolvedID(table, message(key)); got != "ReceiveText.StationTraffic" {
			t.Errorf("%s resolved to %q, want the station traffic moment", key, got)
		}
	}
	if got := resolvedID(table, message("$Pirate_OnDeclarePiracyAttack07;")); got != "ReceiveText.Pirate.OnDeclarePiracyAttack" {
		t.Errorf("a pirate key resolved to %q, want its own moment", got)
	}
	for name, value := range map[string]any{"another family": "$Military_Patrol01;", "typed": "o7", "not text": float64(7)} {
		if got := resolvedID(table, message(value)); got != "ReceiveText.Channel.npc" {
			t.Errorf("a %s message resolved to %q, want the channel cue", name, got)
		}
	}
}

// FR-638's note: a moment naming a whole key stem is narrower than the family its stem belongs to, so it
// answers its own stem while the family keeps the rest, whichever of the two the table writes first.
func TestAWholeKeyStemMomentIsNarrowerThanStationTraffic(t *testing.T) {
	t.Parallel()
	granted := mustCue(t, cue.Definition{
		ID: "ReceiveText.STATION.docking.granted", Source: "journal", Event: "ReceiveText",
		Stem: map[string]string{"Message": "STATION_docking_granted"},
	})
	table := cue.NewTable([]cue.Cue{mustCue(t, stationTraffic), granted})

	if got := resolvedID(table, message("$STATION_docking_granted;")); got != granted.ID() {
		t.Errorf("the granted key resolved to %q, want the moment naming its stem", got)
	}
	if got := resolvedID(table, message("$STATION_NoFireZone_entered;")); got != "ReceiveText.StationTraffic" {
		t.Errorf("another station key resolved to %q, want the station traffic moment", got)
	}
}

// A family moment constrains one field, so it counts as one towards how specific it is.
func TestAFamilyMomentConstrainsOneField(t *testing.T) {
	t.Parallel()
	if got := mustCue(t, stationTraffic).Specificity(); got != 1 {
		t.Errorf("Specificity() = %d, want 1", got)
	}
}

// A family moment names one field's beginnings and nothing beside them, each one a key stem could start
// with. Anything else is refused.
func TestAFamilyMomentWrittenWronglyIsRefused(t *testing.T) {
	t.Parallel()
	station := map[string][]string{"Message": {"STATION_"}}
	cases := map[string]cue.Definition{
		"two fields": {ID: "ReceiveText.StationTraffic", Source: "journal", Event: "ReceiveText",
			Begins: map[string][]string{"Message": {"STATION_"}, "From": {"STATION_"}}},
		"a key stem beside it": {ID: "ReceiveText.StationTraffic", Source: "journal", Event: "ReceiveText",
			Begins: station, Stem: map[string]string{"Message": "Pirate_Arrival"}},
		"a match field beside it": {ID: "ReceiveText.StationTraffic", Source: "journal", Event: "ReceiveText",
			Begins: station, Match: map[string]string{"Channel": "npc"}},
		"no beginning listed": {ID: "ReceiveText.StationTraffic", Source: "journal", Event: "ReceiveText",
			Begins: map[string][]string{"Message": {}}},
		"a variant number": {ID: "ReceiveText.StationTraffic", Source: "journal", Event: "ReceiveText",
			Begins: map[string][]string{"Message": {"STATION1"}}},
		"an empty beginning": {ID: "ReceiveText.StationTraffic", Source: "journal", Event: "ReceiveText",
			Begins: map[string][]string{"Message": {""}}},
	}
	for name, definition := range cases {
		if _, err := cue.New(definition); !errors.Is(err, cue.ErrInvalidCue) {
			t.Errorf("%s: err = %v, want a refusal", name, err)
		}
	}
}
