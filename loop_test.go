package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
)

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

		app.handleTray(taskbar.Command{
			Kind: taskbar.CommandSelectVoice, Chosen: taskbar.Voice{Name: "Alpha"},
		})

		if app.session.active.Name != "Alpha" {
			t.Fatalf("active voice is %q, want Alpha", app.session.active.Name)
		}
	})

	// FR-509: a machine voice chosen from the tray is cast as one, by its id.
	t.Run("selecting a machine voice casts it", func(t *testing.T) {
		app, _, _ := fixtureApp(t)

		app.handleTray(taskbar.Command{
			Kind:   taskbar.CommandSelectVoice,
			Chosen: taskbar.Voice{Kind: taskbar.Machine, Name: "bf_emma"},
		})

		if app.session.active != (castVoice{Name: "bf_emma", Display: "Emma (British, female)", Machine: true}) {
			t.Fatalf("active voice is %+v, want bf_emma cast as a machine voice", app.session.active)
		}
	})

	t.Run("a voice the tray cannot cast is not fatal", func(t *testing.T) {
		app, _, _ := fixtureApp(t)

		app.handleTray(taskbar.Command{
			Kind: taskbar.CommandSelectVoice, Chosen: taskbar.Voice{Name: "Bystander"},
		})

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
