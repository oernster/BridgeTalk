package audio

// FR-526's note: a made line plays without the FLAC library writing to the output. The library logs
// through the standard logger whenever a frame names a sample rate its own tests lack; 24 kHz, the
// rate every made line is at, is one of them.

import (
	"bytes"
	"log"
	"testing"

	"github.com/gopxl/beep/v2"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/take"
	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
)

// streamBuffer is how many frames each read of the decoded line asks for.
const streamBuffer = 512

// Not parallel: it takes over the standard logger to read what decoding writes to it, which Go
// lets it do before any parallel test resumes.
func TestAMadeLinePlaysWithoutTheFlacLibraryLogging(t *testing.T) {
	voice, err := machinevoice.Parse("bf_emma")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	store := madelines.New(t.TempDir())
	samples := make([]float32, madelines.SampleRate)
	for index := range samples {
		samples[index] = float32(index%streamBuffer) / streamBuffer
	}
	if err := store.Write(voice, "line", samples); err != nil {
		t.Fatalf("Write: %v", err)
	}

	var logged bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logged)
	defer log.SetOutput(previous)

	streamer, format, closer, err := decode(take.File(store.Path(voice, "line")))
	if err != nil {
		t.Fatalf("decoding: %v", err)
	}
	defer func() { _ = closer() }()
	defer func() { _ = streamer.Close() }()
	read := 0
	buffer := make([][2]float64, streamBuffer)
	for {
		count, more := streamer.Stream(buffer)
		read += count
		if !more {
			break
		}
	}

	if logged.Len() != 0 {
		t.Errorf("decoding a made line logged %q", logged.String())
	}
	if format.SampleRate != beep.SampleRate(madelines.SampleRate) || read != len(samples) {
		t.Errorf("decoded %d frames at %d Hz, want %d at %d Hz", read, format.SampleRate, len(samples), madelines.SampleRate)
	}
}
