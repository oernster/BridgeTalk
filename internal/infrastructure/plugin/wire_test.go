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
		plugintest.Voice{ID: "one", Name: "The First Officer", Group: "Crew", Ready: true},
		plugintest.Voice{ID: "two", Name: "The Second", Reason: "its recordings are not on this machine"},
	)

	got, err := plugin.DecodeDescription(written)

	if err != nil {
		t.Fatalf("reading a description: %v", err)
	}
	// FR-582: the first voice names a group; the second names none, which reads as empty.
	want := plugin.Description{
		Name: "Bridge Crew",
		Voices: []plugin.VoiceInfo{
			{ID: "one", Name: "The First Officer", Group: "Crew", Ready: true},
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
		take.Of(`C:\audio\alone.mp3`),
		take.Of(`C:\audio\one.mp3`, `C:\audio\two.mp3`, `C:\audio\three.mp3`),
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

// FR-588: a part may be a span of a file, read back with its format, offset and length. The
// offset sits past what a 32 bit number reaches, so a reader narrowing it would be caught.
func TestASpanReadsBackWithItsPlaceInTheFile(t *testing.T) {
	t.Parallel()
	const beyondThirtyTwoBits = int64(1) << 33
	written := take.Take{
		take.SpanOf(`C:\audio\many.bin`, "mp3", beyondThirtyTwoBits, 4000),
		take.File(`C:\audio\tail.wav`),
	}

	got, err := plugin.DecodeTakes(plugintest.Takes(written))

	if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], written) {
		t.Errorf("takes = %v, %v; want %v", got, err, written)
	}
}

// FR-590: a span's numbers are not judged by the reader, since whether a span lies inside its file
// is a question about the file; a negative one reads back as sent, for the player to pass over.
func TestANegativeSpanReadsBackForThePlayerToJudge(t *testing.T) {
	t.Parallel()
	written := take.Take{take.SpanOf(`C:\audio\many.bin`, "mp3", -1, -1)}

	got, err := plugin.DecodeTakes(plugintest.Takes(written))

	if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], written) {
		t.Errorf("takes = %v, %v; want %v", got, err, written)
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
		{"a path runs past the end", join(counted(1), counted(1), counted(plugintest.PartFile), counted(40))},
		{"a path is not text", join(counted(1), counted(1), counted(plugintest.PartFile), text([]byte{0xc3, 0x28}))},
		{"a part is neither a file nor a span", join(counted(1), counted(1), counted(unknownKind), text([]byte("a.mp3")))},
		{"a span ends before its format", join(counted(1), counted(1), counted(plugintest.PartSpan), text([]byte("a.bin")))},
		{"a span ends before its length", join(counted(1), counted(1), counted(plugintest.PartSpan),
			text([]byte("a.bin")), text([]byte("mp3")), position(0))},
		{"a part runs short of its kind", join(counted(1), counted(1))},
		{"bytes follow the takes", append(plugintest.Takes(take.Of("a.mp3")), 0)},
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
	w.Text("")
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

// position writes one of a span's 64 bit numbers.
func position(value int64) []byte {
	w := &plugintest.Writer{}
	w.Position(value)
	return w.Bytes()
}

// unknownKind is a part kind the layout does not define.
const unknownKind = 7
