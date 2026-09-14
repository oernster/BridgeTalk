package madelines_test

// FR-517, FR-523, FR-526, FR-527 and FR-530: the made lines kept for machine voices, read back
// through the FLAC library the player decodes with.

import (
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

func voiceNamed(t *testing.T, id string) machinevoice.Voice {
	t.Helper()
	voice, err := machinevoice.Parse(id)
	if err != nil {
		t.Fatalf("Parse(%q): %v", id, err)
	}
	return voice
}

// written stores a made line the test expects to be written.
func written(t *testing.T, store madelines.Store, voice machinevoice.Voice, key string, samples []float32) {
	t.Helper()
	if err := store.Write(voice, key, samples); err != nil {
		t.Fatalf("Write(%s, %s): %v", voice.ID(), key, err)
	}
}

// decoded reads a made line back, answering its samples and what its stream says of itself.
//
// The file is opened here and handed to flac.New rather than opened by flac.Open: that wraps the
// file in a buffered reader, so the stream's Close never reaches the file, measured by a test folder
// that could not be deleted while the line stayed open.
func decoded(t *testing.T, path string) ([]int32, *meta.StreamInfo) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	defer file.Close()
	stream, err := flac.New(file)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var samples []int32
	for {
		frame, err := stream.ParseNext()
		if errors.Is(err, io.EOF) {
			return samples, stream.Info
		}
		if err != nil {
			t.Fatalf("decoding %s: %v", path, err)
		}
		samples = append(samples, frame.Subframes[0].Samples...)
	}
}

// sixteen is FR-526's rule for one sample: clamped to between -1 and 1, scaled to 16 bits, rounded.
func sixteen(sample float32) int32 {
	return int32(math.Round(math.Max(-1, math.Min(1, float64(sample))) * math.MaxInt16))
}

// FR-526's acceptance: a made line is mono 16-bit FLAC at the model's rate, decoding to its samples
// clamped, scaled and rounded.
func TestAMadeLineDecodesToItsSamplesRoundedToSixteenBits(t *testing.T) {
	t.Parallel()
	store := madelines.New(t.TempDir())
	emma := voiceNamed(t, "bf_emma")

	written(t, store, emma, "line", []float32{-1.5, -0.5, 0, 0.5, 1.5})

	samples, info := decoded(t, store.Path(emma, "line"))
	if want := []int32{-32767, -16384, 0, 16384, 32767}; !slices.Equal(samples, want) {
		t.Errorf("decoded %v, want %v", samples, want)
	}
	if info.SampleRate != madelines.SampleRate || info.NChannels != 1 || info.BitsPerSample != madelines.BitsPerSample || info.NSamples != 5 {
		t.Errorf("stream info = %+v, want %d Hz mono %d-bit holding 5 samples", info, madelines.SampleRate, madelines.BitsPerSample)
	}
}

// FR-526: a line longer than a block comes back sample for sample whatever its blocks hold: silence,
// the widest swing a sample can make, a changing wave and a last block too short to predict over.
func TestALongLineComesBackSampleForSample(t *testing.T) {
	t.Parallel()
	store := madelines.New(t.TempDir())
	emma := voiceNamed(t, "bf_emma")
	var samples []float32
	samples = append(samples, make([]float32, madelines.BlockSize)...)
	for index := range madelines.BlockSize {
		samples = append(samples, float32(1-2*(index%2)))
	}
	for index := range madelines.BlockSize {
		samples = append(samples, float32(0.8*math.Sin(float64(index)/20)))
	}
	for index := range madelines.ShortestPredicted - 1 {
		samples = append(samples, float32(index)/10)
	}

	written(t, store, emma, "long", samples)

	got, _ := decoded(t, store.Path(emma, "long"))
	want := make([]int32, 0, len(samples))
	for _, sample := range samples {
		want = append(want, sixteen(sample))
	}
	if !slices.Equal(got, want) {
		t.Errorf("decoded %d samples differing from the %d written", len(got), len(want))
	}
}

// Keys lists a voice's made lines alone. Another voice's are not its; nor is a file left by an
// interrupted write (FR-517) or anything else in its folder.
func TestKeysListAVoicesMadeLinesAlone(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := madelines.New(dir)
	emma, michael, alice := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael"), voiceNamed(t, "bf_alice")
	written(t, store, emma, "b", []float32{0})
	written(t, store, emma, "a", []float32{0})
	written(t, store, michael, "c", []float32{0})
	if err := os.WriteFile(store.Path(emma, "d")+madelines.PartSuffix, []byte("half"), 0o644); err != nil {
		t.Fatalf("leaving a part: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, emma.ID(), "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("leaving notes: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, emma.ID(), "e.flac"), 0o755); err != nil {
		t.Fatalf("making a folder named like a line: %v", err)
	}

	if got := store.Keys(emma); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("bf_emma's keys = %v, want a and b", got)
	}
	if got := store.Keys(michael); !slices.Equal(got, []string{"c"}) {
		t.Errorf("am_michael's keys = %v, want c", got)
	}
	if got := store.Keys(alice); len(got) != 0 {
		t.Errorf("bf_alice's keys = %v, want none", got)
	}
}

// FR-517: a line is written beside its place, then moved into it, so an interrupted write leaves no
// made line. Writing over a part left behind and over an earlier made line both succeed.
func TestALineIsWrittenWholeOrNotAtAll(t *testing.T) {
	t.Parallel()
	store := madelines.New(t.TempDir())
	emma := voiceNamed(t, "bf_emma")
	written(t, store, emma, "first", []float32{0})
	part := store.Path(emma, "line") + madelines.PartSuffix
	if err := os.WriteFile(part, []byte("half a line"), 0o644); err != nil {
		t.Fatalf("leaving a part: %v", err)
	}

	written(t, store, emma, "line", []float32{0.5})
	written(t, store, emma, "line", []float32{-0.5})

	if got, _ := decoded(t, store.Path(emma, "line")); !slices.Equal(got, []int32{sixteen(-0.5)}) {
		t.Errorf("decoded %v, want the second line written", got)
	}
	if _, err := os.Stat(part); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the part is still there: %v", err)
	}
}

