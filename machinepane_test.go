package main

// What the Cast pane asks about machine voices, answered from the facade: FR-508, FR-515, FR-518,
// FR-520, FR-522, FR-528 and FR-530.

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
)

// madeOut waits for making to end, then answers what the pane would read.
func madeOut(t *testing.T, app *App) MakingDTO {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		if got := app.Making(); !got.Making {
			return got
		}
		select {
		case <-deadline:
			t.Fatal("making never ended")
		case <-time.After(time.Millisecond):
		}
	}
}

// FR-508 and FR-528: every machine voice offered, in the order offered, by its id and the name the
// screen shows.
func TestTheCastPaneOffersEveryMachineVoiceByItsName(t *testing.T) {
	app, _, _ := fixtureApp(t)

	rows := app.MachineVoices()

	offered := machinevoice.All()
	if len(rows) != len(offered) {
		t.Fatalf("got %d machine voices, want the %d offered", len(rows), len(offered))
	}
	for index, voice := range offered {
		if want := (MachineVoiceDTO{ID: voice.ID(), Name: voice.Name()}); rows[index] != want {
			t.Errorf("row %d = %+v, want %+v", index, rows[index], want)
		}
	}
	if rows[1] != (MachineVoiceDTO{ID: "bf_emma", Name: "Emma (British, female)"}) {
		t.Errorf("row 1 = %+v, want bf_emma shown as Emma (British, female)", rows[1])
	}
}

// FR-515 and FR-522: once making ends, the pane reads every line made and the moments spoken for.
func TestMakingReportsHowFarItHasGot(t *testing.T) {
	app, _, _ := fixtureApp(t)
	if idle := app.Making(); idle.Voice != "" || idle.Failed == nil {
		t.Errorf("before a cast Making = %+v, want no voice and an empty list of failures", idle)
	}
	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}

	got := madeOut(t, app)

	want := MakingDTO{Voice: "bf_emma", Current: 6, Total: 6, CuesServed: 2, Failed: []LineFailureDTO{}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Making = %+v, want %+v", got, want)
	}
}

// FR-518, FR-520 and FR-530: a line that could not be made names its moment by title, its place and
// why; a write that stopped making and a delete that failed each say why.
func TestMakingReportsWhatWentWrong(t *testing.T) {
	app, _, _ := fixtureApp(t)
	store := makingtest.NewStore()
	store.FailWrite = 2
	store.DeleteErr = errors.New("a made line is in use")
	fixtureMaking(t, app.session, offeredFiles(nil), store).FailOn = 1
	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}

	got := madeOut(t, app)

	if len(got.Failed) != 1 {
		t.Fatalf("failed = %+v, want the first line alone", got.Failed)
	}
	failure := got.Failed[0]
	if failure.Cue.ID != "Cast.Confirmed" || failure.Cue.Title != "Cast: confirmed" || failure.Line != 1 ||
		failure.Reason != makingtest.ErrModel.Error() {
		t.Errorf("failure = %+v, want the confirmation's first line refused by the model", failure)
	}
	if got.Stopped != makingtest.ErrDisk.Error() || got.NotDeleted != store.DeleteErr.Error() || got.Current != 1 {
		t.Errorf("Making = %+v, want stopped by the disk after one line, the delete failure kept", got)
	}
}

// FR-515: the page is told as making moves, once for each change rather than on every tick.
func TestAPollAnnouncesMakingOnlyWhenItHasMoved(t *testing.T) {
	app, _, log := fixtureApp(t)
	app.pollAndAnnounce()
	if seen := log.countEmitted(makingEvent); seen != 0 {
		t.Fatalf("an idle poll announced making %d times", seen)
	}
	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}
	madeOut(t, app)

	app.pollAndAnnounce()
	app.pollAndAnnounce()

	if seen := log.countEmitted(makingEvent); seen != 1 {
		t.Errorf("announced making %d times over two polls, want once", seen)
	}
	if got, ok := log.lastEmitted(t, makingEvent).(MakingDTO); !ok || got.Current != 6 {
		t.Errorf("announced %+v, want the six lines made", got)
	}
}

// The page tells a machine voice from a recorded voice of the same name by the state (FR-540).
func TestTheStateSaysWhetherTheCastVoiceIsAMachineVoice(t *testing.T) {
	app, _, _ := fixtureApp(t)
	if err := app.SelectVoice("Alpha"); err != nil || app.State().MachineVoice {
		t.Errorf("casting Alpha: %v, machine voice %v; want a recorded voice", err, app.State().MachineVoice)
	}
	if err := app.CastMachineVoice("bf_emma"); err != nil || !app.State().MachineVoice {
		t.Errorf("casting bf_emma: %v, machine voice %v; want a machine voice", err, app.State().MachineVoice)
	}
	if strings.TrimSpace(app.State().VoiceDisplay) == "" {
		t.Error("the cast machine voice has no name to show")
	}
}
