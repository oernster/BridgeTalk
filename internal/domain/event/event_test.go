package event_test

import (
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/event"
)

var observed = time.Date(2026, 8, 26, 9, 30, 0, 0, time.UTC)

func TestEventCarriesWhatItWasGiven(t *testing.T) {
	built := event.New(
		event.SourceJournal,
		"Bounty",
		event.EdgeNone,
		map[string]any{"Reward": 4200.0, "VictimFaction": "Sirius"},
		observed,
	)

	if built.Source() != event.SourceJournal {
		t.Fatalf("source: got %q", built.Source())
	}
	if built.Name() != "Bounty" {
		t.Fatalf("name: got %q", built.Name())
	}
	if built.Edge() != event.EdgeNone {
		t.Fatalf("edge: got %q", built.Edge())
	}
	if !built.At().Equal(observed) {
		t.Fatalf("at: got %v, want %v", built.At(), observed)
	}
	if built.FieldCount() != 2 {
		t.Fatalf("field count: got %d, want 2", built.FieldCount())
	}
}

func TestEventReadsOneFieldAndReportsAMissingOne(t *testing.T) {
	built := event.New(event.SourceStatus, "Docked", event.EdgeRising,
		map[string]any{"Docked": true}, observed)

	value, ok := built.Field("Docked")
	if !ok || value != true {
		t.Fatalf("present field: got %v, %v", value, ok)
	}
	if _, ok := built.Field("Landed"); ok {
		t.Fatal("a field that was never set reported as present")
	}
}

// The constructor copies the caller's map. Without that copy an event could change
// under the cue engine after it was raised, which is the whole reason fields is
// unexported.
func TestEventDoesNotRetainTheCallersMap(t *testing.T) {
	fields := map[string]any{"Reward": 1.0}
	built := event.New(event.SourceJournal, "Bounty", event.EdgeNone, fields, observed)

	fields["Reward"] = 999.0
	fields["Added"] = "later"

	value, _ := built.Field("Reward")
	if value != 1.0 {
		t.Fatalf("field changed with the caller's map: got %v", value)
	}
	if built.FieldCount() != 1 {
		t.Fatalf("field count changed with the caller's map: got %d", built.FieldCount())
	}
}

func TestEventWithNoFieldsCountsNone(t *testing.T) {
	built := event.New(event.SourceStatus, "Flags", event.EdgeFalling, nil, observed)

	if built.FieldCount() != 0 {
		t.Fatalf("field count: got %d, want 0", built.FieldCount())
	}
}
