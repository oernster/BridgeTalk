package plugin_test

// What a plugin says arrives as bytes from somebody else's code. These tests read the
// layouts back that plugintest writes, then work through every way an answer can be wrong:
// short, negative, oversized, not text, longer than it said. None of them may panic and
// none may reach the application as a voice.

import (
	"errors"
	"reflect"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/take"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin/plugintest"
)

func TestADescriptionReadsBackAsItWasWritten(t *testing.T) {
	t.Parallel()

	written := plugintest.Description("Bridge Crew",
		plugintest.Voice{ID: "one", Name: "The First Officer", Ready: true},
		plugintest.Voice{ID: "two", Name: "The Second", Reason: "its recordings are not on this machine"},
	)

	got, err := plugin.DecodeDescription(written)

	if err != nil {
		t.Fatalf("reading a description: %v", err)
	}
	want := plugin.Description{
		Name: "Bridge Crew",
		Voices: []plugin.VoiceInfo{
			{ID: "one", Name: "The First Officer", Ready: true},
			{ID: "two", Name: "The Second", Reason: "its recordings are not on this machine"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("description = %+v, want %+v", got, want)
	}
}

func TestAPluginOfferingNoVoiceIsReadWithoutComplaint(t *testing.T) {
	t.Parallel()

	got, err := plugin.DecodeDescription(plugintest.Description("Empty"))

	if err != nil || got.Name != "Empty" || len(got.Voices) != 0 {
		t.Errorf("description = %+v, %v; want a named plugin offering nothing", got, err)
	}
}

func TestTakesReadBackWithTheirPartsInOrder(t *testing.T) {
	t.Parallel()

	written := plugintest.Takes(
		[]string{`C:\audio\alone.mp3`},
		[]string{`C:\audio\one.mp3`, `C:\audio\two.mp3`, `C:\audio\three.mp3`},
	)

	got, err := plugin.DecodeTakes(written)

	if err != nil {
		t.Fatalf("reading takes: %v", err)
	}
	want := []take.Take{
		take.Of(`C:\audio\alone.mp3`),
		take.Of(`C:\audio\one.mp3`, `C:\audio\two.mp3`, `C:\audio\three.mp3`),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("takes = %v, want %v", got, want)
	}
}

func TestACueTheVoiceCannotServeReadsAsNoTakes(t *testing.T) {
	t.Parallel()

	got, err := plugin.DecodeTakes(plugintest.Takes())

	if err != nil || len(got) != 0 {
		t.Errorf("takes = %v, %v; want nothing and no complaint", got, err)
	}
}

// Every refusal below names what was wrong. The message is not pinned word for word, since
// it is prose that may improve; what is pinned is that it is a refusal rather than a panic,
// a partial answer or an accepted one.
func TestAMalformedDescriptionIsRefused(t *testing.T) {
	t.Parallel()

	for _, each := range []struct {
		name string
		data []byte
	}{
		{"nothing at all", nil},
		{"it ends before its voice count", []byte{1, 0}},
		{"it promises a voice it has no bytes for", counted(1)},
		{"a length runs past the end", append(counted(0), counted(9)...)},
		{"a length is negative", append(counted(0), counted(-1)...)},
		{"a name is not text", append(counted(0), text([]byte{0xff, 0xfe})...)},
		{"a voice is malformed after a good name", join(counted(1), text([]byte("A")), counted(99))},
		{"bytes follow the description", append(plugintest.Description("A"), 0)},
	} {
		t.Run(each.name, func(t *testing.T) {
			t.Parallel()

			got, err := plugin.DecodeDescription(each.data)

			if !errors.Is(err, plugin.ErrMalformed) {
				t.Fatalf("description = %+v, %v; want a refusal", got, err)
			}
			if len(got.Voices) != 0 || got.Name != "" {
				t.Errorf("a refused description still answered %+v", got)
			}
		})
	}
}

func TestMalformedTakesAreRefused(t *testing.T) {
	t.Parallel()

	for _, each := range []struct {
		name string
		data []byte
	}{
		{"nothing at all", nil},
		{"it promises a take it has no bytes for", counted(1)},
		{"a take has no parts", append(counted(1), counted(0)...)},
		{"a part count is negative", append(counted(1), counted(-2)...)},
		{"a path runs past the end", join(counted(1), counted(1), counted(40))},
		{"a path is not text", join(counted(1), counted(1), text([]byte{0xc3, 0x28}))},
		{"bytes follow the takes", append(plugintest.Takes([]string{"a.mp3"}), 0)},
	} {
		t.Run(each.name, func(t *testing.T) {
			t.Parallel()

			got, err := plugin.DecodeTakes(each.data)

			if !errors.Is(err, plugin.ErrMalformed) {
				t.Fatalf("takes = %v, %v; want a refusal", got, err)
			}
			if got != nil {
				t.Errorf("a refused answer still offered %v", got)
			}
		})
	}
}

// A voice whose readiness flag is any non-zero number is ready. The wire carries a flag
// rather than a boolean, so a plugin writing 2 means yes rather than something new.
func TestAnyNonZeroReadyFlagMeansReady(t *testing.T) {
	t.Parallel()

	w := &plugintest.Writer{}
	w.Count(1)
	w.Text("A")
	w.Text("id")
	w.Text("Name")
	w.Count(2)
	w.Text("")

	got, err := plugin.DecodeDescription(w.Bytes())

	if err != nil || len(got.Voices) != 1 || !got.Voices[0].Ready {
		t.Errorf("description = %+v, %v; want the voice read as ready", got, err)
	}
}

// counted writes one bare number, for building an answer that is deliberately wrong.
func counted(value int32) []byte {
	w := &plugintest.Writer{}
	w.Count(value)
	return w.Bytes()
}

// text writes one length-prefixed run of bytes, which need not be text at all.
func text(raw []byte) []byte {
	w := &plugintest.Writer{}
	w.Text(string(raw))
	return w.Bytes()
}

// join runs several written pieces together.
func join(pieces ...[]byte) []byte {
	var out []byte
	for _, piece := range pieces {
		out = append(out, piece...)
	}
	return out
}
