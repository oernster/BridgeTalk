// Package audio decodes and plays clips.
//
// Decoding and output are pure Go, so the application builds with cgo disabled and
// depends on no system codec, no media framework and no external process. The
// output device runs on its own thread inside the library, which is why nothing
// here may touch the user interface directly.
package audio

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

// deviceSampleRate is the rate the output device runs at. Clips recorded at another
// rate are resampled to it rather than the device being reopened per clip, because
// reopening between clips would click.
const deviceSampleRate beep.SampleRate = 44100

// bufferDivisor sets the device buffer as a fraction of a second.
//
// Read what beep does with it before changing it: Init SPLITS the figure in half,
// giving one half to the driver and one to the player, so the cushion against a late
// refill is half of what this asks for. A tenth of a second therefore left fifty
// milliseconds; fifty milliseconds is not a cushion at all on a machine running
// a game. The refill happens on an ordinary goroutine at ordinary priority, competing
// for the processor and for pages with everything the game is doing; on a launch, the
// busiest moment there is, it lost that race often enough to break the speech up. A
// music player beside the same game does not, because it holds hundreds of
// milliseconds rather than fifty.
//
// A second's worth split in half is a quarter of a second each side. What it costs is
// interruption: an alert cutting in cannot be heard until the audio already handed to
// the device has played, so it arrives up to that much later. Half a second of delay
// on an alert is not something anybody notices; speech breaking up on every launch is.
const bufferDivisor = 2

// stallInterval is how long a silence between two requests for samples has to run
// before it is recorded as the device having gone hungry.
//
// The device asks for samples as it consumes them, so under a device that is being
// fed the requests arrive well inside one buffer's own duration. A gap longer than a
// whole buffer means the buffer emptied before it was refilled, which is what a
// listener hears as a break. It is an indicator rather than a proof: it says the
// refill was late, which is the condition that produces the break.
const stallInterval = time.Second / bufferDivisor

// resampleQuality is beep's interpolation width. Four is its documented general
// choice, good enough for speech without the cost of a wide filter.
const resampleQuality = 4

// volumeBase and volumeOctaves shape the gain curve. Perceived loudness is roughly
// logarithmic in gain, so a linear slider mapped straight onto gain spends most of
// its travel in a range that all sounds the same. The slider position instead picks
// a point volumeOctaves below full; the gain is that many halvings. At the top the
// clip plays as recorded, at the middle it is an eighth of that, at the bottom it is
// silence.
//
// Two is beep's documented natural base. The curve is computed here rather than
// through effects.Volume because that type reads its own fields; a slider has to be
// heard while a clip is already playing.
const (
	volumeBase    = 2
	volumeOctaves = 6
)

// fullVolume is the level at which a clip plays exactly as recorded.
const fullVolume = 1.0

// ErrNoClips is returned when a request carries nothing playable.
var ErrNoClips = errors.New("no clips to play")

// Player plays a sequence of clips, one at a time, with a gap between them.
type Player struct {
	mu       sync.Mutex
	playing  bool
	cancel   chan struct{}
	finished chan struct{}
	silent   bool
	volume   float64

	// What the device asked for and when, so a refill that arrived too late can be
	// counted rather than guessed at. lastPull is zero between clips, where a long
	// silence is the gap between clips rather than a fault.
	lastPull time.Time
	stalls   int
	worst    time.Duration
}

// NewPlayer opens the output device.
//
// A device that cannot be opened is not fatal: the application still watches the
// journal and still reports what it would have said, which is far more useful than
// refusing to start on a machine with no sound.
func NewPlayer() (*Player, error) {
	player := &Player{finished: make(chan struct{}, 1), volume: fullVolume}
	err := speaker.Init(deviceSampleRate, deviceSampleRate.N(time.Second/bufferDivisor))
	if err != nil {
		player.silent = true
		return player, fmt.Errorf("opening audio device: %w", err)
	}
	return player, nil
}

// Silent reports whether the player is running without an output device.
func (p *Player) Silent() bool { return p.silent }

// SetVolume sets the playback level, where zero is silence and one is the clip as
// recorded. A level outside that range is clamped rather than refused, because the
// caller is a slider and a slider cannot usefully be told it is wrong.
func (p *Player) SetVolume(level float64) {
	switch {
	case level < 0:
		level = 0
	case level > fullVolume:
		level = fullVolume
	}
	p.mu.Lock()
	p.volume = level
	p.mu.Unlock()
}

// Volume reports the playback level.
func (p *Player) Volume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

// gain converts the level into the multiplier applied to each sample. It is read
// once per buffer rather than once per clip, so moving the slider is heard straight
// away instead of at the next clip.
func (p *Player) gain() float64 {
	level := p.Volume()
	if level <= 0 {
		return 0
	}
	return math.Pow(volumeBase, (level-fullVolume)*volumeOctaves)
}

