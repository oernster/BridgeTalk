package main

// Casting a machine voice from the facade, over fakes of what its lines are made through: FR-511,
// FR-512, FR-519, FR-521, FR-523, FR-527 and FR-539 to FR-542.

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/take"
	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
	"github.com/oernster/bridge-talk/internal/infrastructure/library"
	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
)

// errMissing is the reason a test's files refuse bf_emma with.
var errMissing = errors.New("bf_emma.bin is missing")

// FR-511 and FR-528: casting a machine voice speaks with it, starts making its lines and tells the
// page the voice by its id and by the name the screen shows.
func TestCastingAMachineVoiceSpeaksWithItAndTellsThePage(t *testing.T) {
	app, _, log := fixtureApp(t)

	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}

	if progress := app.session.making.Progress(); !app.session.hasVoice() || progress.Voice != "bf_emma" {
		t.Fatalf("progress = %+v with a voice cast %v; want bf_emma cast", progress, app.session.hasVoice())
	}
	state, ok := log.lastEmitted(t, stateEvent).(StateDTO)
	if !ok || state.Voice != "bf_emma" || state.VoiceDisplay != "Emma (British, female)" {
		t.Errorf("announced %+v, want bf_emma shown as Emma (British, female)", state)
	}
}

// FR-521: a cast machine voice confirms in its current made line of the confirmation.
func TestCastingAMachineVoiceIsConfirmedInAMadeLine(t *testing.T) {
	app, player, _ := fixtureApp(t)
	store := makingtest.NewStore()
	store.Hold("bf_emma", makingtest.Key("du"))
	fixtureMaking(t, app.session, offeredFiles(nil), store).BlockOn = 1

	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}

	want := []take.Take{take.Of(makingtest.PathOf("bf_emma", makingtest.Key("du")))}
	if !slices.EqualFunc(player.played, want, slices.Equal) {
		t.Errorf("played %v, want the one made confirmation %v", player.played, want)
	}
}

// FR-521: with no confirmation made yet the cast succeeds playing nothing.
func TestAMachineVoiceWithNothingMadeYetIsCastInSilence(t *testing.T) {
	app, player, _ := fixtureApp(t)
	fixtureMaking(t, app.session, offeredFiles(nil), makingtest.NewStore()).BlockOn = 1

	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}
	if len(player.played) != 0 {
		t.Errorf("played %v with nothing made", player.played)
	}
}

// FR-519: a machine voice that is not offered or whose files cannot be read is refused, leaving the
// voice already cast speaking and kept.
func TestAMachineVoiceThatCannotBeCastChangesNothing(t *testing.T) {
	app, _, _ := fixtureApp(t)
	settings := &fakeSettings{}
	app.settings = settings
	fixtureMaking(t, app.session, offeredFiles(map[string]error{"bf_emma": errMissing}), makingtest.NewStore())
	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting Alpha: %v", err)
	}

	for id, want := range map[string]error{"bf_emma": errMissing, "xx_nobody": machinevoice.ErrUnknownVoice} {
		if err := app.CastMachineVoice(id); !errors.Is(err, want) {
			t.Errorf("casting %q = %v, want %v", id, err, want)
		}
	}
	if app.session.active.Name != "Alpha" || settings.held.Voice != "Alpha" || settings.held.MachineVoice != "" {
		t.Errorf("active %q, kept %+v; want Alpha still cast and kept", app.session.active.Name, settings.held)
	}
}

// FR-540: the cast machine voice is kept apart from a recorded one, casting either forgetting the
// other.
func TestTheCastMachineVoiceIsKeptApartFromARecordedOne(t *testing.T) {
	app, _, _ := fixtureApp(t)
	settings := &fakeSettings{}
	app.settings = settings

	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}
	if settings.held.MachineVoice != "bf_emma" || settings.held.Voice != "" {
		t.Errorf("kept %+v, want bf_emma alone", settings.held)
	}
	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting Alpha: %v", err)
	}
	if settings.held.Voice != "Alpha" || settings.held.MachineVoice != "" {
		t.Errorf("kept %+v, want Alpha alone", settings.held)
	}
}

// FR-527: casting a recorded voice keeps every made line and ends the machine voice's cast.
func TestCastingARecordedVoiceKeepsEveryMadeLine(t *testing.T) {
	app, _, _ := fixtureApp(t)
	store := makingtest.NewStore()
	fixtureMaking(t, app.session, offeredFiles(nil), store)
	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}
	untilMade(t, app)

	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting Alpha: %v", err)
	}

	if held := store.Held("bf_emma"); len(held) != 3 || app.session.making.Progress().Voice != "" {
		t.Errorf("left %v with progress %+v; want the confirmation's 3 lines kept and no machine voice", held, app.session.making.Progress())
	}
}

// FR-512: a run with a machine voice kept opens casting it.
func TestAKeptMachineVoiceIsCastAtStart(t *testing.T) {
	current, _ := fixtureSession(t, newFakePlayer())
	var warnings strings.Builder

	current.castAtStart(keptCast{machine: "bf_emma"}, current.available[0], &warnings)

	if current.active.Name != "bf_emma" || current.making.Progress().Voice != "bf_emma" || warnings.Len() != 0 {
		t.Errorf("cast %q warning %q; want bf_emma cast in silence", current.active.Name, warnings.String())
	}
}

