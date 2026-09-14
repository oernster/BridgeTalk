package main

// FR-514 and FR-521 through the facade: a machine voice's confirmation is played once it is written,
// and a cue that fires before its line is made waits for that line.

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/domain/event"
)

// makingDeadline is how long a test waits for the fakes to finish making, far beyond the moment they
// take.
const makingDeadline = 2 * time.Second

// untilMade waits for making to end, failing the test past makingDeadline.
func untilMade(t *testing.T, app *App) {
	t.Helper()
	deadline := time.Now().Add(makingDeadline)
	for app.session.making.Progress().Making {
		if time.Now().After(deadline) {
			t.Fatal("making never ended")
		}
		time.Sleep(time.Millisecond)
	}
}

// playedSoFar returns what the device has been given, under its lock.
func playedSoFar(player *fakePlayer) [][]string {
	player.mu.Lock()
	defer player.mu.Unlock()
	return append([][]string(nil), player.played...)
}

// FR-521: with nothing made at the cast, the confirmation plays on the tick after it is written, once.
func TestAConfirmationWrittenAfterTheCastIsPlayedOnTheNextTick(t *testing.T) {
	app, player, _ := fixtureApp(t)
	maker := fixtureMaking(t, app.session, offeredFiles(nil), makingtest.NewStore())
	maker.PauseOn = 1

	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}
	<-maker.Started
	app.tickMaking()
	if got := playedSoFar(player); len(got) != 0 {
		t.Fatalf("played %v before the confirmation was written", got)
	}

	close(maker.Resume)
	untilMade(t, app)
	app.tickMaking()
	app.tickMaking()

	// Every line is made by then, so the confirmation is any one of its three, chosen at random.
	var lines []string
	for _, sounds := range []string{"du", "dɪ", "di"} {
		lines = append(lines, makingtest.PathOf("bf_emma", makingtest.Key(sounds)))
	}
	if got := playedSoFar(player); len(got) != 1 || len(got[0]) != 1 || !slices.Contains(lines, got[0][0]) {
		t.Errorf("played %v, want one of the confirmation's lines %v once", got, lines)
	}
}

// FR-514: a machine cast hands its lines to the reaction service, so a cue fired before its line is
// made is recorded as making rather than unbound; a poll tick after the line is written speaks it.
func TestACueFiredBeforeItsLineIsMadeWaitsThroughTheFacade(t *testing.T) {
	app, _, _ := fixtureApp(t)
	maker := fixtureMaking(t, app.session, offeredFiles(nil), makingtest.NewStore())
	maker.PauseOn = 1
	source := &fakeSource{name: "journal"}
	app.sources = []ports.EventSource{source}
	app.session.reporter = reporter{app: app}

	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}
	<-maker.Started
	source.events = []event.Event{event.New(
		event.SourceJournal, "Docked", event.EdgeNone, map[string]any{}, time.Date(2026, 9, 14, 11, 0, 0, 0, time.UTC),
	)}
	app.pollAndAnnounce()
	close(maker.Resume)
	untilMade(t, app)
	app.pollAndAnnounce()

	var outcomes []string
	for _, line := range app.Reactions() {
		if line.Cue == "Docked" {
			outcomes = append(outcomes, line.Outcome)
		}
	}
	if len(outcomes) == 0 || outcomes[0] != ports.OutcomeMaking || !slices.Contains(outcomes, ports.OutcomePlayed) {
		t.Errorf("Docked was recorded %v, want making first, then played once its line was written", outcomes)
	}
}

// FR-521: a voice cast before the confirmation is written leaves it unplayed.
func TestAConfirmationIsForgottenWhenAnotherVoiceIsCastFirst(t *testing.T) {
	app, player, _ := fixtureApp(t)
	maker := fixtureMaking(t, app.session, offeredFiles(nil), makingtest.NewStore())
	maker.PauseOn = 1

	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}
	<-maker.Started
	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting Alpha: %v", err)
	}
	app.tickMaking()

	got := playedSoFar(player)
	if len(got) != 1 {
		t.Fatalf("played %v, want Alpha's own confirmation alone", got)
	}
	if strings.Contains(got[0][0], "bf_emma") {
		t.Errorf("played %v, a confirmation of the voice cast before", got)
	}
}
