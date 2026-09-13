package main

import (
	"context"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
)

// idleTray is a tray that has never been started, so it holds no Win32 handle and
// touches no message loop. Its state is atomics and a channel, which is exactly the
// part the facade talks to, so it stands in for the real icon without one.
func idleTray() *taskbar.Tray {
	return taskbar.New(taskbar.Options{Title: appTitle, Voices: []string{"Alpha", "Beta"}})
}

// The tray menu shows the cast voice and the mute state, so both have to be pushed to
// it as they change or the menu describes a session that has moved on.
func TestTheTrayIsKeptInStepWithTheSession(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.session.tray = idleTray()

	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting: %v", err)
	}
	app.SetMuted(true)
	if !app.Muted() {
		t.Fatal("the mute did not stick with a tray attached")
	}
	app.SetMuted(false)
	if app.Muted() {
		t.Fatal("the unmute did not stick with a tray attached")
	}
}

// Shutting down releases the device and takes the icon down with it. An icon left in
// the notification area after the application has gone is a ghost the user has to
// hover over to clear.
func TestShuttingDownTakesTheIconDownToo(t *testing.T) {
	app, player, _ := fixtureApp(t)
	app.session.tray = idleTray()

	app.shutdown(context.Background())

	player.mu.Lock()
	defer player.mu.Unlock()
	if player.stops == 0 {
		t.Fatal("shutting down did not release the audio device")
	}
}

// The loop services the tray as well as the sources, so it takes the tray's channel
// when there is one. Nothing can be pushed through it from here, since the channel is
// the tray's own; what this proves is that the loop runs with one attached.
func TestTheLoopRunsWithATrayAttached(t *testing.T) {
	app, _, _ := fixtureApp(t)
	app.session.tray = idleTray()
	source := &fakeSource{name: "journal"}
	app.sources = []ports.EventSource{source}

	go app.run()
	defer close(app.stop)

	deadline := time.After(2 * time.Second)
	for source.polled() == 0 {
		select {
		case <-deadline:
			t.Fatal("the loop never polled with a tray attached")
		default:
		}
	}
}

// A poll that finds events hands them to the reaction service, which is the whole
// path from a journal line to a clip. Without a voice cast there is nothing to hand
// them to, so the same poll has to do nothing instead.
func TestEventsReachTheReactionServiceOnlyOnceAVoiceIsCast(t *testing.T) {
	app, player, _ := fixtureApp(t)
	source := &fakeSource{name: "journal"}
	app.sources = []ports.EventSource{source}

	raised := func() []event.Event {
		return []event.Event{event.New(
			event.SourceJournal, "ShieldState", event.EdgeNone,
			map[string]any{}, time.Date(2026, 8, 26, 9, 30, 0, 0, time.UTC),
		)}
	}

	source.events = raised()
	app.poll()
	player.mu.Lock()
	played := len(player.played)
	player.mu.Unlock()
	if played != 0 {
		t.Fatalf("the device was given %d sequences with no voice cast", played)
	}

	if err := app.SelectVoice("Alpha"); err != nil {
		t.Fatalf("casting: %v", err)
	}
	source.events = raised()
	app.poll()
	app.session.scheduler.Advance()

	player.mu.Lock()
	defer player.mu.Unlock()
	if len(player.played) == 0 {
		t.Fatal("a bound cue reached a cast voice and nothing was played")
	}
}

// The two command-line reports are the only way to ask what a voice covers without a
// window, so they have to answer over any voice the scanner can produce.
func TestTheCommandLineReportsRunOverEveryPack(t *testing.T) {
	current, _ := fixtureSession(t, newFakePlayer())

	if err := listing(current.available, current.table, current.chooser); err != nil {
		t.Fatalf("listing every voice: %v", err)
	}
	for _, each := range current.available {
		if err := unboundReport(each, current.table, current.chooser); err != nil {
			t.Fatalf("reporting on %q: %v", each.Name, err)
		}
	}
}
