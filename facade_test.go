package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
)

// A decision has to reach both the history the pane reads and the channel it listens
// on, because the pane is built once and updated live thereafter.
func TestADecisionIsRecordedAndAnnounced(t *testing.T) {
	app, _, log := fixtureApp(t)

	reporter{app: app}.Report(ports.Reaction{
		At:      time.Date(2026, 8, 26, 9, 30, 0, 0, time.UTC),
		Cue:     "ShieldState.ShieldsUp.false",
		Clip:    filepath.Join("C:", "Recordings", "a.mp3"),
		Outcome: "played",
	})

	history := app.Reactions()
	if len(history) != 1 {
		t.Fatalf("the history holds %d lines, want one", len(history))
	}
	line := history[0]
	if line.At != "09:30:00" {
		t.Errorf("at = %q, want the time of day alone", line.At)
	}
	if line.Cue != "ShieldState.ShieldsUp.false" {
		t.Errorf("line = %+v, want the cue carried through", line)
	}
	if line.Clip != "a.mp3" {
		t.Errorf("clip = %q, want the file name alone", line.Clip)
	}

	announced, ok := log.lastEmitted(t, reactionEvent).(ReactionDTO)
	if !ok {
		t.Fatal("the reaction was announced as something other than a ReactionDTO")
	}
	if announced != line {
		t.Errorf("announced %+v, recorded %+v; they have to be the same line", announced, line)
	}
}

// The history is bounded so a long session does not accumulate a session's worth of
// chatter in memory. The oldest lines go, not the newest.
func TestTheHistoryKeepsOnlyTheMostRecentDecisions(t *testing.T) {
	app, _, _ := fixtureApp(t)

	for index := 0; index < reactionHistory+10; index++ {
		reporter{app: app}.Report(ports.Reaction{Cue: "ShieldState.ShieldsUp.false", Outcome: "played"})
	}

	history := app.Reactions()
	if len(history) != reactionHistory {
		t.Fatalf("the history holds %d lines, want the cap of %d", len(history), reactionHistory)
	}
}

// The history is copied on the way out. Handing the slice itself over would let the
// page's own reading change under a decision being recorded beside it.
func TestTheHistoryIsCopiedRatherThanHandedOver(t *testing.T) {
	app, _, _ := fixtureApp(t)
	reporter{app: app}.Report(ports.Reaction{Cue: "ShieldState.ShieldsUp.false", Outcome: "played"})

	taken := app.Reactions()
	taken[0].Cue = "something else entirely"

	if app.Reactions()[0].Cue != "ShieldState.ShieldsUp.false" {
		t.Fatal("changing the returned slice changed the history behind it")
	}
}

