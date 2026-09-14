package main

// FR-551 and FR-552: the pauses tool makes each joined line with the model, digests its samples, asks
// the break finder once a voice then builds each voice's pauses in the order the script joins its
// lines; a line whose break is doubtful gets no pause.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/pause"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
	"github.com/oernster/bridge-talk/internal/domain/speech"
	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

const (
	// tokenScale keeps every sample the fake maker makes between -1 and 1.
	tokenScale = 1024
	// samplesPerToken is how many samples the fake maker makes for each number of a line.
	samplesPerToken = 8
	// styleFileBytes is the size of a style file: MaxSymbols rows of StyleWidth four-byte numbers.
	styleFileBytes = speech.MaxSymbols * speech.StyleWidth * 4
	// medianFinal is the final voiced stretch most fake answers give, in seconds.
	medianFinal = 0.48
)

// fakeMaker makes samples from a line's numbers, keeping what it made; the call numbered failAt fails.
type fakeMaker struct {
	made   [][]float32
	failAt int
	err    error
}

// Make answers samples built from the numbers, refusing a style row the model would not read.
func (m *fakeMaker) Make(_ context.Context, tokens []int64, style []float32) ([]float32, error) {
	if len(m.made)+1 == m.failAt {
		return nil, m.err
	}
	if len(style) != speech.StyleWidth {
		return nil, fmt.Errorf("a style row of %d numbers", len(style))
	}
	samples := make([]float32, len(tokens)*samplesPerToken)
	for index := range samples {
		samples[index] = float32(tokens[index%len(tokens)]) / tokenScale
	}
	m.made = append(m.made, samples)
	return samples, nil
}

// fakeFinder keeps every request it was handed and answers as the test says.
type fakeFinder struct {
	asked  []request
	answer func(asked request) ([]found, error)
}

// Find records the request and answers as the test says.
func (f *fakeFinder) Find(asked request) ([]found, error) {
	f.asked = append(f.asked, asked)
	return f.answer(asked)
}

// ref answers a pointer to its own copy of a value.
func ref[T any](value T) *T { return &value }

// sure answers every file with a cut at its place plus one and the median final.
func sure(asked request) ([]found, error) {
	answered := make([]found, 0, len(asked.Files))
	for at := range asked.Files {
		answered = append(answered, found{Cut: ref(at + 1), Final: ref(medianFinal)})
	}
	return answered, nil
}

// answering answers every request with the answers given.
func answering(given ...found) func(request) ([]found, error) {
	return func(request) ([]found, error) { return given, nil }
}

// joinedScript builds a script of three cues whose British and American sounds join six lines.
func joinedScript(t *testing.T) script.Voiced {
	t.Helper()
	saved := map[string]script.Saved{
		"Docked": {
			Lines:   []string{"Docking complete, commander.", "Down safely.", "Docked, commander."},
			British: []string{"bəkəmˈɑndə.", "bɪ", "bikəmˈɑndə."}, American: []string{"æəkəmˈændəɹ.", "æɪ", "ækəmˈændəɹ."},
		},
		"Jumped": {
			Lines:   []string{"Jumping, commander.", "Jumped, commander.", "Arrived."},
			British: []string{"dəkəmˈɑndə.", "dɪkəmˈɑndə.", "di"}, American: []string{"dækəmˈændəɹ.", "dɪkəmˈændəɹ.", "di"},
		},
		"Undocked": {
			Lines:   []string{"Clear, commander.", "Undocked.", "Away, commander."},
			British: []string{"ənkəmˈɑndə.", "ən", "ɪnkəmˈɑndə."}, American: []string{"ænkəmˈændəɹ.", "æn", "ɪnkəmˈændəɹ."},
		},
	}
	commander := map[string][]string{"commander": {"kəmˈɑndə", "kəmˈændəɹ"}}
	voiced, err := scripttest.BuildJoining(saved, commander, []string{"commander"})
	if err != nil {
		t.Fatalf("BuildJoining: %v", err)
	}
	return voiced
}

// voicesNamed parses voice ids.
func voicesNamed(t *testing.T, ids ...string) []machinevoice.Voice {
	t.Helper()
	voices := make([]machinevoice.Voice, 0, len(ids))
	for _, id := range ids {
		voice, err := machinevoice.Parse(id)
		if err != nil {
			t.Fatalf("Parse(%q): %v", id, err)
		}
		voices = append(voices, voice)
	}
	return voices
}