// FR-520 and FR-237: a line that cannot be written is refused naming the path once, leaving no part.
func TestALineThatCannotBeWrittenIsRefusedNamingItOnce(t *testing.T) {
	t.Parallel()
	emma := voiceNamed(t, "bf_emma")

	plain := filepath.Join(t.TempDir(), "plain")
	if err := os.WriteFile(plain, []byte("x"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", plain, err)
	}
	blocked := madelines.New(plain)
	refusedOnce(t, "a folder under a file", blocked.Write(emma, "line", []float32{0}), filepath.Join(plain, emma.ID()))

	partFolder := madelines.New(t.TempDir())
	part := partFolder.Path(emma, "line") + madelines.PartSuffix
	if err := os.MkdirAll(part, 0o755); err != nil {
		t.Fatalf("making %s: %v", part, err)
	}
	refusedOnce(t, "a folder where the part goes", partFolder.Write(emma, "line", []float32{0}), part)

	lineFolder := madelines.New(t.TempDir())
	line := lineFolder.Path(emma, "line")
	if err := os.MkdirAll(filepath.Join(line, "inside"), 0o755); err != nil {
		t.Fatalf("making %s: %v", line, err)
	}
	refusedOnce(t, "a folder where the line goes", lineFolder.Write(emma, "line", []float32{0}), line)
	if _, err := os.Stat(line + madelines.PartSuffix); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused write left its part: %v", err)
	}
}

// FR-527: DeleteAllBut deletes every other voice's made lines, keeping the voice named; DeleteAll
// deletes every one. Where nothing was ever made there is nothing to refuse.
// FR-527: deleting removes the voice's lines under the keys given and nothing else; a key with no
// line is no failure.
func TestDeletingRemovesOnlyTheKeysGiven(t *testing.T) {
	t.Parallel()
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	store := madelines.New(t.TempDir())
	written(t, store, emma, "a", []float32{0})
	written(t, store, emma, "b", []float32{0})
	written(t, store, michael, "a", []float32{0})

	if err := store.Delete(emma, []string{"a", "never made"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if !slices.Equal(store.Keys(emma), []string{"b"}) || !slices.Equal(store.Keys(michael), []string{"a"}) {
		t.Errorf("after deleting bf_emma's a: bf_emma %v, am_michael %v", store.Keys(emma), store.Keys(michael))
	}
}

// FR-530 and FR-237: a made line that cannot be deleted is refused naming it once; the other keys are
// still deleted.
func TestLinesThatCannotBeDeletedAreRefusedNamingThemOnce(t *testing.T) {
	t.Parallel()
	store := madelines.New(t.TempDir())
	emma := voiceNamed(t, "bf_emma")
	written(t, store, emma, "held", []float32{0})
	written(t, store, emma, "free", []float32{0})
	held, err := os.Open(store.Path(emma, "held"))
	if err != nil {
		t.Fatalf("holding a line open: %v", err)
	}
	defer held.Close()

	refusedOnce(t, "a line held open", store.Delete(emma, []string{"held", "free"}), store.Path(emma, "held"))

	if !slices.Equal(store.Keys(emma), []string{"held"}) {
		t.Errorf("bf_emma's lines %v; want free deleted past the refusal", store.Keys(emma))
	}
}

// FR-523: made lines live in the product's own data folder, which a library root never is.
func TestMadeLinesLiveInTheProductsDataFolder(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	t.Setenv("XDG_DATA_HOME", base)

	got, err := madelines.Dir()

	if want := filepath.Join(base, product.Slug, madelines.Folder); err != nil || got != want {
		t.Errorf("Dir() = %q, %v; want %q", got, err, want)
	}
	for _, key := range []string{"LOCALAPPDATA", "XDG_DATA_HOME", "HOME", "USERPROFILE"} {
		t.Setenv(key, "")
	}
	if _, err := madelines.Dir(); err == nil {
		t.Error("a folder was invented with nowhere to put it")
	}
}

// refusedOnce fails the test for each way a refusal over path breaks FR-237.
func refusedOnce(t *testing.T, what string, refused error, path string) {
	t.Helper()
	for _, problem := range refusal.Check(refused, path) {
		t.Errorf("%s: %s", what, problem)
	}
}
