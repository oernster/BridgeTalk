package audio

// The output device: one oto player fed from a mixer, able to drop the audio already queued
// when a take cuts in (NFR-P-202).
//
// beep's own speaker package wires the same parts but keeps its oto player to itself, so nothing
// queued ahead of a new take could ever be dropped. Measured on 2026-09-13 on a real device: 240 ms
// sat in the player ahead of every new take, most of it silence. Owning the player is what lets
// it be reset.

import (
	"math"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/gopxl/beep/v2"
)

// The stream handed to oto: stereo at sixteen bits a channel.
const (
	channels        = 2
	bytesPerChannel = 2
	bytesPerFrame   = channels * bytesPerChannel
)

// playerBuffer is the cushion the player holds against a late refill, the quarter second beep's
// speaker held (bufferDivisor). driverBuffer is what Windows is asked to hold. It is kept small
// because it can never be dropped: everything in it plays before a take that cuts in (Oliver,
// 2026-09-13). Whether a launch still breaks the speech with it this small is read off the stall
// count on the Status pane.
const (
	playerBuffer = time.Second / bufferDivisor / 2
	driverBuffer = 100 * time.Millisecond
)

// queueSpan is the most audio the player can hold. oto reads a whole buffer's worth whenever its
// buffer is below full, so up to twice the buffer is queued. A mixer that has sounded nothing for
// longer than this holds nothing but silence in the player.
const queueSpan = 2 * playerBuffer

// device is the one output context a process may open; oto refuses a second. It is the single
// package-level value here, forced by that refusal rather than chosen.
var device struct {
	once    sync.Once
	context *oto.Context
	err     error
}

// outputContext opens the device on first use and hands every later caller the same answer.
func outputContext() (*oto.Context, error) {
	device.once.Do(func() {
		context, ready, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   int(deviceSampleRate),
			ChannelCount: channels,
			Format:       oto.FormatSignedInt16LE,
			BufferSize:   driverBuffer,
		})
		if err != nil {
			device.err = err
			return
		}
		<-ready
		device.context = context
	})
	return device.context, device.err
}

// outputQueue is the part of oto's player the speaker drives once it is started: the queue it
// fills, drops and measures. A hand-written fake stands in for it where a test has no device.
type outputQueue interface {
	Play()
	Reset()
	BufferedSize() int
	Close() error
}

// speaker mixes what is playing and hands it to the device.
type speaker struct {
	mu     sync.Mutex
	mixer  beep.Mixer
	player outputQueue
	clock  func() time.Time
	// lastSound is when the mixer last held something to play; zero once the queue is known to
	// hold nothing but silence.
	lastSound time.Time
}

// openSpeaker starts a speaker on the device.
func openSpeaker() (*speaker, error) {
	context, err := outputContext()
	if err != nil {
		return nil, err
	}
	out := &speaker{clock: time.Now}
	player := context.NewPlayer(out)
	player.SetBufferSize(deviceSampleRate.N(playerBuffer) * bytesPerFrame)
	player.Play()
	out.player = player
	return out, nil
}

// Read hands oto the next stretch of mixed audio as sixteen-bit stereo. oto may call it from two
// goroutines at once, so the whole of it runs under the lock.
func (s *speaker) Read(buf []byte) (int, error) {
	samples := make([][2]float64, len(buf)/bytesPerFrame)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mixer.Len() > 0 {
		s.lastSound = s.clock()
	}
	filled, _ := s.mixer.Stream(samples)
	for i, frame := range samples[:filled] {
		for c, value := range frame {
			encoded := int16(math.Max(-1, math.Min(1, value)) * math.MaxInt16)
			at := i*bytesPerFrame + c*bytesPerChannel
			buf[at] = byte(encoded)
			buf[at+1] = byte(encoded >> 8)
		}
	}
	return filled * bytesPerFrame, nil
}

// add starts a streamer. Where the player can hold only silence, that silence is dropped first,
// so the take is not heard a quarter second late. Where a take has just sounded, its end may
// still be queued, so the new one waits for it rather than cutting its last words off.
//
// The reset happens under the lock, before the streamer is added: a read already under way then
// reaches the new take only after the reset, never before it. oto's read releases its own lock
// before calling Read, so holding this one across the reset cannot deadlock.
func (s *speaker) add(streamer beep.Streamer) {
	s.mu.Lock()
	quiet := s.mixer.Len() == 0 && s.clock().Sub(s.lastSound) > queueSpan
	if quiet {
		s.dropQueued()
	}
	s.mixer.Add(streamer)
	s.mu.Unlock()
	if quiet {
		s.player.Play()
	}
}

// stop drops every streamer and the audio queued behind them, so a cut is heard at once. What
// is queued after it is silence, which the next take may drop.
func (s *speaker) stop() {
	s.mu.Lock()
	s.mixer.Clear()
	s.dropQueued()
	s.lastSound = time.Time{}
	s.mu.Unlock()
	s.player.Play()
}

// dropQueued empties the player's queue, leaving it paused until Play. The caller holds the lock.
//
// oto marks Reset deprecated in favour of Pause or Seek; neither will do here. Pause keeps the
// queue, which is the whole problem. Seek resets the queue and then calls back into this speaker
// while holding oto's own lock; Read reaches that same lock through the first-pull record
// (NFR-P-202), so the two would deadlock. Reset is reached through outputQueue, so the linter no
// longer sees the deprecation; this comment is where it is recorded.
func (s *speaker) dropQueued() {
	s.player.Reset()
}

// queued reports how much audio the player holds that the device has not yet taken.
func (s *speaker) queued() time.Duration {
	return deviceSampleRate.D(s.player.BufferedSize() / bytesPerFrame)
}

// close releases the speaker's player. The device stays open, since oto cannot close it.
func (s *speaker) close() {
	_ = s.player.Close()
}
