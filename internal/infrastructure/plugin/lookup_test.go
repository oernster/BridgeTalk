package plugin_test

// Asking a plugin voice what it can play. A plugin is somebody else's code, so every way it
// can answer badly has to end in silence rather than in a fault: a cue no voice serves is
// silence, which is always preferred to a wrong line.

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin/plugintest"
)

// loaded loads one library as the only plugin in a folder, which is how every test here
// gets a voice to ask.
func loaded(t *testing.T, library plugin.Library) *plugin.Set {
	t.Helper()
	set := plugin.Load(folder(t, "one.dll"),
		opening(map[string]plugin.Library{"one.dll": library}, nil))
	t.Cleanup(set.Close)
	if len(set.Voices()) != 1 {
		t.Fatalf("loaded %d voices and refused %+v, want one voice", len(set.Voices()), set.Refusals)
	}
	return set
}

func TestAVoiceAnswersATakeWithItsPartsInOrder(t *testing.T) {
	t.Parallel()

	voice := loaded(t, crew("Crew")).Voices()[0]

	takes, ok := voice.Lookup("StartJump")

	if !ok || len(takes) != 1 {
		t.Fatalf("takes = %v, %v; want the one take it holds", takes, ok)
	}
	want := []string{`C:\audio\a.mp3`, `C:\audio\b.mp3`, `C:\audio\c.mp3`}
	for index, part := range want {
		if takes[0][index] != part {
			t.Fatalf("take = %v, want %v in that order", takes[0], want)
		}
	}
}

// A voice knows which plugin offered it, which is what lets two voices sharing a name be
// told apart by the plugin each came from (FR-568).
func TestAVoiceKnowsWhichPluginOfferedIt(t *testing.T) {
	t.Parallel()

	set := loaded(t, crew("Bridge Crew"))
	voice := set.Voices()[0]

	if voice.Plugin() != set.Plugins[0] {
		t.Fatal("the voice does not know the plugin that offered it")
	}
	if voice.Plugin().Name != "Bridge Crew" || voice.Plugin().File != "one.dll" {
		t.Errorf("plugin = %q from %q, want the name it gave and the file it came from",
			voice.Plugin().Name, voice.Plugin().File)
	}
	if offered := set.Plugins[0].Voices(); len(offered) != 1 || offered[0] != voice {
		t.Errorf("the plugin offers %v, want the one voice", offered)
	}
}

func TestACueTheVoiceCannotServeAnswersNothing(t *testing.T) {
	t.Parallel()

	voice := loaded(t, crew("Crew")).Voices()[0]

	if takes, ok := voice.Lookup(cue.ID("NoSuchCue")); ok || takes != nil {
		t.Errorf("takes = %v, %v; want silence", takes, ok)
	}
}

// counting wraps a library and counts what it was asked, so a test can prove a question was
// never put rather than merely that its answer was ignored.
type counting struct {
	plugin.Library
	asked atomic.Int32
}

func (c *counting) Takes(voiceIndex int32, cueID []byte, buffer []byte) int32 {
	c.asked.Add(1)
	return c.Library.Takes(voiceIndex, cueID, buffer)
}

// A voice whose audio is not on this machine cannot be cast (FR-570), so a question
// reaching it is one nobody should have asked. It is not passed on.
func TestAVoiceThatIsNotReadyIsNeverAsked(t *testing.T) {
	t.Parallel()

	absent := crew("Crew")
	absent.Voices[0].Ready = false
	absent.Voices[0].Reason = "its recordings are not on this machine"
	library := &counting{Library: absent}

	voice := loaded(t, library).Voices()[0]
	takes, ok := voice.Lookup("StartJump")

	if ok || takes != nil {
		t.Errorf("takes = %v, %v; want silence", takes, ok)
	}
	if asked := library.asked.Load(); asked != 0 {
		t.Errorf("the plugin was asked %d times about a voice that is not ready", asked)
	}
	if voice.Reason == "" {
		t.Error("the voice carries no reason, so nothing could tell the user why")
	}
}

// liar answers one size then writes another, which is the shape of a plugin that changed
// its mind between the two calls. Reading the buffer anyway would turn a truncated list into
// a shorter one that looks complete.
type liar struct {
	plugin.Library
	writes int32
}

func (l *liar) Takes(voiceIndex int32, cueID []byte, buffer []byte) int32 {
	if len(buffer) == 0 {
		return l.Library.Takes(voiceIndex, cueID, nil)
	}
	l.Library.Takes(voiceIndex, cueID, buffer)
	return l.writes
}

func TestAPluginThatAnswersADifferentSizeIsNotRead(t *testing.T) {
	t.Parallel()

	for _, each := range []struct {
		name   string
		writes int32
	}{
		{"it writes fewer bytes than it asked for", 4},
		{"it writes more bytes than it asked for", 4096},
		{"it refuses the second call", plugintest.Refused},
	} {
		t.Run(each.name, func(t *testing.T) {
			t.Parallel()

			voice := loaded(t, &liar{Library: crew("Crew"), writes: each.writes}).Voices()[0]

			if takes, ok := voice.Lookup("StartJump"); ok || takes != nil {
				t.Errorf("takes = %v, %v; want silence", takes, ok)
			}
		})
	}
}

func TestAPluginRefusingToAnswerIsSilence(t *testing.T) {
	t.Parallel()

	refusing := crew("Crew")
	refusing.RefuseTakes = true

	voice := loaded(t, refusing).Voices()[0]

	if takes, ok := voice.Lookup("StartJump"); ok || takes != nil {
		t.Errorf("takes = %v, %v; want silence", takes, ok)
	}
}

// After the set is closed the thread is gone. A cue firing as the application shuts down is
// a race nobody should have to hear about, so it answers silence rather than panicking.
func TestAskingAfterCloseIsSilenceRatherThanAPanic(t *testing.T) {
	t.Parallel()

	set := plugin.Load(folder(t, "one.dll"),
		opening(map[string]plugin.Library{"one.dll": crew("Crew")}, nil))
	voice := set.Voices()[0]
	set.Close()

	if takes, ok := voice.Lookup("StartJump"); ok || takes != nil {
		t.Errorf("takes = %v, %v; want silence", takes, ok)
	}
}

// watching records how many calls are inside the library at once, which is what proves the
// thread serialises them.
type watching struct {
	plugin.Library
	inside atomic.Int32
	most   atomic.Int32
}

func (w *watching) Takes(voiceIndex int32, cueID []byte, buffer []byte) int32 {
	now := w.inside.Add(1)
	for {
		most := w.most.Load()
		if now <= most || w.most.CompareAndSwap(most, now) {
			break
		}
	}
	answer := w.Library.Takes(voiceIndex, cueID, buffer)
	w.inside.Add(-1)
	return answer
}

// Calls arrive one at a time, so a plugin author needs no locking of their own
// (PLUGINS-GUIDE.md). Without the thread these would overlap, since the poll loop and the
// window both ask.
func TestCallsIntoAPluginNeverOverlap(t *testing.T) {
	t.Parallel()

	library := &watching{Library: crew("Crew")}
	voice := loaded(t, library).Voices()[0]

	var waiting sync.WaitGroup
	for range 32 {
		waiting.Add(1)
		go func() {
			defer waiting.Done()
			voice.Lookup("StartJump")
		}()
	}
	waiting.Wait()

	if most := library.most.Load(); most != 1 {
		t.Errorf("%d calls were inside the plugin at once, want 1", most)
	}
}
