// How loud a clip is played: the curve behind the slider and the level it sets.
//
// It is apart from player.go because it is a concern of its own with an answer of its own to
// give: what a slider position means in gain. The level is read once per buffer rather than once
// per clip, so moving the slider is heard straight away, which is why gain sits beside the curve
// it reads rather than beside the sequencing.
package audio

import "math"

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
