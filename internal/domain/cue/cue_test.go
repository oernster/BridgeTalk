package cue_test

import (
	"errors"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
)

// at is a fixed instant, so nothing in these tests reads a clock.
var at = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

func mustCue(t *testing.T, definition cue.Definition) cue.Cue {
	t.Helper()
	built, err := cue.New(definition)
	if err != nil {
		t.Fatalf("building cue %q: %v", definition.ID, err)
	}
	return built
}

func TestNewRejectsBadDefinitions(t *testing.T) {
	cases := map[string]cue.Definition{
		"no id":           {Source: "journal", Event: "FSDJump"},
		"unknown source":  {ID: "a", Source: "telepathy", Event: "FSDJump"},
		"no event":        {ID: "a", Source: "journal"},
		"no flag":         {ID: "a", Source: "status"},
		"unknown edge":    {ID: "a", Source: "status", Flag: "X", Edge: "sideways"},
		"bad priority":    {ID: "a", Source: "journal", Event: "X", Priority: "urgent"},
		"negative cooldn": {ID: "a", Source: "journal", Event: "X", Cooldown: -time.Second},
	}
	for name, definition := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := cue.New(definition); err == nil {
				t.Fatalf("expected %s to be rejected", name)
			}
		})
	}
}

func TestMatchesRequiresSourceAndName(t *testing.T) {
	built := mustCue(t, cue.Definition{ID: "a", Source: "journal", Event: "FSDJump"})

	right := event.New(event.SourceJournal, "FSDJump", event.EdgeNone, nil, at)
	if !built.Matches(right) {
		t.Fatal("cue should match its own event")
	}
	wrongName := event.New(event.SourceJournal, "Docked", event.EdgeNone, nil, at)
	if built.Matches(wrongName) {
		t.Fatal("cue matched a different event name")
	}
	wrongSource := event.New(event.SourceStatus, "FSDJump", event.EdgeNone, nil, at)
	if built.Matches(wrongSource) {
		t.Fatal("cue matched a different source")
	}
}

func TestMatchesHonoursEdge(t *testing.T) {
	built := mustCue(t, cue.Definition{
		ID: "a", Source: "status", Flag: "LandingGearDown", Edge: "rising",
	})
	rising := event.New(event.SourceStatus, "LandingGearDown", event.EdgeRising, nil, at)
	falling := event.New(event.SourceStatus, "LandingGearDown", event.EdgeFalling, nil, at)
	if !built.Matches(rising) {
		t.Fatal("rising cue should match a rising edge")
	}
	if built.Matches(falling) {
		t.Fatal("rising cue matched a falling edge")
	}
}

func TestMatchesComparesPayloadAcrossTypes(t *testing.T) {
	built := mustCue(t, cue.Definition{
		ID: "a", Source: "journal", Event: "ShieldState",
		Match: map[string]string{"ShieldsUp": "false"},
	})
	// The journal writes booleans and numbers, the cue table writes text, so the
	// comparison has to bridge the two.
	down := event.New(event.SourceJournal, "ShieldState", event.EdgeNone,
		map[string]any{"ShieldsUp": false}, at)
	up := event.New(event.SourceJournal, "ShieldState", event.EdgeNone,
		map[string]any{"ShieldsUp": true}, at)
	if !built.Matches(down) {
		t.Fatal("cue should match ShieldsUp=false")
	}
	if built.Matches(up) {
		t.Fatal("cue matched ShieldsUp=true")
	}
}

func TestMatchesComparesNumbersWrittenAsJSON(t *testing.T) {
	built := mustCue(t, cue.Definition{
		ID: "a", Source: "journal", Event: "FSDJump",
		Match: map[string]string{"BoostUsed": "1"},
	})
	// JSON numbers decode as float64, so a literal 1 arrives as 1.0.
	boosted := event.New(event.SourceJournal, "FSDJump", event.EdgeNone,
		map[string]any{"BoostUsed": float64(1)}, at)
	if !built.Matches(boosted) {
		t.Fatal("cue should match a JSON number written as text")
	}
}

func TestResolvePrefersTheMoreSpecificCue(t *testing.T) {
	broad := mustCue(t, cue.Definition{ID: "broad", Source: "journal", Event: "SupercruiseExit"})
	narrow := mustCue(t, cue.Definition{
		ID: "narrow", Source: "journal", Event: "SupercruiseExit",
		Match: map[string]string{"BodyType": "Station"},
	})
	// Declared broad-first on purpose: specificity must decide, not declaration order.
	table := cue.NewTable([]cue.Cue{broad, narrow})

	encounter := event.New(event.SourceJournal, "SupercruiseExit", event.EdgeNone,
		map[string]any{"BodyType": "Station"}, at)
	resolved, ok := table.Resolve(encounter)
	if !ok {
		t.Fatal("expected a match")
	}
	if resolved.ID() != "narrow" {
		t.Fatalf("resolved %q, want the more specific cue", resolved.ID())
	}

	plain := event.New(event.SourceJournal, "SupercruiseExit", event.EdgeNone, nil, at)
	resolved, ok = table.Resolve(plain)
	if !ok || resolved.ID() != "broad" {
		t.Fatalf("plain event resolved to %q, want broad", resolved.ID())
	}
}