// FR-541: a kept machine voice that cannot be cast at start is warned about, then the recorded voice
// is cast; with none found, nothing is.
func TestAKeptMachineVoiceThatCannotBeCastFallsBackToARecordedVoice(t *testing.T) {
	for _, each := range []struct {
		name, id, said, cast string
		refused, recorded    bool
	}{
		{"files that cannot be read", "bf_emma", errMissing.Error(), "Alpha", true, true},
		{"a voice not offered", "xx_nobody", "xx_nobody", "Alpha", false, true},
		{"no recorded voice to fall back to", "bf_emma", errMissing.Error(), "", true, false},
	} {
		t.Run(each.name, func(t *testing.T) {
			current, _ := fixtureSession(t, newFakePlayer())
			if each.refused {
				fixtureMaking(t, current, offeredFiles(map[string]error{"bf_emma": errMissing}), makingtest.NewStore())
			}
			var recorded library.Voice
			if each.recorded {
				recorded = current.available[0]
			} else {
				current.available = nil
			}
			var warnings strings.Builder

			current.castAtStart(keptCast{machine: each.id}, recorded, &warnings)

			if current.active.Name != each.cast || current.hasVoice() != each.recorded {
				t.Errorf("cast %q, a voice cast %v; want %q", current.active.Name, current.hasVoice(), each.cast)
			}
			if !strings.Contains(warnings.String(), each.said) {
				t.Errorf("warned %q, want it to say %q", warnings.String(), each.said)
			}
		})
	}
}

// FR-540, FR-569 and FR-701: a voice given by -voice outranks every kept voice for the run,
// whichever kind was kept. The flag names a recorded voice, so honouring it and then casting
// something else would be neither.
func TestAVoiceGivenForTheRunOutranksAKeptMachineVoice(t *testing.T) {
	t.Parallel()

	held := ports.Settings{MachineVoice: "bf_emma", Plugin: "Crew", PluginVoice: "one"}

	if got := keptFrom("Alpha", held); got != (keptCast{}) {
		t.Errorf("with -voice given the kept voice is %+v, want none", got)
	}
	if got := keptFrom("", held); got.machine != "bf_emma" || got.plugin != "Crew" || got.pluginVoice != "one" {
		t.Errorf("with no -voice the kept voice is %+v, want what was stored", got)
	}
}

// FR-542: looking again keeps a cast machine voice, since it is not among the recordings.
func TestLookingAgainKeepsACastMachineVoice(t *testing.T) {
	app, _, _ := fixtureApp(t)
	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}
	audiotest.WriteTake(t, filepath.Join(app.libraryRoot, "Carol", "Docked", "take.wav"))

	if found, err := app.Rescan(); err != nil || found != len(app.session.available) {
		t.Fatalf("looking again: %d, %v", found, err)
	}
	if app.session.active.Name != "bf_emma" || app.session.making.Progress().Voice != "bf_emma" {
		t.Errorf("cast %q, progress %+v; want bf_emma still cast", app.session.active.Name, app.session.making.Progress())
	}
}

// Closing stops making, keeping what was written, then releases the model.
func TestShuttingDownStopsMakingThenReleasesTheModel(t *testing.T) {
	app, _, _ := fixtureApp(t)
	maker := fixtureMaking(t, app.session, offeredFiles(nil), makingtest.NewStore())
	maker.BlockOn = 1
	if err := app.CastMachineVoice("bf_emma"); err != nil {
		t.Fatalf("casting bf_emma: %v", err)
	}
	makingtest.Await(t, maker.Started, "making starting")

	app.shutdown(context.Background())

	if app.session.making.Progress().Making || maker.Closed() != 1 {
		t.Errorf("making %v, model released %d times; want stopped and released once",
			app.session.making.Progress().Making, maker.Closed())
	}
}

// FR-523 and FR-539: where the made lines' folder or the application cannot be found, every machine
// voice is refused with why, writing and deleting nothing.
func TestWithNowhereToKeepMadeLinesNoMachineVoiceIsCast(t *testing.T) {
	unavailable := errors.New("LOCALAPPDATA is not set")
	if _, err := filesUnless(offeredFiles(nil), nil).Open(machinevoice.All()[0]); err != nil {
		t.Fatalf("with nothing unavailable the files refused: %v", err)
	}
	app, _, _ := fixtureApp(t)
	store := makingtest.NewStore()
	fixtureMaking(t, app.session, filesUnless(offeredFiles(nil), unavailable), store)

	if err := app.CastMachineVoice("bf_emma"); !errors.Is(err, unavailable) {
		t.Errorf("casting = %v, want refused saying %v", err, unavailable)
	}
	if entries := store.Entries(); len(entries) != 0 || app.session.hasVoice() {
		t.Errorf("store saw %v, a voice cast %v; want nothing touched", entries, app.session.hasVoice())
	}
}

// FR-509: the tray offers the recorded voices found, then every machine voice by the name it is
// shown by, each carrying the kind that says how to cast it.
func TestTheTrayOffersTheMachineVoicesAfterTheRecordedVoices(t *testing.T) {
	current, _ := fixtureSession(t, newFakePlayer())

	choices := trayChoices(current.available, nil)

	want := playable(current.available)
	for _, voice := range machinevoice.All() {
		want = append(want, taskbar.Choice{
			Voice: taskbar.Voice{Kind: taskbar.Machine, Name: voice.ID()},
			Label: voice.Name(),
		})
	}
	if !slices.Equal(choices, want) {
		t.Errorf("tray offers %+v, want %+v", choices, want)
	}
}
