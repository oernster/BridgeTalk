// Package audiotest lays out recordings for tests: the smallest file in each format it
// can make that the player's decoders actually read as sound.
//
// The scan decodes the start of every take before it offers one (FR-204), so a test that
// wrote a single byte where a take belongs would find nothing there. Every package that lays
// out a recordings tree writes it through here, so what a take that plays looks like is
// known in one place.
package audiotest

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// NotARecording is the body of a file that is no recording: a text file, a picture, a file
// standing where a folder belongs or a take that will not play.
var NotARecording = []byte("not a recording")

// The shape every recording made here shares. An MPEG-1 Layer III frame holds a fixed
// count of samples, so the WAV take is made the same length as the MP3 one.
const (
	sampleRate      = 44100
	channels        = 2
	bitsPerSample   = 16
	bitsPerByte     = 8
	samplesPerFrame = 1152
	bitRate         = 128000
)

// frameHeader opens an MPEG-1 Layer III frame: the sync word, no checksum, 128 kbit/s at
// 44.1 kHz, no padding, stereo.
var frameHeader = []byte{0xFF, 0xFB, 0x90, 0x00}

// MP3 returns one frame of MPEG-1 Layer III silence: the header, then zeros.
//
// A frame whose side information is all zeros carries no audio data, which the decoder reads
// as silence rather than refusing. Measured on 2026-09-13: it decodes to 1152 samples.
func MP3() []byte {
	frame := make([]byte, samplesPerFrame*bitRate/sampleRate/bitsPerByte)
	copy(frame, frameHeader)
	return frame
}

// WAV builds the smallest real WAV file holding a given count of frames: a header the
// decoder accepts and the samples behind it. Zero frames makes the header alone.
func WAV(t testing.TB, frames int) []byte {
	t.Helper()
	blockAlign := channels * bitsPerSample / bitsPerByte
	dataSize := frames * blockAlign

	var out bytes.Buffer
	write := func(values ...any) {
		for _, value := range values {
			if err := binary.Write(&out, binary.LittleEndian, value); err != nil {
				t.Fatalf("building the wav: %v", err)
			}
		}
	}
	const (
		formatChunkSize = 16
		pcm             = 1
		headerAfterRiff = 36
	)
	out.WriteString("RIFF")
	write(uint32(headerAfterRiff + dataSize))
	out.WriteString("WAVE")
	out.WriteString("fmt ")
	write(uint32(formatChunkSize), uint16(pcm), uint16(channels), uint32(sampleRate))
	write(uint32(sampleRate*blockAlign), uint16(blockAlign), uint16(bitsPerSample))
	out.WriteString("data")
	write(uint32(dataSize))
	for index := 0; index < frames*channels; index++ {
		write(int16(index))
	}
	return out.Bytes()
}

// Recording returns a recording that plays, in the format an extension names. A format
// this package cannot make fails the test, rather than quietly writing something else.
func Recording(t testing.TB, extension string) []byte {
	t.Helper()
	switch strings.ToLower(extension) {
	case ".mp3":
		return MP3()
	case ".wav":
		return WAV(t, samplesPerFrame)
	}
	t.Fatalf("no recording can be made in the %q format here; name the take .mp3 or .wav", extension)
	return nil
}

// WriteTake writes a recording that plays at path, in the format its extension names.
func WriteTake(t testing.TB, path string) {
	t.Helper()
	WriteFile(t, path, Recording(t, filepath.Ext(path)))
}

// WriteFile writes body at path, making the directories above it.
func WriteFile(t testing.TB, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
