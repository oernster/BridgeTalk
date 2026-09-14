package main

// A reaction reaches the page under its moment's full title as well as its id (FR-719, FR-233),
// whichever kind of voice is cast.

import (
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
)

// shieldsTitle is the full title FR-233 gives the fixture's shield cue: the whole id read as
// words, with no group heading standing apart from it.
const shieldsTitle = "Shield state: shields up false"

// reportShields reports one played reaction for the fixture's shield cue, then answers the
// line the history holds for it and the line announced to the page.
func reportShields(t *testing.T, app *App, log *recorder) (ReactionDTO, ReactionDTO) {
	t.Helper()
	reporter{app: app}.Report(ports.Reaction{
		At:      time.Date(2026, 9, 14, 9, 30, 0, 0, time.UTC),
		Cue:     "ShieldState.ShieldsUp.false",
		Clip:    "a.mp3",
		Outcome: ports.OutcomePlayed,
	})
	history := app.Reactions()
	if len(history) != 1 {
		t.Fatalf("the history holds %d lines, want one", len(history))
	}
	announced, ok := log.lastEmitted(t, reactionEvent).(ReactionDTO)
	if !ok {
		t.Fatal("the reaction was announced as something other than a ReactionDTO")
	}
	return history[0], announced
}

// checkTitled fails where either line lacks the shield cue's full title or its id.
func checkTitled(t *testing.T, recorded, announced ReactionDTO) {
	t.Helper()
	for _, line := range []ReactionDTO{recorded, announced} {
		if line.Title != shieldsTitle || line.Cue != "ShieldState.ShieldsUp.false" {
			t.Errorf("line = %+v, want the id with the title %q", line, shieldsTitle)
		}
	}
}

func TestAReactionCarriesItsMomentsFullTitleForARecordedVoice(t *testing.T) {
	app, _, log := fixtureApp(t)
	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting Alpha: %v", err)
	}

	recorded, announced := reportShields(t, app, log)

	checkTitled(t, recorded, announced)
}

func TestAReactionCarriesItsMomentsFullTitleForAMachineVoice(t *testing.T) {
	app, _, log := fixtureApp(t)
	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}

	recorded, announced := reportShields(t, app, log)

	checkTitled(t, recorded, announced)
}
