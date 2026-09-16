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
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"

	"github.com/oernster/bridge-talk/internal/refusal"
)

// deviceSampleRate is the rate the output device runs at. Clips recorded at another
// rate are resampled to it rather than the device being reopened per clip, because
// reopening between clips would click.
const deviceSampleRate beep.SampleRate = 44100

// bufferDivisor sets, as a fraction of a second, the audio this package reasons in: half a second,
// of which the player holds a quarter against a late refill (speaker.go).
//
// A tenth of a second once left fifty milliseconds in the player, which is no cushion at all on a
// machine running a game. The refill happens on an ordinary goroutine at ordinary priority,
// competing for the processor and for pages with everything the game is doing; on a launch, the
// busiest moment there is, it lost that race often enough to break the speech up.
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
	out      *speaker

	// load reads a clip whole; left nil, the package's own load does. A test sets it to hold a
	// read part way, which is where a take can be cancelled while nothing of it has reached the
	// speaker yet.
	load func(path string) (beep.Streamer, error)

	// record notes a part that would not open; left nil, a line on error output does, which is
	// where the run log keeps it (FR-574, FR-715). A test sets it to read the note back.
	record func(line string)

	// sequence is held while a sequence is cancelled and the speaker cleared behind it; it is also
	// held while a clip read for a sequence is handed to the speaker. A clip read for a sequence
	// cancelled meanwhile is then never added after the clear. It is not p.mu: the speaker's
	// Read holds the speaker's lock and takes p.mu, so p.mu held across an add could deadlock.
	sequence sync.Mutex

	// When the current clip was first asked for and how much audio sat ahead of it then, which
	// is what the latency benchmark reads (NFR-P-202).
	firstPull   time.Time
	queuedAhead time.Duration

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
	out, err := openSpeaker()
	if err != nil {
		player.silent = true
		return player, fmt.Errorf("opening audio device: %w", err)
	}
	player.out = out
	return player, nil
}

// Silent reports whether the player is running without an output device.
func (p *Player) Silent() bool { return p.silent }

// Done yields once per completed or stopped sequence.
func (p *Player) Done() <-chan struct{} { return p.finished }

// Playing reports whether a sequence is in progress.
func (p *Player) Playing() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

// Play starts a sequence and returns immediately, ending whatever was playing first.
//
// The old sequence is replaced and the new one claimed under the one lock, so there is
// no moment reading idle between them for PlayIfIdle to start something in.
//
// Only a sequence still playing is cut, with the audio queued behind it (NFR-P-202). A
// sequence reads as ended once its last samples are queued, which is before the device
// has played them; dropping the queue then would cut off the end of a take that has
// already finished. With nothing to replace, the speaker decides: queued silence goes,
// the end of a take just sounded is waited for.
func (p *Player) Play(clips []string, gap time.Duration) error {
	if len(clips) == 0 {
		return ErrNoClips
	}
	p.sequence.Lock()
	p.mu.Lock()
	replaced := p.cancel
	cancel := p.claim()
	p.mu.Unlock()

	if replaced != nil {
		close(replaced)
		if !p.silent {
			p.out.stop()
		}
	}
	p.sequence.Unlock()
	p.launch(clips, gap, cancel)
	return nil
}

// PlayIfIdle starts a sequence only when nothing is playing, reporting whether it did.
//
// A press on a button must never cut short a clip already sounding (FR-236). Asking
// Playing and then calling Play leaves a gap in which another caller can start
// something, so the question and the start are taken under the one lock.
func (p *Player) PlayIfIdle(clips []string, gap time.Duration) (bool, error) {
	if len(clips) == 0 {
		return false, ErrNoClips
	}
	p.mu.Lock()
	if p.playing {
		p.mu.Unlock()
		return false, nil
	}
	cancel := p.claim()
	p.mu.Unlock()

	p.launch(clips, gap, cancel)
	return true, nil
}

// claim marks a sequence as playing and returns the channel that cancels it. The
// caller holds the lock.
func (p *Player) claim() chan struct{} {
	p.playing = true
	cancel := make(chan struct{})
	p.cancel = cancel
	return cancel
}

// launch runs a claimed sequence; with no device it completes at once.
func (p *Player) launch(clips []string, gap time.Duration, cancel chan struct{}) {
	if p.silent {
		p.complete(cancel)
		return
	}
	go p.run(clips, gap, cancel)
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
	source, err := p.loadClip(path)
	if err != nil {
		// One unreadable clip should not abandon the rest of the sequence; the part passed
		// over is recorded rather than dropped in silence (FR-574). Most of a line is better
		// than none of it; nothing else would ever say a part is missing, since a take short of
		// a part still sounds like a take.
		p.passedOver(path, err)
		return true
	}

	// The read can outlast the sequence it was for; a Play or a Stop meanwhile has already
	// cleared the speaker. The clip is handed over only while its sequence still stands, checked
	// under the lock they clear under, so it can never land behind a clear (NFR-P-202).
	p.sequence.Lock()
	select {
	case <-cancel:
		p.sequence.Unlock()
		return false
	default:
	}
	p.beginClip()
	ended := make(chan struct{})
	p.out.add(beep.Seq(
		levelled{source: source, player: p, clock: time.Now},
		beep.Callback(func() { close(ended) }),
	))
	p.sequence.Unlock()

	// A cancel needs nothing dropped here: Play and Stop have already dropped this clip along
	// with the audio queued behind it; dropping again could take the take that replaced it.
	select {
	case <-ended:
		return true
	case <-cancel:
		return false
	}
}

// passedOver records one part of a take that would not open, with the reason it would not.
//
// The audio belongs to the user and can be moved or removed at any time without the voice
// offering it knowing, so this is an ordinary event rather than a fault: it is a note, worded
// as every other thing passed over is; it stops nothing.
func (p *Player) passedOver(path string, err error) {
	line := refusal.PassedOver("the part "+path, refusal.Reason(err).Error())
	if p.record != nil {
		p.record(line)
		return
	}
	fmt.Fprintln(os.Stderr, line)
}

// loadClip reads a clip whole with the player's load where one is set, else the package's own.
func (p *Player) loadClip(path string) (beep.Streamer, error) {
	if p.load != nil {
		return p.load(path)
	}
	return load(path)
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
	p.sequence.Lock()
	defer p.sequence.Unlock()
	p.mu.Lock()
	cancel := p.cancel
	p.cancel = nil
	p.playing = false
	p.mu.Unlock()
	if cancel != nil {
		close(cancel)
	}
	if !p.silent {
		p.out.stop()
	}
}

// Close stops playback and releases the player's hold on the device.
func (p *Player) Close() error {
	p.Stop()
	if p.silent {
		return nil
	}
	p.out.close()
	return nil
}
