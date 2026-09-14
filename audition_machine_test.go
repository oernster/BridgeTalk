package main

// FR-545 to FR-548 through the facade: a machine voice auditioned on the script's groups, its line made
// on the press, the buttons held while it is made, Stop leaving it unplayed and a failure said.

import (
	"errors"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
)

// auditionInBackground presses a machine voice's group button on a goroutine of its own, answering
// when the press has returned with what it answered.
func auditionInBackground(app *App, id, group string) (<-chan struct{}, *AuditionDTO, *error) {
	done := make(chan struct{})
	var played AuditionDTO
	var failure error
	go func() {
		defer close(done)
		played, failure = app.AuditionMachineVoice(id, group)
	}()
	return done, &played, &failure
}

// FR-546: every machine voice is auditioned on the script's groups, each counting its lines.
func TestAMachineVoiceIsAuditionedOnTheScriptsGroups(t *testing.T) {
	app, _, _ := fixtureApp(t)

	groups := app.MachineAuditionGroups()

	want := []GroupDTO{{Key: "Cast", Label: label("Cast"), Clips: 3}, {Key: "Docked", Label: label("Docked"), Clips: 3}}
	if len(groups) != len(want) || groups[0] != want[0] || groups[1] != want[1] {
		t.Errorf("groups = %v, want %v", groups, want)
	}
}

// FR-546: a machine voice not cast is auditioned while muted. The line drawn is made, kept and played
// once; the voice stays uncast.
func TestAMachineVoiceAuditionMakesTheLineDrawnKeepsItAndPlaysIt(t *testing.T) {
	app, player, _ := fixtureApp(t)
	store := makingtest.NewStore()
	maker := fixtureMaking(t, app.session, offeredFiles(nil), store)
	app.SetMuted(true)

	played, err := app.AuditionMachineVoice("bf_emma", "Docked")

	if err != nil {
		t.Fatalf("auditioning bf_emma: %v", err)
	}
	held := store.Held("bf_emma")
	if len(held) != 1 || maker.Made() != 1 {
		t.Fatalf("made %d keeping %v, want the line drawn alone", maker.Made(), held)
	}
	path := makingtest.PathOf("bf_emma", held[0])
	if got := playedSoFar(player); len(got) != 1 || len(got[0]) != 1 || got[0][0] != path {
		t.Errorf("played %v, want %q once", got, path)
	}
	if played.Group != "Docked" || played.Clip != clipName(path) {
		t.Errorf("answered %+v, want the group and the made line's file", played)
	}
	if app.State().Voice == "bf_emma" {
		t.Error("auditioning bf_emma cast her")
	}
}

// FR-546: a line already made plays without being made again.
func TestAMachineVoiceAuditionOfALineAlreadyMadeMakesNothing(t *testing.T) {
	app, player, _ := fixtureApp(t)
	store := makingtest.NewStore()
	store.Hold("bf_emma", makingtest.Key("bə"), makingtest.Key("bɪ"), makingtest.Key("bi"))
	maker := fixtureMaking(t, app.session, offeredFiles(nil), store)

	if _, err := app.AuditionMachineVoice("bf_emma", "Docked"); err != nil {
		t.Fatalf("auditioning bf_emma: %v", err)
	}

	if maker.Made() != 0 || len(playedSoFar(player)) != 1 {
		t.Errorf("made %d and played %v, want a made line played with nothing made", maker.Made(), playedSoFar(player))
	}
}

// FR-547: while an audition's line is being made, the page is told something is under way, asking
// says so and a further press is ignored.
func TestAPressWhileAnAuditionsLineIsBeingMadeIsIgnored(t *testing.T) {
	app, player, log := fixtureApp(t)
	maker := fixtureMaking(t, app.session, offeredFiles(nil), makingtest.NewStore())
	maker.PauseOn = 1

	done, _, failure := auditionInBackground(app, "bf_emma", "Docked")
	makingtest.Await(t, maker.Started, "the audition's line starting")

	if !app.Playing() || !announcedPlaying(log) {
		t.Errorf("while the line was made, Playing = %v and announced = %v; want both true", app.Playing(), announcedPlaying(log))
	}
	ignored, err := app.AuditionMachineVoice("bf_emma", "Docked")
	if err != nil || ignored.Clip != "" {
		t.Errorf("a press while the line was made answered %+v, %v; want it ignored", ignored, err)
	}
	close(maker.Resume)
	makingtest.Await(t, done, "the audition finishing")

	if *failure != nil || maker.Made() != 1 || len(playedSoFar(player)) != 1 {
		t.Errorf("the first press answered %v after making %d and playing %v; want one line made and played",
			*failure, maker.Made(), playedSoFar(player))
	}
}

