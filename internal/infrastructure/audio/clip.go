// Reading a clip off disk and turning it into samples the device can take.
//
// It sits beside player.go because it is a concern that comes out whole: everything
// here happens before a clip reaches the speaker; nothing here knows the speaker
// exists at all.
package audio

import (
	"fmt"

	"github.com/gopxl/beep/v2"
)

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

// decode opens a clip with the decoder matching its extension, naming the clip in any
// error it returns.
func decode(path string) (beep.StreamSeekCloser, beep.Format, func() error, error) {
	streamer, format, closer, err := open(path)
	if err != nil {
		return nil, beep.Format{}, closer, fmt.Errorf("%s: %w", path, err)
	}
	return streamer, format, closer, nil
}
