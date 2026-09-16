package plugintest_test

// The test plugin stands in for a library file, so it has to obey the contract a library
// file obeys. A fake nobody tests can drift into answering the way its user happens to ask,
// which would leave the adapter proved against a plugin no real one resembles. These hold
// it to PLUGINS-GUIDE.md: the size before the answer, a negative for a refusal and never
// for a size, then nothing written into a buffer that cannot hold the whole answer.

import (
	"bytes"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin/plugintest"
)

// crew is a plugin offering one voice that can answer two cues, one of them with a take
// recorded in three pieces.
func crew() *plugintest.Plugin {
	return &plugintest.Plugin{
		Name: "Bridge Crew",
		Voices: []plugintest.Voice{{
			ID: "one", Name: "The First Officer", Ready: true,
			Answers: map[string][][]string{
				"DockingGranted": {{`C:\audio\granted.mp3`}},
				"StartJump":      {{`C:\audio\a.mp3`, `C:\audio\b.mp3`, `C:\audio\c.mp3`}},
			},
		}},
	}
}

func TestAPluginStatesTheVersionItWasBuiltAgainst(t *testing.T) {
	t.Parallel()

	if got := crew().Version(); got != plugintest.ABIVersion {
		t.Errorf("version = %d, want %d by default", got, plugintest.ABIVersion)
	}
	if got := (&plugintest.Plugin{ABI: 7}).Version(); got != 7 {
		t.Errorf("version = %d, want the one it was given", got)
	}
	if got := (&plugintest.Plugin{RefuseVersion: true}).Version(); got != plugintest.Refused {
		t.Errorf("version = %d, want a refusal", got)
	}
}

func TestAnAnswerIsSizedBeforeItIsFilled(t *testing.T) {
	t.Parallel()

	subject := crew()

	size := subject.Describe(nil)
	if size <= 0 {
		t.Fatalf("size = %d, want the bytes the description needs", size)
	}

	buffer := make([]byte, size)
	if written := subject.Describe(buffer); written != size {
		t.Fatalf("wrote %d bytes into a buffer of %d", written, size)
	}

	got, err := plugin.DecodeDescription(buffer)
	if err != nil || got.Name != "Bridge Crew" || len(got.Voices) != 1 {
		t.Errorf("the filled buffer read as %+v, %v", got, err)
	}
}

func TestABufferTooSmallIsRefusedRatherThanPartlyFilled(t *testing.T) {
	t.Parallel()

	subject := crew()
	short := make([]byte, subject.Describe(nil)-1)

	if got := subject.Describe(short); got != plugintest.Refused {
		t.Fatalf("describe = %d, want a refusal", got)
	}
	for index, written := range short {
		if written != 0 {
			t.Fatalf("byte %d was written into a buffer that was refused", index)
		}
	}
}

func TestACueIsAnsweredWithItsTakesAndAnUnknownCueWithNone(t *testing.T) {
	t.Parallel()

	subject := crew()

	for _, each := range []struct {
		cue   string
		takes int
		parts int
	}{
		{"StartJump", 1, 3},
		{"DockingGranted", 1, 1},
		{"NoSuchCue", 0, 0},
	} {
		t.Run(each.cue, func(t *testing.T) {
			t.Parallel()

			id := []byte(each.cue)
			buffer := make([]byte, subject.Takes(0, id, nil))
			if written := subject.Takes(0, id, buffer); written != int32(len(buffer)) {
				t.Fatalf("wrote %d bytes into a buffer of %d", written, len(buffer))
			}

			got, err := plugin.DecodeTakes(buffer)
			if err != nil {
				t.Fatalf("reading takes: %v", err)
			}
			if len(got) != each.takes {
				t.Fatalf("takes = %v, want %d", got, each.takes)
			}
			if each.takes > 0 && len(got[0]) != each.parts {
				t.Errorf("the take has %d parts, want %d", len(got[0]), each.parts)
			}
		})
	}
}

func TestAVoiceThePluginNeverOfferedIsRefused(t *testing.T) {
	t.Parallel()

	subject := crew()

	for _, index := range []int32{-1, 1, 99} {
		if got := subject.Takes(index, []byte("StartJump"), nil); got != plugintest.Refused {
			t.Errorf("voice %d answered %d, want a refusal", index, got)
		}
	}
}

func TestARefusingPluginRefuses(t *testing.T) {
	t.Parallel()

	if got := (&plugintest.Plugin{RefuseDescribe: true}).Describe(nil); got != plugintest.Refused {
		t.Errorf("describe = %d, want a refusal", got)
	}
	subject := crew()
	subject.RefuseTakes = true
	if got := subject.Takes(0, []byte("StartJump"), nil); got != plugintest.Refused {
		t.Errorf("takes = %d, want a refusal", got)
	}
}

func TestACorruptPluginWritesSomethingThatIsNoLayout(t *testing.T) {
	t.Parallel()

	subject := crew()
	subject.Corrupt = true

	buffer := make([]byte, subject.Describe(nil))
	subject.Describe(buffer)

	if _, err := plugin.DecodeDescription(buffer); err == nil {
		t.Error("a corrupt description read back without complaint")
	}
}

// The standalone builder and a plugin answering for itself write the same bytes, so a test
// reading one layout directly is reading the layout a plugin really writes.
func TestTheStandaloneBuilderWritesWhatAPluginWrites(t *testing.T) {
	t.Parallel()

	subject := crew()
	id := []byte("StartJump")
	fromPlugin := make([]byte, subject.Takes(0, id, nil))
	subject.Takes(0, id, fromPlugin)

	standalone := plugintest.Takes([]string{`C:\audio\a.mp3`, `C:\audio\b.mp3`, `C:\audio\c.mp3`})

	if !bytes.Equal(fromPlugin, standalone) {
		t.Errorf("the builder wrote %v, the plugin wrote %v", standalone, fromPlugin)
	}
}

func TestAVoiceThatIsNotReadyCarriesItsReason(t *testing.T) {
	t.Parallel()

	written := plugintest.Description("Crew", plugintest.Voice{
		ID: "two", Name: "The Second", Reason: "its recordings are not on this machine",
	})

	got, err := plugin.DecodeDescription(written)

	if err != nil || len(got.Voices) != 1 {
		t.Fatalf("description = %+v, %v", got, err)
	}
	if got.Voices[0].Ready || got.Voices[0].Reason == "" {
		t.Errorf("voice = %+v, want it not ready with a reason", got.Voices[0])
	}
}