func TestResolveReportsNoMatch(t *testing.T) {
	table := cue.NewTable([]cue.Cue{
		mustCue(t, cue.Definition{ID: "a", Source: "journal", Event: "FSDJump"}),
	})
	unknown := event.New(event.SourceJournal, "Music", event.EdgeNone, nil, at)
	if _, ok := table.Resolve(unknown); ok {
		t.Fatal("an unlistened event should not resolve")
	}
}

func TestEventFieldsAreCopiedOnConstruction(t *testing.T) {
	fields := map[string]any{"StarSystem": "Sol"}
	built := event.New(event.SourceJournal, "FSDJump", event.EdgeNone, fields, at)
	fields["StarSystem"] = "Deciat"

	value, ok := built.Field("StarSystem")
	if !ok || value != "Sol" {
		t.Fatalf("field = %v, want Sol: the event kept a reference to the caller's map", value)
	}
}

func TestIDGroupReadsTheFirstSegment(t *testing.T) {
	t.Parallel()
	cases := map[cue.ID]string{
		"ShieldState.ShieldsUp.false":  "ShieldState",
		"LightsOn.Set":                 "LightsOn",
		"DockingDenied.Reason.NoSpace": "DockingDenied",
		"StartJump":                    "StartJump",
		"":                             "",
		".leading":                     "",
	}
	for id, want := range cases {
		if got := id.Group(); got != want {
			t.Errorf("ID(%q).Group() = %q, want %q", id, got, want)
		}
	}
}

// The cue table is text, so a match is compared against the rendered form of
// whatever the payload holds. Anything that is not a string, a bool or a number
// takes the default branch.
func TestMatchComparesAValueThatIsNeitherTextNorNumber(t *testing.T) {
	t.Parallel()
	item := mustCue(t, cue.Definition{
		ID: "WingJoin", Source: "journal", Event: "WingJoin",
		Match: map[string]string{"Others": "[a b]"},
	})

	matching := event.New(event.SourceJournal, "WingJoin", event.EdgeNone,
		map[string]any{"Others": []string{"a", "b"}}, at)
	if !item.Matches(matching) {
		t.Error("a rendered slice that equals the expectation did not match")
	}

	other := event.New(event.SourceJournal, "WingJoin", event.EdgeNone,
		map[string]any{"Others": []string{"c"}}, at)
	if item.Matches(other) {
		t.Error("a rendered slice that differs from the expectation matched")
	}
}

func TestResolveFindsNothingWhenTheEventIsKnownButNoCueMatches(t *testing.T) {
	t.Parallel()
	table := cue.NewTable([]cue.Cue{
		mustCue(t, cue.Definition{
			ID: "DockingDenied.Reason.NoSpace", Source: "journal", Event: "DockingDenied",
			Match: map[string]string{"Reason": "NoSpace"},
		}),
	})

	candidate := event.New(event.SourceJournal, "DockingDenied", event.EdgeNone,
		map[string]any{"Reason": "TooLarge"}, at)

	if _, ok := table.Resolve(candidate); ok {
		t.Error("an event the table listens for but no cue matches resolved to a cue")
	}
}

func TestNamesListsTheDistinctThingsOneSourceListensFor(t *testing.T) {
	t.Parallel()
	table := cue.NewTable([]cue.Cue{
		mustCue(t, cue.Definition{ID: "DockingDenied.Reason.NoSpace", Source: "journal", Event: "DockingDenied",
			Match: map[string]string{"Reason": "NoSpace"}}),
		mustCue(t, cue.Definition{ID: "DockingDenied.Reason.TooLarge", Source: "journal", Event: "DockingDenied",
			Match: map[string]string{"Reason": "TooLarge"}}),
		mustCue(t, cue.Definition{ID: "Bounty", Source: "journal", Event: "Bounty"}),
		mustCue(t, cue.Definition{ID: "Docked.Set", Source: "status", Flag: "Docked",
			Edge: "rising"}),
	})

	journal := table.Names(event.SourceJournal)
	if len(journal) != 2 || journal[0] != "Bounty" || journal[1] != "DockingDenied" {
		t.Errorf("journal names = %v, want [Bounty DockingDenied] with the repeat collapsed", journal)
	}

	status := table.Names(event.SourceStatus)
	if len(status) != 1 || status[0] != "Docked" {
		t.Errorf("status names = %v, want [Docked]", status)
	}
}

