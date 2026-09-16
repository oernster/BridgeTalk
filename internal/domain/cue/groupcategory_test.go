package cue_test

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// FR-749: a group belongs to the category holding most of its cues, the earlier category on a tie,
// and to none where none of its cues has one.
func TestAGroupBelongsToTheCategoryHoldingMostOfItsMoments(t *testing.T) {
	t.Parallel()

	var cues []cue.Cue
	for _, each := range []cue.Definition{
		{ID: "ReceiveText.Channel.npc", Source: "journal", Event: "ReceiveText", Purpose: "A message.", Category: "Comms"},
		{ID: "ReceiveText.Channel.player", Source: "journal", Event: "ReceiveText", Purpose: "A message.", Category: "Comms"},
		{ID: "ReceiveText.Channel.station", Source: "journal", Event: "ReceiveText", Purpose: "Traffic.", Category: "Docking and stations"},
		{ID: "Docked", Source: "journal", Event: "Docked", Purpose: "Docked.", Category: "Docking and stations"},
		{ID: "Scan.Kind.star", Source: "journal", Event: "Scan", Purpose: "A star.", Category: "Exploration"},
		{ID: "Scan.Kind.ship", Source: "journal", Event: "Scan", Purpose: "A ship.", Category: "Comms"},
		{ID: "Cast.Confirmed", Source: "application", Event: "cast", Purpose: "Cast."},
	} {
		built, err := cue.New(each)
		if err != nil {
			t.Fatalf("building %s: %v", each.ID, err)
		}
		cues = append(cues, built)
	}
	table, err := cue.NewCategorisedTable([]string{"Docking and stations", "Comms", "Exploration"}, cues)
	if err != nil {
		t.Fatalf("building the table: %v", err)
	}

	for group, want := range map[string]string{
		"ReceiveText": "Comms",
		"Docked":      "Docking and stations",
		"Scan":        "Comms",
		"Cast":        "",
		"UnderAttack": "",
	} {
		if got := table.GroupCategory(group); got != want {
			t.Errorf("GroupCategory(%q) = %q, want %q", group, got, want)
		}
	}
}
