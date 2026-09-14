package ports

// The ports a machine voice is made through: the model that turns a line into samples, the files
// a voice is made from and the store its made lines are kept in. Each is implemented in
// infrastructure; the making service works against these alone.

import (
	"context"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// SpeechMaker makes a line's samples with the model (FR-511).
//
// Make is handed the numbers the model reads for a line, the boundary at each end included,
// with the style row for that line (speech.Style.For). It answers the samples the model made,
// unchanged. Making one line took 204 to 348 ms, measured, so it is never called between an
// event and its speech.
//
// The context ends when making is stopped (FR-516). An implementation that cannot interrupt the
// model mid-line checks it before starting one; either way the line is not written.
type SpeechMaker interface {
	Make(ctx context.Context, tokens []int64, style []float32) ([]float32, error)
}

// Material is what a machine voice's lines are made from besides their speech sounds: the style
// the model reads and the digests each made line's key is taken from (FR-513).
type Material struct {
	Style speech.Style
	Files making.Files
}

// VoiceFiles reads the files a machine voice is made from: the model, the voice's style file and
// ONNX Runtime.
//
// Open refuses where any of them is missing or cannot be read, with a reason that names that
// file once (FR-519). The refusal comes before anything changes, so the cast it stops leaves
// everything as it was.
type VoiceFiles interface {
	Open(voice machinevoice.Voice) (Material, error)
}

// MadeLines keeps the lines made for machine voices, under the application's own data directory
// and never under the library root (FR-523). A made line is found by its voice and its key.
type MadeLines interface {
	// Keys lists the keys of a voice's made lines on disk. A directory that cannot be read holds
	// none: making them again meets the same fault when it writes, which FR-520 reports.
	Keys(voice machinevoice.Voice) []string

	// Write stores a made line whole or not at all (FR-517), answering why where it cannot
	// (FR-520).
	Write(voice machinevoice.Voice, key string, samples []float32) error

	// Path returns where a made line is played from.
	Path(voice machinevoice.Voice, key string) string

	// DeleteAllBut deletes the made lines of every machine voice except the one given (FR-527),
	// answering why where one cannot be deleted (FR-530).
	DeleteAllBut(voice machinevoice.Voice) error

	// DeleteAll deletes every made line, as casting a recorded voice does (FR-527), answering
	// why where one cannot be deleted (FR-530).
	DeleteAll() error
}
