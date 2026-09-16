// The formats the player decodes, plus the question the scan asks of a take before it
// offers one: will it play?
//
// It sits beside clip.go because both open a clip by its extension; the list of extensions
// has one home, here, so the scan cannot come to recognise a format the player cannot read.
package audio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"

	"github.com/oernster/bridge-talk/internal/domain/take"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// ErrUnsupportedFormat is returned for a file the decoders do not recognise.
var ErrUnsupportedFormat = errors.New("unsupported audio format")

// ErrNoSound is returned for a file that decodes to no samples at all.
var ErrNoSound = errors.New("it holds no sound")

// ErrSpanOutsideFile is returned for a span whose offset or length is negative or which reaches
// past the end of its file (FR-590).
var ErrSpanOutsideFile = errors.New("its span does not lie inside its file")

// opener decodes an open part in one format.
type opener func(part io.ReadSeekCloser) (beep.StreamSeekCloser, beep.Format, error)

// openers are the formats the player decodes, keyed by extension in lower case. A span names its
// format by the same word without the dot (FR-589), so this stays the one list of formats.
var openers = map[string]opener{
	".mp3":  func(part io.ReadSeekCloser) (beep.StreamSeekCloser, beep.Format, error) { return mp3.Decode(part) },
	".wav":  func(part io.ReadSeekCloser) (beep.StreamSeekCloser, beep.Format, error) { return wav.Decode(part) },
	".flac": func(part io.ReadSeekCloser) (beep.StreamSeekCloser, beep.Format, error) { return flac.Decode(part) },
	".ogg":  func(part io.ReadSeekCloser) (beep.StreamSeekCloser, beep.Format, error) { return vorbis.Decode(part) },
}

// extensionMark is what a format word is given to read as an extension.
const extensionMark = "."

// formatOf answers the key a part's decoder is found by: its file's extension for a whole file,
// the format it names for a span, whose file's own extension says nothing about what is inside.
func formatOf(part take.Part) string {
	if part.Span != nil {
		return extensionMark + strings.ToLower(part.Span.Format)
	}
	return strings.ToLower(filepath.Ext(part.Path))
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
	streamer, _, closer, err := open(take.File(path))
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

// open opens a part with the decoder its format names. Its errors name no file, for the reason
// Playable gives; decode adds the part for the player.
func open(part take.Part) (beep.StreamSeekCloser, beep.Format, func() error, error) {
	decoder, known := openers[formatOf(part)]
	if !known {
		return nil, beep.Format{}, noCloser, fmt.Errorf("%w: %s", ErrUnsupportedFormat, formatOf(part))
	}
	// Opened for reading alone: a part is the user's audio, never written (FR-572).
	file, err := os.Open(part.Path)
	if err != nil {
		return nil, beep.Format{}, noCloser, refusal.Reason(err)
	}
	reading, err := readerFor(file, part.Span)
	if err != nil {
		_ = file.Close()
		return nil, beep.Format{}, noCloser, err
	}
	streamer, format, err := decoder(reading)
	if err != nil {
		_ = file.Close()
		return nil, beep.Format{}, noCloser, err
	}
	return streamer, format, file.Close, nil
}

// spanReader is a stretch of an open file that reads, seeks and closes as a file of its own, so
// a decoder cannot tell it from one.
type spanReader struct {
	*io.SectionReader
	io.Closer
}

// readerFor answers what a decoder reads: the whole file; the span of it a part names where it is one.
//
// A span is foreign input, since a plugin gave it. A negative offset or length is refused by the part
// rather than read; so is a span reaching past the end of the file as it stands now (FR-590). The size
// is read as the part is opened, so a file changed since the plugin looked is measured as it is rather
// than as it was.
func readerFor(file *os.File, span *take.Span) (io.ReadSeekCloser, error) {
	if span == nil {
		return file, nil
	}
	info, err := file.Stat()
	if err != nil {
		return nil, refusal.Reason(err)
	}
	if span.Offset < 0 || span.Length < 0 || span.Offset > info.Size()-span.Length {
		return nil, fmt.Errorf("%w: %d bytes from byte %d of a file of %d bytes",
			ErrSpanOutsideFile, span.Length, span.Offset, info.Size())
	}
	return spanReader{io.NewSectionReader(file, span.Offset, span.Length), file}, nil
}
