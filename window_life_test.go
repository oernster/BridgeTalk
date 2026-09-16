package main

import (
	"context"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
)

// saw reports whether an event was announced at any point, for the events raised
// before a test can start waiting on one.
func (r *recorder) saw(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, event := range r.events {
		if event.name == name {
			return true
		}
	}
	return false
}

// TestTheCrossAsksRatherThanClosing covers the close choice.
//
// The window used to close outright, which ended a resident application the commander
// expected to still be listening. The cross now cancels its own close and asks. True
// from beforeClose is what cancels it, so the reading here is deliberately the way
// round Wails means it rather than the way round the word "close" suggests.
func TestTheCrossAsksRatherThanClosing(t *testing.T) {
	player := newFakePlayer()
	app, log := newTestApp(t, player)
	app.session.tray = &taskbar.Tray{}
	raised := 0
	app.show = func() { raised++ }

	if prevented := app.beforeClose(context.Background()); !prevented {
		t.Error("the close went ahead, so the cross ended the application without asking")
	}
	if raised != 1 {
		t.Errorf("the window was raised %d times, want once before the question is drawn", raised)
	}
	if !log.saw(closeRequestEvent) {
		t.Error("nothing asked the page for the close choice, so the cross would do nothing at all")
	}
}

// TestAQuitAlreadyDecidedIsNotAskedAboutAgain covers the flag: without it the dialog
// would ask about the quit it was just told to perform.
func TestAQuitAlreadyDecidedIsNotAskedAboutAgain(t *testing.T) {
	player := newFakePlayer()
	app, _ := newTestApp(t, player)
	app.session.tray = &taskbar.Tray{}
	quits := 0
	app.quit = func() { quits++ }
	app.show = func() {}

	app.Quit()
	if quits != 1 {
		t.Errorf("quit ran %d times, want once", quits)
	}
	if prevented := app.beforeClose(context.Background()); prevented {
		t.Error("the close was cancelled after a quit was decided, so the dialog would ask again")
	}
}

// TestTheCrossClosesWithNowhereToHide covers the case where the question has no good
// answer: with no tray icon, hiding the window would leave the application running
// with nothing on screen to bring it back.
func TestTheCrossClosesWithNowhereToHide(t *testing.T) {
	player := newFakePlayer()
	app, log := newTestApp(t, player)
	app.session.tray = nil

	if prevented := app.beforeClose(context.Background()); prevented {
		t.Error("the close was cancelled with no tray to hide into")
	}
	if log.saw(closeRequestEvent) {
		t.Error("the page was asked to choose with no tray to choose")
	}
}

// TestMinimiseToTrayHidesWithoutEnding proves the dialog's other answer keeps the
// application running, since hiding and quitting are the two things it must not mix up.
func TestMinimiseToTrayHidesWithoutEnding(t *testing.T) {
	player := &fakePlayer{}
	app, _ := newTestApp(t, player)
	hides, quits := 0, 0
	app.hide = func() { hides++ }
	app.quit = func() { quits++ }

	app.MinimiseToTray()
	if hides != 1 || quits != 0 {
		t.Errorf("hides = %d and quits = %d, want the window put away and the run left alone", hides, quits)
	}

	app.RequestQuit()
	if quits != 1 {
		t.Errorf("quits = %d after RequestQuit, want one", quits)
	}
}

// TestTheTrayIconBringsTheWindowBack covers the only route back to a window that has
// been put away. Both mouse gestures and the menu's Open all arrive as one command,
// so the facade is asserted on the command rather than on the gesture.
func TestTheTrayIconBringsTheWindowBack(t *testing.T) {
	player := newFakePlayer()
	app, _ := newTestApp(t, player)
	restored, quits := 0, 0
	app.restore = func() { restored++ }
	app.quit = func() { quits++ }

	app.handleTray(taskbar.Command{Kind: taskbar.CommandShow})

	if restored != 1 {
		t.Errorf("the window was restored %d times, want once: the icon is the only way back", restored)
	}
	if quits != 0 {
		t.Error("asking for the window back ended the application instead")
	}
}

// TestASummonedWindowIsToldToOpenOnTheCast covers where the window comes back to.
//
// Hiding a window does not reload the page, so it returns on whichever pane was open
// when it was put away, which is usually the one that was being fiddled with rather
// than the one worth seeing. The facade says so on the way back rather than the page
// guessing, because only the facade knows a summons from an ordinary raise.
func TestASummonedWindowIsToldToOpenOnTheCast(t *testing.T) {
	player := newFakePlayer()
	app, log := newTestApp(t, player)
	app.restore = func() {}

	app.handleTray(taskbar.Command{Kind: taskbar.CommandShow})

	if !log.saw(windowShownEvent) {
		t.Error("nothing told the page the window was summoned, so it opens wherever it was left")
	}
}

// FR-814: a tray the desktop never took is no tray. The cross then closes rather than asking to hide
// into nothing; a window started hidden for it is shown with the keyboard; one already showing is
// left alone.
func TestATrayTheDesktopNeverTookIsNoTray(t *testing.T) {
	for _, hidden := range []bool{true, false} {
		app, log := newTestApp(t, newFakePlayer())
		app.session.tray = &taskbar.Tray{}
		app.startedHidden = hidden
		restored := 0
		app.restore = func() { restored++ }

		app.handleTray(taskbar.Command{Kind: taskbar.CommandNoTray})

		if prevented := app.beforeClose(context.Background()); prevented {
			t.Errorf("hidden %v: the close was cancelled with no tray to hide into", hidden)
		}
		if log.saw(closeRequestEvent) {
			t.Errorf("hidden %v: the page was asked to choose with no tray to choose", hidden)
		}
		if want := map[bool]int{true: 1, false: 0}[hidden]; restored != want {
			t.Errorf("hidden %v: the window was brought back %d times, want %d", hidden, restored, want)
		}
		if hidden && (!log.saw(windowShownEvent) || !app.broughtBack.Load()) {
			t.Error("a window shown for a tray that never came was not announced or cannot take the keyboard")
		}
	}
}
