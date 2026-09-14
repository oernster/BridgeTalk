// Package voicefiles reads the files a machine voice is made from out of the folder setup fills
// (FR-524): the model, ONNX Runtime and a style file for each voice.
//
// A file that is missing or cannot be read refuses the voice, naming that file once (FR-519).
// Nothing is kept between casts: the model's digest took 173 to 176 ms, measured on 2026-09-14 with
// the file already in the system's cache, which a cast can afford.
package voicefiles

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/speech"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// The names setup gives the shared files in the folder.
const (
	// ModelFile is the model every machine voice is made with.
	ModelFile = "model.onnx"
	// RuntimeFile is ONNX Runtime, which runs the model.
	RuntimeFile = "onnxruntime.dll"
)

// styleExtension follows a voice's id to name its style file, such as bf_emma.bin.
const styleExtension = ".bin"

// StyleFile names a voice's style file in the folder: its id followed by .bin, such as bf_emma.bin.
func StyleFile(voice machinevoice.Voice) string { return voice.ID() + styleExtension }

// float32Bytes is how many bytes one number of a style file takes.
const float32Bytes = 4

// Files reads machine voices' files from one folder.
type Files struct {
	dir string
}

// New reads from the folder given.
func New(dir string) Files { return Files{dir: dir} }

// Open reads what a voice's lines are made from: its style and the digests of its style file and
// the model (FR-513). ONNX Runtime is read as well, so a damaged one refuses the cast rather than
// the first line made (FR-519).
func (f Files) Open(voice machinevoice.Voice) (ports.Material, error) {
	if _, err := digestFile(filepath.Join(f.dir, RuntimeFile)); err != nil {
		return ports.Material{}, err
	}
	style, styleDigest, err := f.style(voice)
	if err != nil {
		return ports.Material{}, err
	}
	modelDigest, err := digestFile(filepath.Join(f.dir, ModelFile))
	if err != nil {
		return ports.Material{}, err
	}
	return ports.Material{Style: style, Files: making.Files{Style: styleDigest, Model: modelDigest}}, nil
}

// style reads a voice's style file as the numbers the model reads, with the file's digest.
func (f Files) style(voice machinevoice.Voice) (speech.Style, string, error) {
	path := filepath.Join(f.dir, StyleFile(voice))
	raw, err := os.ReadFile(path)
	if err != nil {
		return speech.Style{}, "", fmt.Errorf("reading %s: %w", path, refusal.Reason(err))
	}
	if len(raw)%float32Bytes != 0 {
		return speech.Style{}, "", fmt.Errorf("reading %s: %w: %d bytes", path, speech.ErrMisshapenStyle, len(raw))
	}
	values := make([]float32, len(raw)/float32Bytes)
	for index := range values {
		values[index] = math.Float32frombits(binary.LittleEndian.Uint32(raw[index*float32Bytes:]))
	}
	style, err := speech.NewStyle(values)
	if err != nil {
		return speech.Style{}, "", fmt.Errorf("reading %s: %w", path, err)
	}
	// A bytes.Reader never fails a read, so there is no error to handle.
	sum, _ := digest(bytes.NewReader(raw))
	return style, sum, nil
}

// digestFile reads a file whole into its digest, refusing one that cannot be read.
func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, refusal.Reason(err))
	}
	defer file.Close()
	sum, err := digest(file)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, refusal.Reason(err))
	}
	return sum, nil
}

// digest is the SHA-256 of everything a reader holds, written as hexadecimal.
func digest(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
