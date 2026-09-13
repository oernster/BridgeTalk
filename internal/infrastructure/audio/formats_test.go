package audio

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

// FR-203: the four formats the player decodes are recognised, whatever case the extension
// is written in; nothing else is.
func TestTheFormatsThePlayerDecodesAreRecognisedInAnyCase(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"a.mp3", "a.WAV", "a.Flac", "a.ogg"} {
		if !Recognised(name) {
			t.Errorf("%s was not recognised", name)
		}
	}
	for _, name := range []string{"a.txt", "a.jpg", "a", "mp3", "a.mp3.bak"} {
		if Recognised(name) {
			t.Errorf("%s was recognised", name)
		}
	}
}

// FR-204: a recording that plays is called playable, in each format a take can be made in.
// So is one cut short after its first samples: only the start is decoded, which is the
// limit Playable states; the decoder reads a short file to its end without complaint.
func TestARecordingThatPlaysIsPlayable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, name := range []string{"take.mp3", "take.wav"} {
		path := filepath.Join(dir, name)
		audiotest.WriteTake(t, path)
		if err := Playable(path); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	header := len(audiotest.WAV(t, 0))
	blockAlign := len(audiotest.WAV(t, 1)) - header
	cutShort := filepath.Join(dir, "cutShort.wav")
	audiotest.WriteFile(t, cutShort, audiotest.WAV(t, 64)[:header+blockAlign])
	if err := Playable(cutShort); err != nil {
		t.Errorf("a take cut short after its first samples: %v", err)
	}
}

// FR-204: a file named as a recording that will not play says why, in every way it can fail,
// and names no file: the scan names the take where it was found.
func TestARecordingThatWillNotPlaySaysWhyWithoutNamingItself(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	header := len(audiotest.WAV(t, 0))
	cases := []struct {
		name string
		body []byte
		want error
	}{
		{"broken.mp3", audiotest.NotARecording, nil},
		{"broken.wav", audiotest.NotARecording, nil},
		{"broken.flac", audiotest.NotARecording, nil},
		{"broken.ogg", audiotest.NotARecording, nil},
		{"headerOnly.wav", audiotest.WAV(t, 64)[:header], nil},
		{"silent.wav", audiotest.WAV(t, 0), ErrNoSound},
		{"notes.txt", audiotest.NotARecording, ErrUnsupportedFormat},
	}
	for _, each := range cases {
		path := filepath.Join(dir, each.name)
		audiotest.WriteFile(t, path, each.body)
		err := Playable(path)
		if err == nil {
			t.Errorf("%s was called playable", each.name)
			continue
		}
		if each.want != nil && !errors.Is(err, each.want) {
			t.Errorf("%s: got %v, want %v", each.name, err, each.want)
		}
		if strings.Contains(err.Error(), dir) {
			t.Errorf("%s: %q names the file", each.name, err)
		}
	}

	absent := filepath.Join(dir, "absent.mp3")
	if err := Playable(absent); !errors.Is(err, fs.ErrNotExist) || strings.Contains(err.Error(), dir) {
		t.Errorf("a take that is not there: got %v, want not-exist without the path", err)
	}
}