func TestAClipIsTrimmedToItsFileNameForTheLog(t *testing.T) {
	cases := map[string]string{
		`C:\Recordings\alpha\b.mp3`: "b.mp3",
		"/recordings/alpha/b.mp3":   "b.mp3",
		"b.mp3":                     "b.mp3",
		"":                          "",
	}
	for path, want := range cases {
		if got := baseName(path); got != want {
			t.Errorf("baseName(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestTheStateCarriesTheDirectoriesItIsReadingFrom(t *testing.T) {
	app, _, _ := fixtureApp(t)

	state := app.State()
	if state.JournalDir != "journal-dir" || state.StatusPath != "status-file" {
		t.Errorf("state = %+v, want the paths it was assembled over", state)
	}
	if state.Version != version {
		t.Errorf("version = %q, want the built version %q", state.Version, version)
	}
	if state.Total != 4 {
		t.Errorf("total = %d, want the whole table even with nothing cast", state.Total)
	}
}

func TestMutingSilencesTheDeviceAndUnmutingDoesNot(t *testing.T) {
	app, player, log := fixtureApp(t)

	app.SetMuted(true)
	player.mu.Lock()
	stops := player.stops
	player.mu.Unlock()
	if stops != 1 {
		t.Fatalf("muting stopped the device %d times, want once", stops)
	}
	if !app.Muted() {
		t.Fatal("the mute did not stick")
	}

	app.SetMuted(false)
	player.mu.Lock()
	stops = player.stops
	player.mu.Unlock()
	if stops != 1 {
		t.Fatalf("unmuting stopped the device again: stops = %d", stops)
	}
	if log.countEmitted(stateEvent) != 2 {
		t.Fatalf("the state was announced %d times, want once per change",
			log.countEmitted(stateEvent))
	}
}

func TestTheVolumeIsReadFromAndWrittenToTheDevice(t *testing.T) {
	app, player, _ := fixtureApp(t)

	if app.Volume() != 1 {
		t.Fatalf("volume = %v, want the device's own level", app.Volume())
	}
	app.SetVolume(0.25)
	if got := player.Volume(); got != 0.25 {
		t.Fatalf("the device is at %v, want 0.25", got)
	}
	if app.Volume() != 0.25 {
		t.Fatalf("volume reads %v, want what was just set", app.Volume())
	}
}

// With no audio device there is nothing to ask and nothing to set; both controls
// still have to answer rather than panicking on a nil device.
func TestTheVolumeControlsAnswerWithNoDevice(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.session.player = nil

	if app.Volume() != 0 {
		t.Fatalf("volume = %v, want silence with no device", app.Volume())
	}
	app.SetVolume(0.5)
}

// Every tray choice goes through the same door as the control it mirrors, so the two
// cannot drift apart. Quit in particular records its intent; without that the close
// dialog would appear over the quit it was just asked to perform.
func TestEachTrayChoiceActsThroughTheControlItMirrors(t *testing.T) {
	t.Run("quit records the intent", func(t *testing.T) {
		app, _, _ := fixtureApp(t)
		quits := 0
		app.quit = func() { quits++ }

		app.handleTray(taskbar.Command{Kind: taskbar.CommandQuit})

		if quits != 1 {
			t.Fatalf("the tray asked to quit %d times, want once", quits)
		}
		if !app.quitting.Load() {
			t.Fatal("a tray quit did not record its intent, so the dialog would ask again")
		}
	})

	t.Run("mute toggles rather than sets", func(t *testing.T) {
		app, _, _ := fixtureApp(t)

		app.handleTray(taskbar.Command{Kind: taskbar.CommandToggleMute})
		if !app.Muted() {
			t.Fatal("the tray toggle did not mute")
		}
		app.handleTray(taskbar.Command{Kind: taskbar.CommandToggleMute})
		if app.Muted() {
			t.Fatal("the tray toggle did not unmute")
		}
	})

	t.Run("selecting a voice casts it", func(t *testing.T) {
		app, _, _ := fixtureApp(t)

		app.handleTray(taskbar.Command{Kind: taskbar.CommandSelectVoice, Voice: "Alpha"})

		if app.session.active.Name != "Alpha" {
			t.Fatalf("active voice is %q, want Alpha", app.session.active.Name)
		}
	})

	// FR-509: a machine voice chosen from the tray is cast as one, by its id.
	t.Run("selecting a machine voice casts it", func(t *testing.T) {
		app, _, _ := fixtureApp(t)

		app.handleTray(taskbar.Command{Kind: taskbar.CommandSelectMachineVoice, Voice: "bf_emma"})

		if app.session.active != (castVoice{Name: "bf_emma", Display: "Emma (British, female)", Machine: true}) {
			t.Fatalf("active voice is %+v, want bf_emma cast as a machine voice", app.session.active)
		}
	})

	t.Run("a voice the tray cannot cast is not fatal", func(t *testing.T) {
		app, _, _ := fixtureApp(t)

		app.handleTray(taskbar.Command{Kind: taskbar.CommandSelectVoice, Voice: "Bystander"})

		if app.session.hasVoice() {
			t.Fatal("the tray cast a voice that is not there")
		}
	})
}

// A source that fails is reported to the stream and the poll carries on. One unreadable
// directory must not stop the other source; a busy journal would otherwise silence the
// ship's state as well.
func TestAFailingSourceDoesNotStopTheOthers(t *testing.T) {
	app, _, _ := fixtureApp(t)
	failing := &fakeSource{name: "failing", err: errors.New("the directory is busy")}
	working := &fakeSource{name: "working"}
	app.sources = []ports.EventSource{failing, working}

	app.poll()

	if failing.polled() != 1 || working.polled() != 1 {
		t.Fatalf("polls were %d and %d, want one each", failing.polled(), working.polled())
	}
}

// startup begins the loop and shutdown ends it, releasing the device on the way out.
func TestTheLoopStartsWithTheWindowAndStopsWithIt(t *testing.T) {
	app, player, _ := fixtureApp(t)
	source := &fakeSource{name: "journal"}
	app.sources = []ports.EventSource{source}

	app.startup(context.Background())
	deadline := time.After(2 * time.Second)
	for source.polled() == 0 {
		select {
		case <-deadline:
			t.Fatal("the loop never polled its sources")
		default:
		}
	}
	app.shutdown(context.Background())

	player.mu.Lock()
	defer player.mu.Unlock()
	if player.stops == 0 {
		t.Fatal("shutting down did not release the audio device")
	}
}

// Before Wails calls startup there is no context to act on. Every window operation is
// dropped rather than panicking on a path nothing can retry.
func TestEveryWindowOperationBeforeStartupIsDropped(t *testing.T) {
	app, _, _ := fixtureApp(t)

	app.hideInWails()
	app.restoreInWails()
	app.showInWails()
	app.quitWails()
	app.emitToWails(stateEvent, nil)
}

// The dialog cannot run without a window, so asking before there is one says so rather
// than reaching into a framework that is not up yet.
func TestChoosingADirectoryBeforeTheWindowIsReadyIsRefused(t *testing.T) {
	app, _, _ := fixtureApp(t)

	if _, err := app.pickDirectory("anything", ""); err == nil {
		t.Fatal("a directory was chosen with no window to choose in")
	}
	if _, err := app.ChooseLibraryRoot(); err == nil {
		t.Fatal("a library root was chosen with no window")
	}
	if _, err := app.ChooseJournalDir(); err == nil {
		t.Fatal("a journal directory was chosen with no window")
	}
}

// Only what has been chosen is written down. Recording a detected path would freeze a
// guess, so a profile that later moves would keep pointing at where it used to be.
func TestAChoiceIsRememberedAndAFailureToRememberIsReported(t *testing.T) {
	app, _, _ := fixtureApp(t)

	// With no store there is nowhere to remember, which is not a failure.
	if err := app.rememberLibraryRoot("recordings"); err != nil {
		t.Fatalf("remembering with no store: %v", err)
	}

	store := &fakeSettings{}
	app.settings = store
	if err := app.rememberLibraryRoot("recordings"); err != nil {
		t.Fatalf("remembering the recordings directory: %v", err)
	}
	if err := app.rememberJournalDir("journal"); err != nil {
		t.Fatalf("remembering the journal directory: %v", err)
	}
	if store.held.LibraryRoot != "recordings" || store.held.JournalDir != "journal" {
		t.Fatalf("remembered %+v, want each directory kept by its own writer", store.held)
	}

	store.failure = errors.New("the disk is full")
	err := app.rememberJournalDir("journal")
	if err == nil {
		t.Fatal("a failed save reported success")
	}
	if !strings.Contains(err.Error(), "the disk is full") {
		t.Errorf("error = %q, want the store's own words carried through", err)
	}
}

// A program running from the temporary directory would be gone by the next sign-in, so
// registering it would have Windows chase a file that is deleted by the morning.
func TestALoginEntryIsRefusedForACopyRunningFromTheTemporaryDirectory(t *testing.T) {
	running, err := os.Executable()
	if err != nil {
		t.Fatalf("finding the test binary: %v", err)
	}
	if !underTemporaryDirectory(running) {
		t.Skip("this test binary is not running from the temporary directory")
	}
	app, _, _ := fixtureApp(t)

	// Only the enabling side is exercised. Disabling writes to the real login entry,
	// which is the user's own machine state and is not a test's to change.
	refusal := app.SetLaunchOnBoot(true)
	if refusal == nil {
		t.Fatal("a copy running from the temporary directory registered itself")
	}
	if !strings.Contains(refusal.Error(), "temporary directory") {
		t.Errorf("refusal = %q, want it to say why", refusal)
	}
}

// Compared case insensitively because Windows paths are, with a separator on the
// end so a sibling whose name merely starts the same is not taken for a child.
func TestOnlyRealChildrenOfTheTemporaryTreeCountAsTemporary(t *testing.T) {
	temporary := filepath.Clean(os.TempDir())

	inside := filepath.Join(temporary, "build", "app.exe")
	if !underTemporaryDirectory(inside) {
		t.Errorf("%q was not recognised as temporary", inside)
	}
	if !underTemporaryDirectory(strings.ToUpper(inside)) {
		t.Errorf("the comparison is case sensitive; Windows paths are not")
	}

	sibling := temporary + "Extra" + string(filepath.Separator) + "app.exe"
	if underTemporaryDirectory(sibling) {
		t.Errorf("%q was taken for a child of the temporary tree", sibling)
	}
	if underTemporaryDirectory(filepath.Join("C:", "Program Files", "app.exe")) {
		t.Error("an installed path was taken as temporary")
	}
}
