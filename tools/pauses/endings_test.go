package main

// FR-555: the pauses tool makes each line ending on a nasal with the model, digests its samples, asks
// the ending finder once a voice then builds each voice's endings in the order the script lists the
// lines; a line answered with no start has no fade.

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/domain/pause"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

// fakeEndingFinder keeps every request it was handed and answers as the test says.
type fakeEndingFinder struct {
	asked  []endingRequest
	answer func(asked endingRequest) ([]ended, error)
}

// FindEndings records the request and answers as the test says.
func (f *fakeEndingFinder) FindEndings(asked endingRequest) ([]ended, error) {
	f.asked = append(f.asked, asked)
	return f.answer(asked)
}

// startingAt answers every file with a start at its place plus one.
func startingAt(asked endingRequest) ([]ended, error) {
	answered := make([]ended, 0, len(asked.Files))
	for at := range asked.Files {
		answered = append(answered, ended{Start: ref(at + 1)})
	}
	return answered, nil
}

// endingWith answers every request with the answers given.
func endingWith(given ...ended) func(endingRequest) ([]ended, error) {
	return func(endingRequest) ([]ended, error) { return given, nil }
}

// nasalScript builds a script of two cues whose first and third lines end on a nasal in each accent.
func nasalScript(t *testing.T) script.Voiced {
	t.Helper()
	voiced, err := scripttest.Build(map[string]script.Saved{
		"Docked": {
			Lines:   []string{"Down.", "Docked.", "Moving on."},
			British: []string{"dˈWn.", "dˈɒkt.", "mˈuːvɪŋ ˈɒn."}, American: []string{"dˈWn.", "dˈɑkt.", "mˈuvɪŋ ˈɔn."},
		},
		"Jumped": {
			Lines:   []string{"Jumping.", "Arrived.", "Gone."},
			British: []string{"ʤˈʌmpɪŋ.", "əɹˈIvd.", "ɡˈɒn."}, American: []string{"ʤˈʌmpɪŋ.", "əɹˈIvd.", "ɡˈɔn."},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return voiced
}

// runEndings finds the endings of the voices named over the nasal script, answering what it printed.
func runEndings(t *testing.T, made maker, ends endingFinder, dir string, ids ...string) (ending.Book, string, error) {
	t.Helper()
	var out bytes.Buffer
	run := finding{files: voicefiles.New(dir), maker: made, ends: ends, work: t.TempDir(), out: &out}
	book, err := run.endings(context.Background(), nasalScript(t), voicesNamed(t, ids...))
	return book, out.String(), err
}

// FR-555: the fade is 10 ms at the model's rate, 240 samples at 24 kHz, as Oliver chose by ear.
func TestTheFadeIsTenMillisecondsOfSamples(t *testing.T) {
	if got, want := samplesIn(fadeLength), madelines.SampleRate*10/1000; got != want || want != 240 {
		t.Errorf("fade = %d samples, want %d", got, want)
	}
}

// FR-555: the ending finder is handed every setting with the files: 10 ms frames, 3 kHz, -50 dB, a
// share of 0.4 and 50 ms before the end of sound, each frame and span in samples.
func TestTheEndingFinderIsAskedWithEverySetting(t *testing.T) {
	files := []string{"a.wav", "b.wav"}
	want := endingRequest{Frame: 240, HighHz: 3000, LoudDB: -50, Share: 0.4, Within: 1200, Files: files}
	if got := endingRequestFor(files); !reflect.DeepEqual(got, want) {
		t.Errorf("request = %+v, want %+v", got, want)
	}
}

// Each voice is asked about once, over its own accent's lines ending on a nasal; its entries follow
// those lines in order with the start answered for each and the digest of the samples made, beside the
// digests of its style file and the model and the fade.
func TestEachVoicesEndingsFollowItsLinesEndingOnANasalInOrder(t *testing.T) {
	ids := []string{"bf_emma", "am_michael"}
	dir := voiceFolder(t, ids...)
	maker, finder := &fakeMaker{}, &fakeEndingFinder{answer: startingAt}
	book, _, err := runEndings(t, maker, finder, dir, ids...)
	if err != nil {
		t.Fatalf("endings: %v", err)
	}
	if len(finder.asked) != len(ids) {
		t.Fatalf("asked %d times, want once a voice", len(finder.asked))
	}
	made := 0
	for at, voice := range voicesNamed(t, ids...) {
		want := nasalScript(t).EndingOnNasal(voice.Accent())
		got, ok := book.Voice(voice.ID())
		if !ok || len(want) == 0 || len(got.Entries) != len(want) || len(finder.asked[at].Files) != len(want) {
			t.Fatalf("%s: %d entries from %d files, want %d", voice.ID(), len(got.Entries), len(finder.asked[at].Files), len(want))
		}
		for place, line := range want {
			entry := got.Entries[place]
			if entry.Cue != line.Cue || entry.Index != line.Index || entry.Sounds != line.Sounds || entry.Sample != place+1 ||
				entry.Digest != pause.Digest(maker.made[made]) {
				t.Errorf("%s entry %d = %+v, want %+v at sample %d with its samples' digest", voice.ID(), place, entry, line, place+1)
			}
			made++
		}
		if styleFile := filepath.Join(dir, voicefiles.StyleFile(voice)); got.Style != digestOf(t, styleFile) {
			t.Errorf("%s style digest = %q, want its style file's", voice.ID(), got.Style)
		}
	}
	if book.Model() != digestOf(t, filepath.Join(dir, voicefiles.ModelFile)) || book.Fade() != samplesIn(fadeLength) {
		t.Errorf("model digest %q fade %d, want the model file's and %d", book.Model(), book.Fade(), samplesIn(fadeLength))
	}
}

// FR-555: a line answered with no start ends on no burst, so it is given no fade.
func TestALineAnsweredWithNoStartHasNoFade(t *testing.T) {
	finder := &fakeEndingFinder{answer: endingWith(ended{Start: ref(5)}, ended{}, ended{Start: ref(7)}, ended{})}
	book, _, err := runEndings(t, &fakeMaker{}, finder, voiceFolder(t, "bf_emma"), "bf_emma")
	if err != nil {
		t.Fatalf("endings: %v", err)
	}
	voice, _ := book.Voice("bf_emma")
	var samples []int
	for _, entry := range voice.Entries {
		samples = append(samples, entry.Sample)
	}
	if want := []int{5, 0, 7, 0}; !slices.Equal(samples, want) {
		t.Errorf("samples = %v, want %v", samples, want)
	}
}

// A start at the first sample or at or beyond the end of its line is refused, naming the line; so is a
// finder answering a different number of lines than it was handed.
func TestAnEndingFinderAnswerThatCannotBeMatchedToItsLineIsRefused(t *testing.T) {
	for name, each := range map[string]struct {
		answers []ended
		want    error
	}{
		"start at zero":      {answers: []ended{{Start: ref(0)}, {}, {}, {}}, want: errCutOutside},
		"start past the end": {answers: []ended{{Start: ref(1 << 20)}, {}, {}, {}}, want: errCutOutside},
		"too few answers":    {answers: []ended{{}, {}}, want: errAnswerCount},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := runEndings(t, &fakeMaker{}, &fakeEndingFinder{answer: endingWith(each.answers...)}, voiceFolder(t, "bf_emma"), "bf_emma")
			if !errors.Is(err, each.want) {
				t.Errorf("got %v, want %v", err, each.want)
			}
			if each.want == errCutOutside && (err == nil || !strings.Contains(err.Error(), `bf_emma "Docked" line 1`)) {
				t.Errorf("%v does not name the line", err)
			}
		})
	}
}

// FR-555: each voice is printed with each line given a fade, by voice, cue and line, then how many of
// its lines were given one; then the total over every voice.
func TestEachVoiceIsListedWithItsFadedLinesThenTheTotal(t *testing.T) {
	finder := &fakeEndingFinder{answer: endingWith(ended{Start: ref(5)}, ended{}, ended{Start: ref(7)}, ended{})}
	_, printed, err := runEndings(t, &fakeMaker{}, finder, voiceFolder(t, "bf_emma", "bm_george"), "bf_emma", "bm_george")
	if err != nil {
		t.Fatalf("endings: %v", err)
	}
	for _, want := range []string{
		`bf_emma "Docked" line 1`, `bf_emma "Jumped" line 1`, "bf_emma: 2 of 4 lines faded",
		"bm_george: 2 of 4 lines faded", "2 voices: 4 of 8 lines faded",
	} {
		if !strings.Contains(printed, want) {
			t.Errorf("printed %q, want it to hold %q", printed, want)
		}
	}
	if strings.Contains(printed, `bf_emma "Docked" line 3`) {
		t.Errorf("printed %q, listing a line with no fade", printed)
	}
}
