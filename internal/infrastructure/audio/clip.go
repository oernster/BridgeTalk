// Reading a clip off disk and turning it into samples the device can take.
//
// It sits beside player.go because it is a concern that comes out whole: everything
// here happens before a clip reaches the speaker; nothing here knows the speaker
// exists at all.
package audio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"
)

// ErrUnsupportedFormat is returned for a file the decoders do not recognise.
var ErrUnsupportedFormat = errors.New("unsupported audio format")

// deviceFormat is the shape every clip is converted to before it is played: the
// device's own rate, stereo, at the sixteen bits the output stream carries anyway.
// Holding a clip in any other shape would only move the conversion to a worse place.
var deviceFormat = beep.Format{
	SampleRate:  deviceSampleRate,
	NumChannels: 2,
	Precision:   2,
}

// load reads a whole clip into memory, resampled to the device rate.
//
// The alternative, handing the decoder straight to the speaker, decodes each buffer
// inside the device's own callback: the mp3 frames, the resampling and the file reads
// behind them all happen on the thread that must not be late. That thread is a plain
// goroutine at ordinary priority, so under a game that is using the whole machine it
// is competing for CPU and for pages with everything else. Being late there is not
// a slow clip; it is a broken one.
//
// Doing the work here moves all of it onto the player's own goroutine, in the gap
// before the part is due, where being slow costs nothing audible. What the device
// then asks for is a copy out of a buffer already sitting in memory.
//
// The cost is the memory a decoded clip occupies while it plays. At four bytes a
// frame the longest line in any of these voices is a few megabytes and only one clip
// is ever loaded at a time, which is a trade worth making against speech breaking up.
func load(path string) (beep.Streamer, error) {
	streamer, format, closer, err := decode(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = streamer.Close()
		_ = closer()
	}()

	source := beep.Streamer(streamer)
	if format.SampleRate != deviceSampleRate {
		source = beep.Resample(resampleQuality, format.SampleRate, deviceSampleRate, streamer)
	}

	buffer := beep.NewBuffer(deviceFormat)
	buffer.Append(source)
	if buffer.Len() == 0 {
		return nil, fmt.Errorf("%q decoded to no audio", path)
	}
	return buffer.Streamer(0, buffer.Len()), nil
}

// decode opens a clip with the decoder matching its extension.
func decode(path string) (beep.StreamSeekCloser, beep.Format, func() error, error) {
	handle, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, func() error { return nil }, fmt.Errorf("opening %q: %w", path, err)
	}
	closer := handle.Close

	var (
		streamer beep.StreamSeekCloser
		format   beep.Format
	)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		streamer, format, err = mp3.Decode(handle)
	case ".wav":
		streamer, format, err = wav.Decode(handle)
	case ".flac":
		streamer, format, err = flac.Decode(handle)
	case ".ogg":
		streamer, format, err = vorbis.Decode(handle)
	default:
		_ = closer()
		return nil, beep.Format{}, func() error { return nil },
			fmt.Errorf("%w: %s", ErrUnsupportedFormat, filepath.Ext(path))
	}
	if err != nil {
		_ = closer()
		return nil, beep.Format{}, func() error { return nil }, fmt.Errorf("decoding %q: %w", path, err)
	}
	return streamer, format, closer, nil
}
