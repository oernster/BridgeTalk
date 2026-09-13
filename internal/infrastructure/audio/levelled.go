package audio

// What the device is handed: every buffer scaled by the current gain and every request
// for one timed, so a refill that arrived too late is counted rather than guessed at.
//
// It sits beside player.go rather than inside it because it is a concern of its own.
// The player decides what plays and when; this is what happens to each buffer on its
// way to the device.

import (
	"time"

	"github.com/gopxl/beep/v2"
)

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
	if p.lastPull.IsZero() && p.out != nil {
		// The clip's first request: note it and what sits ahead of it (NFR-P-202).
		p.firstPull = at
		p.queuedAhead = p.out.queued()
	}
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
