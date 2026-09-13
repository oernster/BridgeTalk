// The formats the player decodes, plus the question the scan asks of a take before it
// offers one: will it play?
//
// It sits beside clip.go because both open a clip by its extension; the list of extensions
// has one home, here, so the scan cannot come to recognise a format the player cannot read.
package audio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"

	"github.com/oernster/bridge-talk/internal/refusal"
)

// ErrUnsupportedFormat is returned for a file the decoders do not recognise.
var ErrUnsupportedFormat = errors.New("unsupported audio format")

// ErrNoSound is returned for a file that decodes to no samples at all.
var ErrNoSound = errors.New("it holds no sound")

// opener decodes an open file in one format.
type opener func(file *os.File) (beep.StreamSeekCloser, beep.Format, error)

// openers are the formats the player decodes, keyed by extension in lower case.
var openers = map[string]opener{
	".mp3":  func(file *os.File) (beep.StreamSeekCloser, beep.Format, error) { return mp3.Decode(file) },
	".wav":  func(file *os.File) (beep.StreamSeekCloser, beep.Format, error) { return wav.Decode(file) },
	".flac": func(file *os.File) (beep.StreamSeekCloser, beep.Format, error) { return flac.Decode(file) },
	".ogg":  func(file *os.File) (beep.StreamSeekCloser, beep.Format, error) { return vorbis.Decode(file) },
}

// probeSpan is how much of a take Playable decodes before calling it playable.
const probeSpan = 10 * time.Millisecond

// Recognised reports whether a file name carries an extension the player decodes, compared
// case insensitively (FR-203).
func Recognised(name string) bool {
	_, known := openers[strings.ToLower(filepath.Ext(name))]
	return known
}

// Playable reports why a file cannot be played; nil when it decodes to sound (FR-204).
//
// It decodes the start of the file rather than the whole of it: a scan runs over every take
// in the library, while a take is decoded whole only when it is about to be heard. A file
// whose start plays but whose middle is damaged is therefore still offered.
//
// The error names no file. The scan names each take by where it was found, so a path here
// would be read twice.
func Playable(path string) error {
	streamer, _, closer, err := open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = streamer.Close()
		_ = closer()
	}()
	samples := make([][2]float64, deviceSampleRate.N(probeSpan))
	if filled, _ := streamer.Stream(samples); filled == 0 {
		if err := streamer.Err(); err != nil {
			return err
		}
		return ErrNoSound
	}
	return streamer.Err()
}

// noCloser stands in where nothing was opened, so a caller can defer the closer either way.
func noCloser() error { return nil }

// open opens a clip with the decoder its extension names. Its errors name no file, for the
// reason Playable gives; decode adds the path for the player.
func open(path string) (beep.StreamSeekCloser, beep.Format, func() error, error) {
	decoder, known := openers[strings.ToLower(filepath.Ext(path))]
	if !known {
		return nil, beep.Format{}, noCloser, fmt.Errorf("%w: %s", ErrUnsupportedFormat, filepath.Ext(path))
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, noCloser, refusal.Reason(err)
	}
	streamer, format, err := decoder(file)
	if err != nil {
		_ = file.Close()
		return nil, beep.Format{}, noCloser, err
	}
	return streamer, format, file.Close, nil
}