// FR-547: Stop while an audition's line is being made frees the buttons at once; the line is kept once
// written and nothing plays.
func TestStopWhileAnAuditionsLineIsBeingMadePlaysNothingAndKeepsIt(t *testing.T) {
	app, player, log := fixtureApp(t)
	store := makingtest.NewStore()
	maker := fixtureMaking(t, app.session, offeredFiles(nil), store)
	maker.PauseOn = 1

	done, played, failure := auditionInBackground(app, "bf_emma", "Docked")
	makingtest.Await(t, maker.Started, "the audition's line starting")
	app.StopAudition()

	if app.Playing() {
		t.Error("Stop left the audition reported as under way")
	}
	if state, ok := log.lastEmitted(t, playbackEvent).(PlaybackDTO); !ok || state.Playing {
		t.Errorf("after Stop the page was last told %+v, want nothing playing", state)
	}
	close(maker.Resume)
	makingtest.Await(t, done, "the audition finishing")

	if *failure != nil || played.Clip != "" || len(playedSoFar(player)) != 0 {
		t.Errorf("the stopped press answered %+v, %v and played %v; want nothing played", *played, *failure, playedSoFar(player))
	}
	if held := store.Held("bf_emma"); len(held) != 1 {
		t.Errorf("kept %v, want the line made before Stop", held)
	}
}

// FR-548: an audition whose line cannot be made says why, plays nothing and frees the buttons.
func TestAMachineVoiceAuditionThatCannotBeMadeSaysWhy(t *testing.T) {
	app, player, log := fixtureApp(t)
	missing := errors.New("bf_emma.bin is missing")
	fixtureMaking(t, app.session, offeredFiles(map[string]error{"bf_emma": missing}), makingtest.NewStore())

	_, err := app.AuditionMachineVoice("bf_emma", "Docked")

	if !errors.Is(err, missing) {
		t.Errorf("auditioning = %v, want the files' own reason", err)
	}
	if len(playedSoFar(player)) != 0 || app.Playing() {
		t.Errorf("played %v with Playing = %v, want nothing under way", playedSoFar(player), app.Playing())
	}
	if state, ok := log.lastEmitted(t, playbackEvent).(PlaybackDTO); !ok || state.Playing {
		t.Errorf("the page was last told %+v, want nothing playing", state)
	}
}

// FR-546: a group the script holds no lines for is refused in the words the pane shows.
func TestAMachineVoiceAuditionOfAGroupTheScriptLacksIsRefusedByName(t *testing.T) {
	app, _, _ := fixtureApp(t)

	_, err := app.AuditionMachineVoice("bf_emma", "ShieldState")

	if err == nil || !strings.Contains(err.Error(), "Shield state") {
		t.Errorf("auditioning = %v, want a refusal naming Shield state", err)
	}
}

// A voice that is not offered is refused; so is an audition with no device, before anything is made.
func TestAMachineVoiceAuditionIsRefusedForAVoiceNotOfferedOrNoDevice(t *testing.T) {
	app, _, _ := fixtureApp(t)
	maker := fixtureMaking(t, app.session, offeredFiles(nil), makingtest.NewStore())

	if _, err := app.AuditionMachineVoice("xx_nobody", "Docked"); err == nil {
		t.Error("a voice that is not offered auditioned")
	}
	app.session.player = nil
	_, err := app.AuditionMachineVoice("bf_emma", "Docked")
	if err == nil || !strings.Contains(err.Error(), "audio device") || maker.Made() != 0 {
		t.Errorf("with no device auditioning = %v after making %d, want the missing device named with nothing made", err, maker.Made())
	}
}
