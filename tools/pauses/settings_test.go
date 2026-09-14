package main

// FR-551 and FR-553: every setting has its one home in the tool, handed to the finder in each request;
// the silence is 40 ms of samples; each line is written for the finder as a 16-bit mono WAV.

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
)

// FR-553: the silence a pause inserts is 40 ms at the model's rate, 960 samples at 24 kHz.
func TestTheSilenceIsFortyMillisecondsOfSamples(t *testing.T) {
	if got, want := samplesIn(silence), madelines.SampleRate*40/1000; got != want || want != 960 {
		t.Errorf("silence = %d samples, want %d", got, want)
	}
}

// FR-551: the finder is handed every setting with the files: 10 ms frames, gaps of up to two frames
// bridged, a final of at least 20 frames, a 10 ms quiet window and the voice's pitch range by sex.
func TestTheFinderIsAskedWithEverySettingAndTheVoicesPitchRange(t *testing.T) {
	files := []string{"a.wav", "b.wav"}
	for id, want := range map[string]request{
		"bf_emma":   {TimeStep: 0.01, Frame: 240, PitchFloor: 120, PitchCeiling: 350, Bridged: 2, Final: 20, Quiet: 240, Files: files},
		"bm_george": {TimeStep: 0.01, Frame: 240, PitchFloor: 65, PitchCeiling: 200, Bridged: 2, Final: 20, Quiet: 240, Files: files},
	} {
		if got := requestFor(voicesNamed(t, id)[0], files); !reflect.DeepEqual(got, want) {
			t.Errorf("%s request = %+v, want %+v", id, got, want)
		}
	}
}

// wavHeaderBytes is the length of a plain PCM WAV header.
const wavHeaderBytes = 44

// wavSamples reads a written line back, holding it to a 16-bit mono PCM WAV at the model's rate.
func wavSamples(t *testing.T, path string) []int16 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if len(raw) < wavHeaderBytes || string(raw[0:4]) != "RIFF" || string(raw[8:16]) != "WAVEfmt " || string(raw[36:40]) != "data" {
		t.Fatalf("%s is not a plain WAV: % x", path, raw[:min(len(raw), wavHeaderBytes)])
	}
	var header struct {
		Riff       [4]byte
		Size       uint32
		Wave       [8]byte
		FormatSize uint32
		Format     uint16
		Channels   uint16
		Rate       uint32
		ByteRate   uint32
		BlockAlign uint16
		Bits       uint16
		Data       [4]byte
		DataSize   uint32
	}
	if err := binary.Read(bytes.NewReader(raw), binary.LittleEndian, &header); err != nil {
		t.Fatalf("reading the header of %s: %v", path, err)
	}
	dataBytes := len(raw) - wavHeaderBytes
	if header.Size != uint32(len(raw)-8) || header.FormatSize != 16 || header.Format != 1 || header.Channels != 1 ||
		header.Rate != madelines.SampleRate || header.ByteRate != madelines.SampleRate*2 || header.BlockAlign != 2 ||
		header.Bits != madelines.BitsPerSample || header.DataSize != uint32(dataBytes) {
		t.Fatalf("%s header = %+v, want mono 16-bit PCM at %d Hz holding %d bytes", path, header, madelines.SampleRate, dataBytes)
	}
	samples := make([]int16, dataBytes/2)
	if err := binary.Read(bytes.NewReader(raw[wavHeaderBytes:]), binary.LittleEndian, samples); err != nil {
		t.Fatalf("reading the samples of %s: %v", path, err)
	}
	return samples
}

// A line is written as a 16-bit mono WAV at the model's rate, its samples clamped, scaled and rounded.
func TestALineIsWrittenAsSixteenBitMonoWAVAtTheModelsRate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "line.wav")
	if err := writeWAV(path, []float32{-1.5, -0.5, 0, 0.5, 1.5}); err != nil {
		t.Fatalf("writeWAV: %v", err)
	}
	if got, want := wavSamples(t, path), []int16{-32767, -16384, 0, 16384, 32767}; !slices.Equal(got, want) {
		t.Errorf("samples = %v, want %v", got, want)
	}
}

// A line that cannot be written is refused, naming where it was to go.
func TestALineThatCannotBeWrittenIsRefusedNamingItsPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "line.wav")
	if err := writeWAV(path, []float32{0}); err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("got %v, want a failure naming %s", err, path)
	}
}
