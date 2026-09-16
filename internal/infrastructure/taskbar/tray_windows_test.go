//go:build windows

package taskbar

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// tipWait is how long a posted refresh may take to reach the shell before the test gives up.
const tipWait = 2 * time.Second

// FR-710: the hover text follows a mute and a cast from whichever goroutine makes them, sent
// again on the tray's own thread once the state has changed. A menu choice does not send it
// itself, since the choice has not been acted on when the menu returns and the text would be
// the state before. A real window and message loop run here; only the shell is replaced, so
// no icon appears.
func TestTheHoverTextFollowsTheStateOnTheTrayThread(t *testing.T) {
	tray := newTestTray([]string{"Grace", "Jack"}, "Grace")
	sent := make(chan string, commandBuffer)
	tray.notify = func(message uintptr, data *notifyIconData) error {
		if message == nimModify {
			sent <- windows.UTF16ToString(data.szTip[:])
		}
		return nil
	}
	if err := tray.Start(); err != nil {
		t.Fatalf("starting the tray: %v", err)
	}
	defer func() {
		tray.Stop()
		for range tray.Commands() {
		}
	}()

	awaitTip := func(want string) {
		t.Helper()
		select {
		case got := <-sent:
			if got != want {
				t.Fatalf("hover text sent = %q, want %q", got, want)
			}
		case <-time.After(tipWait):
			t.Fatalf("no hover text was sent; want %q", want)
		}
	}
	tray.SetMuted(true)
	awaitTip("Test: Grace (muted)")
	tray.SetActiveVoice(Voice{Name: "Jack"}, "Jack")
	awaitTip("Test: Jack (muted)")

	tray.dispatch(idMute)
	select {
	case got := <-sent:
		t.Fatalf("the menu choice sent %q before it was acted on", got)
	default:
	}
}

// newTestTray builds a tray without touching Win32, so the menu logic can be tested
// without a message loop or a shell. Each voice is shown by its own name.
func newTestTray(voices []string, active string) *Tray {
	choices := make([]Choice, 0, len(voices))
	for _, name := range voices {
		choices = append(choices, Choice{Voice: Voice{Name: name}, Label: name})
	}
	return New(Options{Title: "Test", Voices: choices, Active: Voice{Name: active}})
}

