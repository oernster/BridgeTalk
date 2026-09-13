//go:build windows

package taskbar

import (
	"reflect"
	"testing"
)

// newTestTray builds a tray without touching Win32, so the menu logic can be tested
// without a message loop or a shell. Each voice is shown by its own name.
func newTestTray(voices []string, active string) *Tray {
	choices := make([]Choice, 0, len(voices))
	for _, name := range voices {
		choices = append(choices, Choice{Name: name, Label: name})
	}
	return New(Options{Title: "Test", Voices: choices, ActiveVoice: active})
}

// FR-210: the menu and the hover text show each voice by its label, while a choice carries
// the name that identifies it and the check mark follows that name.
func TestTheMenuShowsEachVoiceByTheNameItIsShownBy(t *testing.T) {
	tray := New(Options{Title: "Test", ActiveVoice: "leo", Voices: []Choice{
		{Name: "grace", Label: "Grace Hart"},
		{Name: "leo", Label: "Leo Marsh"},
	}})

	want := []menuVoice{
		{id: idVoiceBase, label: "Grace Hart", checked: false},
		{id: idVoiceBase + 1, label: "Leo Marsh", checked: true},
	}
	if got := tray.voiceItems(); !reflect.DeepEqual(got, want) {
		t.Errorf("items = %+v, want %+v", got, want)
	}
	if got, want := tray.tooltip(), "Test: Leo Marsh"; got != want {
		t.Errorf("tooltip = %q, want %q", got, want)
	}
	tray.SetActiveVoice("nobody listed")
	if got, want := tray.tooltip(), "Test: nobody listed"; got != want {
		t.Errorf("tooltip = %q, want an unlisted name shown as it is", got)
	}

	tray.dispatch(idVoiceBase)
	if got := <-tray.commands; got != (Command{Kind: CommandSelectVoice, Voice: "grace"}) {
		t.Errorf("command = %+v, want grace chosen by the name that identifies her", got)
	}
}

func TestDispatchMapsMenuIdentifiers(t *testing.T) {
	voices := []string{"Grace", "Jack", "Leo"}
	cases := []struct {
		name   string
		chosen uint32
		want   Command
	}{
		{"mute", idMute, Command{Kind: CommandToggleMute}},
		{"quit", idQuit, Command{Kind: CommandQuit}},
		{"first voice", idVoiceBase, Command{Kind: CommandSelectVoice, Voice: "Grace"}},
		{"last voice", idVoiceBase + 2, Command{Kind: CommandSelectVoice, Voice: "Leo"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			tray := newTestTray(voices, "Grace")
			tray.dispatch(test.chosen)
			select {
			case got := <-tray.commands:
				if got != test.want {
					t.Fatalf("command = %+v, want %+v", got, test.want)
				}
			default:
				t.Fatal("no command was queued")
			}
		})
	}
}

func TestDispatchIgnoresNothingAndOutOfRange(t *testing.T) {
	voices := []string{"Grace"}
	// Zero is what TrackPopupMenu returns when the menu is dismissed without a
	// choice, which happens on every click outside the menu and must be silent.
	for _, chosen := range []uint32{0, 99, idVoiceBase + 5} {
		tray := newTestTray(voices, "Grace")
		tray.dispatch(chosen)
		select {
		case got := <-tray.commands:
			t.Fatalf("identifier %d produced %+v, want nothing", chosen, got)
		default:
		}
	}
}

func TestDispatchDoesNotBlockWhenNobodyIsReading(t *testing.T) {
	tray := newTestTray([]string{"Grace"}, "Grace")
	// A main loop that has stopped reading must not freeze the menu, so sends are
	// dropped once the buffer fills rather than blocking the tray thread.
	for range commandBuffer * 3 {
		tray.dispatch(idMute)
	}
	if len(tray.commands) != commandBuffer {
		t.Fatalf("queued %d commands, want the buffer to cap at %d", len(tray.commands), commandBuffer)
	}
}

func TestTooltipReflectsVoiceAndMuteState(t *testing.T) {
	tray := newTestTray([]string{"Grace"}, "Grace")
	if got, want := tray.tooltip(), "Test: Grace"; got != want {
		t.Fatalf("tooltip = %q, want %q", got, want)
	}
	tray.SetMuted(true)
	if got, want := tray.tooltip(), "Test: Grace (muted)"; got != want {
		t.Fatalf("muted tooltip = %q, want %q", got, want)
	}
	tray.SetActiveVoice("Jack")
	if got, want := tray.tooltip(), "Test: Jack (muted)"; got != want {
		t.Fatalf("after switching voice tooltip = %q, want %q", got, want)
	}
}

func TestTooltipFallsBackToTheTitle(t *testing.T) {
	tray := New(Options{Title: "Test"})
	if got, want := tray.tooltip(), "Test"; got != want {
		t.Fatalf("tooltip = %q, want %q", got, want)
	}
}

func TestStopIsSafeBeforeStart(t *testing.T) {
	// Stop runs from a deferred call, so it must tolerate a tray that never started.
	tray := newTestTray([]string{"Grace"}, "Grace")
	tray.Stop()
	tray.Stop()
}

// TestAClickAsksForTheWindowBack covers the icon's mouse gestures.
//
// Both buttons opened the menu while the window could not be hidden. Now that it can,
// a window put away here is reachable only through this icon. The gesture tried
// first is a plain click; a double click is the same intent pressed twice.
func TestAClickAsksForTheWindowBack(t *testing.T) {
	for _, message := range []uintptr{wmLButtonUp, wmLButtonDblClk} {
		tray := newTestTray([]string{"Grace"}, "Grace")
		tray.windowProc(0, wmTrayCallback, 0, message)
		select {
		case got := <-tray.commands:
			if got.Kind != CommandShow {
				t.Errorf("a click asked for %v, want the window back", got.Kind)
			}
		default:
			t.Errorf("a click on the icon asked for nothing at all, so it reads as broken")
		}
	}
}

// TestTheMenuOffersTheWindowToo keeps a route back that does not need the mouse.
func TestTheMenuOffersTheWindowToo(t *testing.T) {
	tray := newTestTray([]string{"Grace"}, "Grace")
	tray.dispatch(idShow)
	select {
	case got := <-tray.commands:
		if got.Kind != CommandShow {
			t.Errorf("the menu's Open asked for %v, want the window back", got.Kind)
		}
	default:
		t.Error("the menu's Open asked for nothing at all")
	}
}