// voiceFolder lays out the model, ONNX Runtime and a style file for each voice, each file distinct.
func voiceFolder(t *testing.T, ids ...string) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, voicefiles.ModelFile), []byte("model"))
	write(t, filepath.Join(dir, voicefiles.RuntimeFile), []byte("runtime"))
	for at, voice := range voicesNamed(t, ids...) {
		style := make([]byte, styleFileBytes)
		style[0] = byte(at + 1)
		write(t, filepath.Join(dir, voicefiles.StyleFile(voice)), style)
	}
	return dir
}

// write writes body at path.
func write(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// digestOf is the SHA-256 of a file, written as hexadecimal.
func digestOf(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// runFinding finds the pauses of the voices named over the joined script, answering what it printed.
func runFinding(t *testing.T, made maker, finds finder, dir string, ids ...string) (pause.Book, string, error) {
	t.Helper()
	var out bytes.Buffer
	run := finding{files: voicefiles.New(dir), maker: made, finder: finds, work: t.TempDir(), out: &out}
	book, err := run.book(context.Background(), joinedScript(t), voicesNamed(t, ids...))
	return book, out.String(), err
}

// Each voice is asked about once, over its own accent's joined lines; its entries follow those lines
// in order with the cut answered for each, beside the digests of its style file and the model.
func TestEachVoicesEntriesFollowTheLinesTheScriptJoinsInOrder(t *testing.T) {
	ids := []string{"bf_emma", "am_michael"}
	dir := voiceFolder(t, ids...)
	finder := &fakeFinder{answer: sure}
	book, _, err := runFinding(t, &fakeMaker{}, finder, dir, ids...)
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	if len(finder.asked) != len(ids) {
		t.Fatalf("asked %d times, want once a voice", len(finder.asked))
	}
	for at, voice := range voicesNamed(t, ids...) {
		want := joinedScript(t).Joined(voice.Accent())
		got, ok := book.Voice(voice.ID())
		if !ok || len(got.Entries) != len(want) || len(finder.asked[at].Files) != len(want) {
			t.Fatalf("%s: %d entries from %d files, want %d", voice.ID(), len(got.Entries), len(finder.asked[at].Files), len(want))
		}
		for place, line := range want {
			entry := got.Entries[place]
			if entry.Cue != line.Cue || entry.Index != line.Index || entry.Sounds != line.Sounds || entry.Sample != place+1 || entry.Doubtful {
				t.Errorf("%s entry %d = %+v, want %+v at sample %d", voice.ID(), place, entry, line, place+1)
			}
		}
		if styleFile := filepath.Join(dir, voicefiles.StyleFile(voice)); got.Style != digestOf(t, styleFile) {
			t.Errorf("%s style digest = %q, want its style file's", voice.ID(), got.Style)
		}
	}
	if book.Model() != digestOf(t, filepath.Join(dir, voicefiles.ModelFile)) {
		t.Errorf("model digest = %q, want the model file's", book.Model())
	}
	if book.Silence() != samplesIn(silence) {
		t.Errorf("silence = %d, want %d", book.Silence(), samplesIn(silence))
	}
}

// doubtfulOf answers which of a voice's entries are doubtful, in order.
func doubtfulOf(t *testing.T, book pause.Book, id string) []bool {
	t.Helper()
	voice, _ := book.Voice(id)
	doubtful := make([]bool, 0, len(voice.Entries))
	for _, entry := range voice.Entries {
		doubtful = append(doubtful, entry.Doubtful)
	}
	return doubtful
}

// FR-552: a line answered with no cut, with no final or with a final far from the voice's median is
// doubtful; the rest keep the cut answered.
func TestALineWithNoCutNoFinalOrADoubtfulFinalGetsNoPause(t *testing.T) {
	finder := &fakeFinder{answer: answering(
		found{Cut: ref(10), Final: ref(medianFinal)}, found{Final: ref(medianFinal)}, found{Cut: ref(10)},
		found{Cut: ref(10), Final: ref(0.29)}, found{Cut: ref(10), Final: ref(0.52)}, found{Cut: ref(10), Final: ref(medianFinal)},
	)}
	book, _, err := runFinding(t, &fakeMaker{}, finder, voiceFolder(t, "bf_emma"), "bf_emma")
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	if got, want := doubtfulOf(t, book, "bf_emma"), []bool{false, true, true, true, false, false}; !slices.Equal(got, want) {
		t.Errorf("doubtful = %v, want %v", got, want)
	}
}

// FR-552: a line with no break has no final voiced stretch, so it takes no part in its voice's median.
func TestALineWithNoBreakIsLeftOutOfItsVoicesMedian(t *testing.T) {
	finder := &fakeFinder{answer: answering(
		found{Cut: ref(10), Final: ref(0.4)}, found{Cut: ref(10), Final: ref(medianFinal)}, found{Cut: ref(10), Final: ref(0.5)},
		found{}, found{}, found{},
	)}
	book, _, err := runFinding(t, &fakeMaker{}, finder, voiceFolder(t, "bf_emma"), "bf_emma")
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	if got, want := doubtfulOf(t, book, "bf_emma"), []bool{false, false, false, true, true, true}; !slices.Equal(got, want) {
		t.Errorf("doubtful = %v, want %v", got, want)
	}
}

// FR-551: each entry records the digest of the samples the model made for its line; the finder reads
// those same samples at 16 bits.
func TestTheDigestRecordedIsTheSamplesOwnAndTheFinderReadsThemAtSixteenBits(t *testing.T) {
	maker := &fakeMaker{}
	var read [][]int16
	finder := &fakeFinder{answer: func(asked request) ([]found, error) {
		for _, path := range asked.Files {
			read = append(read, wavSamples(t, path))
		}
		return sure(asked)
	}}
	book, _, err := runFinding(t, maker, finder, voiceFolder(t, "bf_emma"), "bf_emma")
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	voice, _ := book.Voice("bf_emma")
	if len(voice.Entries) == 0 || len(read) != len(voice.Entries) || len(maker.made) != len(voice.Entries) {
		t.Fatalf("%d entries from %d files read and %d lines made, want one of each a line", len(voice.Entries), len(read), len(maker.made))
	}
	for at, entry := range voice.Entries {
		if want := pause.Digest(maker.made[at]); entry.Digest != want {
			t.Errorf("entry %d digest = %q, want %q", at, entry.Digest, want)
		}
		if want := asInt16(madelines.SixteenBits(maker.made[at])); !slices.Equal(read[at], want) {
			t.Errorf("file %d holds %v, want %v", at, read[at], want)
		}
	}
}

// asInt16 narrows 16-bit samples held as int32.
func asInt16(samples []int32) []int16 {
	narrowed := make([]int16, 0, len(samples))
	for _, sample := range samples {
		narrowed = append(narrowed, int16(sample))
	}
	return narrowed
}

// FR-551 and FR-552: each voice is printed with how many of its lines are doubtful, listing each by
// voice, cue and line, then the total over every voice.
func TestEachVoiceIsListedWithItsDoubtfulLinesThenTheTotal(t *testing.T) {
	finder := &fakeFinder{answer: answering(
		found{Cut: ref(10), Final: ref(medianFinal)}, found{Final: ref(medianFinal)}, found{Cut: ref(10)},
		found{Cut: ref(10), Final: ref(0.29)}, found{Cut: ref(10), Final: ref(0.52)}, found{Cut: ref(10), Final: ref(medianFinal)},
	)}
	_, printed, err := runFinding(t, &fakeMaker{}, finder, voiceFolder(t, "bf_emma", "bm_george"), "bf_emma", "bm_george")
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	for _, want := range []string{
		"bf_emma: 3 of 6 lines doubtful", `bf_emma "Docked" line 3`, `bf_emma "Jumped" line 1`, `bf_emma "Jumped" line 2`,
		"bm_george: 3 of 6 lines doubtful", "2 voices: 6 of 12 lines doubtful",
	} {
		if !strings.Contains(printed, want) {
			t.Errorf("printed %q, want it to hold %q", printed, want)
		}
	}
	if strings.Contains(printed, `bf_emma "Docked" line 1`) {
		t.Errorf("printed %q, listing a line that has its pause", printed)
	}
}

// The lines written for the finder are removed once it answers, leaving the working folder empty.
func TestTheLinesWrittenForTheFinderAreRemovedOnceItAnswers(t *testing.T) {
	var handed []string
	finder := &fakeFinder{answer: func(asked request) ([]found, error) {
		handed = append(handed, asked.Files...)
		return sure(asked)
	}}
	work := t.TempDir()
	run := finding{files: voicefiles.New(voiceFolder(t, "bf_emma")), maker: &fakeMaker{}, finder: finder, work: work, out: &bytes.Buffer{}}
	if _, err := run.book(context.Background(), joinedScript(t), voicesNamed(t, "bf_emma")); err != nil {
		t.Fatalf("book: %v", err)
	}
	if len(handed) == 0 {
		t.Fatal("the finder was handed no files")
	}
	if left, _ := os.ReadDir(work); len(left) != 0 {
		t.Errorf("the working folder still holds %d entries", len(left))
	}
}