// FR-210: the menu and the hover text show each voice by its label, while a choice carries
// the name that identifies it and the check mark follows that name.
func TestTheMenuShowsEachVoiceByTheNameItIsShownBy(t *testing.T) {
	tray := New(Options{Title: "Test", Active: Voice{Name: "leo"}, Voices: []Choice{
		{Voice: Voice{Name: "grace"}, Label: "Grace Hart"},
		{Voice: Voice{Name: "leo"}, Label: "Leo Marsh"},
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
	tray.SetActiveVoice(Voice{Name: "carol"}, "Carol Hart")
	if got, want := tray.tooltip(), "Test: Carol Hart"; got != want {
		t.Errorf("tooltip = %q, want a voice the menu does not hold shown by the label handed over", got)
	}
	tray.SetActiveVoice(Voice{Name: "nobody listed"}, "")
	if got, want := tray.tooltip(), "Test: nobody listed"; got != want {
		t.Errorf("tooltip = %q, want a blank label shown by the name as it is", got)
	}

	tray.dispatch(idVoiceBase)
	if got := <-tray.commands; got != (Command{Kind: CommandSelectVoice, Chosen: Voice{Name: "grace"}}) {
		t.Errorf("command = %+v, want grace chosen by the name that identifies her", got)
	}
}

// FR-509 and FR-540: the machine voices follow the recorded voices under a separator, each cast by
// its id; a recordings folder carrying the same name is a different voice, checked and named apart.
func TestTheMenuListsMachineVoicesAfterTheRecordedVoices(t *testing.T) {
	tray := New(Options{
		Title:  "Test",
		Active: Voice{Kind: Machine, Name: "bf_emma"},
		Voices: []Choice{
			{Voice: Voice{Name: "bf_emma"}, Label: "bf_emma"},
			{Voice: Voice{Kind: Machine, Name: "bf_emma"}, Label: "Emma (British, female)"},
			{Voice: Voice{Kind: Machine, Name: "am_adam"}, Label: "Adam (American, male)"},
		},
	})

	want := []menuVoice{
		{id: idVoiceBase, label: "bf_emma"},
		{id: idVoiceBase + 1, label: "Emma (British, female)", checked: true, separated: true},
		{id: idVoiceBase + 2, label: "Adam (American, male)"},
	}
	if got := tray.voiceItems(); !reflect.DeepEqual(got, want) {
		t.Errorf("items = %+v, want %+v", got, want)
	}
	if got, want := tray.tooltip(), "Test: Emma (British, female)"; got != want {
		t.Errorf("tooltip = %q, want %q", got, want)
	}
	tray.SetActiveVoice(Voice{Name: "bf_emma"}, "bf_emma")
	if got, want := tray.tooltip(), "Test: bf_emma"; got != want {
		t.Errorf("tooltip = %q, want the recorded voice's own label", got)
	}

	tray.dispatch(idVoiceBase + 1)
	machineEmma := Command{Kind: CommandSelectVoice, Chosen: Voice{Kind: Machine, Name: "bf_emma"}}
	if got := <-tray.commands; got != machineEmma {
		t.Errorf("command = %+v, want the machine voice cast by its id", got)
	}
	tray.dispatch(idVoiceBase)
	if got := <-tray.commands; got != (Command{Kind: CommandSelectVoice, Chosen: Voice{Name: "bf_emma"}}) {
		t.Errorf("command = %+v, want the recorded voice cast by its name", got)
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
		{"first voice", idVoiceBase, Command{Kind: CommandSelectVoice, Chosen: Voice{Name: "Grace"}}},
		{"last voice", idVoiceBase + 2, Command{Kind: CommandSelectVoice, Chosen: Voice{Name: "Leo"}}},
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
	tray.SetActiveVoice(Voice{Name: "Jack"}, "Jack")
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

// FR-710: the hover text says whether playback is muted with no voice cast too, since the mute
// answers with nothing cast.
func TestTheHoverTextSaysMutedWithNoVoiceCast(t *testing.T) {
	tray := New(Options{Title: "Test"})
	tray.SetMuted(true)
	if got, want := tray.tooltip(), "Test (muted)"; got != want {
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

// FR-565 and FR-569: the plugin voices follow the machine voices under a separator of their own,
// and the cast one is checked by its plugin as well as its id, since two plugins may offer a voice
// under one id.
func TestTheMenuListsPluginVoicesAfterTheMachineVoices(t *testing.T) {
	tray := New(Options{
		Title:  "Test",
		Active: Voice{Kind: Plugin, Plugin: "Flight Deck", Name: "one"},
		Voices: []Choice{
			{Voice: Voice{Name: "Grace"}, Label: "Grace Hart"},
			{Voice: Voice{Kind: Machine, Name: "bf_emma"}, Label: "Emma (British, female)"},
			{Voice: Voice{Kind: Plugin, Plugin: "Bridge Crew", Name: "one"}, Label: "Officer (Bridge Crew)"},
			{Voice: Voice{Kind: Plugin, Plugin: "Flight Deck", Name: "one"}, Label: "Officer (Flight Deck)"},
		},
	})

	want := []menuVoice{
		{id: idVoiceBase, label: "Grace Hart"},
		{id: idVoiceBase + 1, label: "Emma (British, female)", separated: true},
		{id: idVoiceBase + 2, label: "Officer (Bridge Crew)", separated: true},
		{id: idVoiceBase + 3, label: "Officer (Flight Deck)", checked: true},
	}
	if got := tray.voiceItems(); !reflect.DeepEqual(got, want) {
		t.Errorf("items = %+v, want %+v", got, want)
	}
	if got, want := tray.tooltip(), "Test: Officer (Flight Deck)"; got != want {
		t.Errorf("tooltip = %q, want %q", got, want)
	}

	tray.dispatch(idVoiceBase + 2)
	chosen := Command{
		Kind:   CommandSelectVoice,
		Chosen: Voice{Kind: Plugin, Plugin: "Bridge Crew", Name: "one"},
	}
	if got := <-tray.commands; got != chosen {
		t.Errorf("command = %+v, want the other plugin's voice of the same id", got)
	}
}