// Stalls reports how many times the device went hungry and the longest it waited.
//
// It is cumulative over the run and shown where it can be read, because a break in
// the speech is otherwise something only the listener knows about and only in words.
// A machine that never starves the player reports nothing at all.
func (p *Player) Stalls() (count int, worst time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stalls, p.worst
}

// beginClip forgets the last request, so the silence between two clips is
// not counted as the device having waited for one.
func (p *Player) beginClip() {
	p.mu.Lock()
	p.lastPull = time.Time{}
	p.mu.Unlock()
}

// observePull records that the device asked for samples, counting the wait since it
// last asked where that ran past a whole buffer.
func (p *Player) observePull(at time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.lastPull.IsZero() {
		if waited := at.Sub(p.lastPull); waited > stallInterval {
			p.stalls++
			if waited > p.worst {
				p.worst = waited
			}
		}
	}
	p.lastPull = at
}

// levelled wraps a streamer so every buffer it produces is scaled by the player's
// current gain, with the device's request for it timed on the way through.
type levelled struct {
	source beep.Streamer
	player *Player
	clock  func() time.Time
}

// Stream fills the buffer from the wrapped streamer, then scales it.
func (l levelled) Stream(samples [][2]float64) (int, bool) {
	l.player.observePull(l.clock())
	filled, ok := l.source.Stream(samples)
	gain := l.player.gain()
	if gain == fullVolume {
		return filled, ok
	}
	for i := range samples[:filled] {
		samples[i][0] *= gain
		samples[i][1] *= gain
	}
	return filled, ok
}

// Err propagates the wrapped streamer's error.
func (l levelled) Err() error { return l.source.Err() }

// Done yields once per completed or stopped sequence.
func (p *Player) Done() <-chan struct{} { return p.finished }

// Playing reports whether a sequence is in progress.
func (p *Player) Playing() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

// Play starts a sequence and returns immediately.
func (p *Player) Play(clips []string, gap time.Duration) error {
	if len(clips) == 0 {
		return ErrNoClips
	}
	p.Stop()

	p.mu.Lock()
	p.playing = true
	cancel := make(chan struct{})
	p.cancel = cancel
	p.mu.Unlock()

	if p.silent {
		p.complete(cancel)
		return nil
	}
	go p.run(clips, gap, cancel)
	return nil
}

// run plays each clip in turn, honouring the cancel signal between and during them.
func (p *Player) run(clips []string, gap time.Duration, cancel chan struct{}) {
	defer p.complete(cancel)
	for index, clip := range clips {
		select {
		case <-cancel:
			return
		default:
		}
		if index > 0 && gap > 0 && !sleepOrCancel(gap, cancel) {
			return
		}
		if !p.playOne(clip, cancel) {
			return
		}
	}
}

// playOne reads and plays a single clip, returning false when cancelled.
//
// The whole clip is read into memory here, on this goroutine, before the device is
// given anything. That is the point: the decoding and the resampling happen while
// nothing is waiting on them rather than inside the device's own request for the next
// buffer, where being slow is heard rather than merely being slow.
func (p *Player) playOne(path string, cancel chan struct{}) bool {
	source, err := load(path)
	if err != nil {
		// One unreadable clip should not abandon the rest of the sequence.
		return true
	}

	p.beginClip()
	ended := make(chan struct{})
	speaker.Play(beep.Seq(
		levelled{source: source, player: p, clock: time.Now},
		beep.Callback(func() { close(ended) }),
	))

	select {
	case <-ended:
		return true
	case <-cancel:
		speaker.Clear()
		return false
	}
}

// sleepOrCancel waits for a gap, returning false when cancelled during it.
func sleepOrCancel(gap time.Duration, cancel chan struct{}) bool {
	timer := time.NewTimer(gap)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-cancel:
		return false
	}
}

// complete marks a sequence finished and signals any waiter.
//
// Play cancels whatever is running before it starts the next sequence, so the
// goroutine it cancelled finishes after the replacement has already begun. It is
// handed the cancel channel it was started with and clears the shared state only
// while that channel is still the current one; otherwise it would report the new
// sequence as idle while it is audibly playing. The signal is sent either way,
// because a waiter is counting sequences rather than tracking which one ended.
func (p *Player) complete(cancel chan struct{}) {
	p.mu.Lock()
	if p.cancel == cancel {
		p.playing = false
		p.cancel = nil
	}
	p.mu.Unlock()
	select {
	case p.finished <- struct{}{}:
	default:
	}
}

// Stop ends the current sequence. It is safe to call when nothing is playing.
func (p *Player) Stop() {
	p.mu.Lock()
	cancel := p.cancel
	p.cancel = nil
	p.playing = false
	p.mu.Unlock()
	if cancel != nil {
		close(cancel)
	}
	if !p.silent {
		speaker.Clear()
	}
}

// Close stops playback and releases the device.
func (p *Player) Close() error {
	p.Stop()
	if p.silent {
		return nil
	}
	speaker.Close()
	return nil
}