func TestNamesIsEmptyForASourceTheTableIgnores(t *testing.T) {
	t.Parallel()
	table := cue.NewTable([]cue.Cue{
		mustCue(t, cue.Definition{ID: "Bounty", Source: "journal", Event: "Bounty"}),
	})

	if got := table.Names(event.SourceStatus); len(got) != 0 {
		t.Errorf("names for an unused source = %v, want none", got)
	}
}

func TestParsePriorityReadsEverySpellingAndRejectsTheRest(t *testing.T) {
	t.Parallel()
	cases := map[string]cue.Priority{
		"flavour": cue.PriorityFlavour,
		"ambient": cue.PriorityAmbient,
		"notice":  cue.PriorityNotice,
		"alert":   cue.PriorityAlert,
	}
	for text, want := range cases {
		got, err := cue.ParsePriority(text)
		if err != nil {
			t.Errorf("ParsePriority(%q): %v", text, err)
			continue
		}
		if got != want {
			t.Errorf("ParsePriority(%q) = %v, want %v", text, got, want)
		}
	}

	if _, err := cue.ParsePriority("urgent"); !errors.Is(err, cue.ErrInvalidCue) {
		t.Errorf("an unknown spelling gave %v, want ErrInvalidCue", err)
	}
}

func TestACueCarriesEveryFieldItWasDefinedWith(t *testing.T) {
	t.Parallel()
	item := mustCue(t, cue.Definition{
		ID: "StartJump", Source: "journal", Event: "StartJump",
		Priority: "alert", Cooldown: 45 * time.Second,
	})

	if item.ID() != "StartJump" {
		t.Errorf("id = %q", item.ID())
	}
	if item.Source() != event.SourceJournal {
		t.Errorf("source = %q", item.Source())
	}
	if item.Name() != "StartJump" {
		t.Errorf("name = %q", item.Name())
	}
	if item.Priority() != cue.PriorityAlert {
		t.Errorf("priority = %d", item.Priority())
	}
	if item.Cooldown() != 45*time.Second {
		t.Errorf("cooldown = %v", item.Cooldown())
	}
}

func TestATableReportsEveryCueItHolds(t *testing.T) {
	t.Parallel()
	first := mustCue(t, cue.Definition{ID: "Bounty", Source: "journal", Event: "Bounty"})
	second := mustCue(t, cue.Definition{ID: "Docked.Set", Source: "status", Flag: "Docked"})
	table := cue.NewTable([]cue.Cue{first, second})

	if table.Len() != 2 {
		t.Errorf("Len() = %d, want 2", table.Len())
	}
	all := table.All()
	if len(all) != 2 || all[0].ID() != first.ID() || all[1].ID() != second.ID() {
		t.Fatalf("All() = %v, want the definition order preserved", all)
	}

	all[0] = second
	if table.All()[0].ID() != first.ID() {
		t.Error("All() handed out the table's own slice, so a caller could rewrite it")
	}
}

// TestAnApplicationCueIsAcceptedAndNeverResolvedByTheGame covers the source added for
// the acknowledgement played when a voice is chosen.
//
// Statement coverage does not reach this on its own: the source was added to an
// existing case list, so no new statement exists to be counted and the gate stayed
// green while the behaviour was untested. These assertions are the actual proof.
func TestAnApplicationCueIsAcceptedAndNeverResolvedByTheGame(t *testing.T) {
	built := mustCue(t, cue.Definition{
		ID: "Cast.Confirmed", Source: "application", Event: "cast",
	})
	if built.Source() != event.SourceApplication {
		t.Fatalf("source = %q, want application", built.Source())
	}

	table := cue.NewTable([]cue.Cue{built})
	if got, ok := table.Resolve(
		event.New(event.SourceJournal, "cast", event.EdgeNone, nil, at),
	); ok {
		t.Fatalf("a journal event resolved the application cue as %q", got.ID())
	}
	if got, ok := table.Resolve(
		event.New(event.SourceStatus, "cast", event.EdgeNone, nil, at),
	); ok {
		t.Fatalf("a status change resolved the application cue as %q", got.ID())
	}

	// It is still part of the vocabulary, so it is counted and can be recorded.
	if len(table.All()) != 1 {
		t.Fatalf("table holds %d cues, want the application cue among them", len(table.All()))
	}
}

// TestAnApplicationCueStillNeedsAName holds the rule that a source alone is not a
// definition: the moment has to be named, as an event or a flag is.
func TestAnApplicationCueStillNeedsAName(t *testing.T) {
	if _, err := cue.New(cue.Definition{ID: "a", Source: "application"}); err == nil {
		t.Fatal("an application cue with no moment named was accepted")
	}
}
