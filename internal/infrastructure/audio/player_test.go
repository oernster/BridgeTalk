package audio

import (
	"math"
	"testing"
	"time"
)

// TestVolumeCurveIsNotLinear pins the shape of the gain curve.
//
// It exists because the first version of this curve computed base to the power of
// the log of the level in the same base, which is the level itself: a plain linear
// gain wearing the comment of an exponential one. A test that only checked the ends
// would have passed it, so the middle is what is asserted here.
func TestVolumeCurveIsNotLinear(t *testing.T) {
	t.Parallel()
	player := &Player{finished: make(chan struct{}, 1), volume: fullVolume}

	if gain := player.gain(); gain != fullVolume {
		t.Errorf("gain at full level is %v, want %v: a clip must play as recorded", gain, fullVolume)
	}

	player.SetVolume(0.5)
	half := player.gain()
	if half >= 0.5 {
		t.Errorf("gain at half level is %v, want well below 0.5: the curve is linear", half)
	}
	if want := math.Pow(volumeBase, -0.5*volumeOctaves); math.Abs(half-want) > 1e-9 {
		t.Errorf("gain at half level is %v, want %v", half, want)
	}

	player.SetVolume(0)
	if gain := player.gain(); gain != 0 {
		t.Errorf("gain at zero is %v, want silence", gain)
	}
}

func TestSetVolumeClampsOutOfRange(t *testing.T) {
	t.Parallel()
	player := &Player{finished: make(chan struct{}, 1), volume: fullVolume}

	player.SetVolume(4)
	if level := player.Volume(); level != fullVolume {
		t.Errorf("level above the top is %v, want it clamped to %v", level, fullVolume)
	}
	player.SetVolume(-4)
	if level := player.Volume(); level != 0 {
		t.Errorf("level below the bottom is %v, want it clamped to 0", level)
	}
}

// stubStreamer fills every sample with a known value so scaling is measurable.
type stubStreamer struct{ remaining int }

func (s *stubStreamer) Stream(samples [][2]float64) (int, bool) {
	if s.remaining <= 0 {
		return 0, false
	}
	filled := min(len(samples), s.remaining)
	for i := range samples[:filled] {
		samples[i] = [2]float64{1, 1}
	}
	s.remaining -= filled
	return filled, true
}

func (s *stubStreamer) Err() error { return nil }

func TestLevelledScalesEverySample(t *testing.T) {
	t.Parallel()
	player := &Player{finished: make(chan struct{}, 1), volume: fullVolume}
	player.SetVolume(0.5)
	wrapped := levelled{source: &stubStreamer{remaining: 3}, player: player, clock: time.Now}

	samples := make([][2]float64, 3)
	filled, ok := wrapped.Stream(samples)
	if !ok || filled != 3 {
		t.Fatalf("stream returned %d, %v, want 3, true", filled, ok)
	}
	want := player.gain()
	for i, sample := range samples {
		if sample[0] != want || sample[1] != want {
			t.Errorf("sample %d is %v, want both channels at %v", i, sample, want)
		}
	}
}

// TestLevelledLeavesFullVolumeUntouched covers the shortcut that skips the scaling
// loop, so a clip at full level is bit-identical to the decoded audio.
func TestLevelledLeavesFullVolumeUntouched(t *testing.T) {
	t.Parallel()
	player := &Player{finished: make(chan struct{}, 1), volume: fullVolume}
	wrapped := levelled{source: &stubStreamer{remaining: 2}, player: player, clock: time.Now}

	samples := make([][2]float64, 2)
	if _, ok := wrapped.Stream(samples); !ok {
		t.Fatal("stream reported no data")
	}
	for i, sample := range samples {
		if sample[0] != 1 || sample[1] != 1 {
			t.Errorf("sample %d is %v, want it unchanged at full level", i, sample)
		}
	}
	if err := wrapped.Err(); err != nil {
		t.Errorf("Err returned %v, want nil", err)
	}
}

// Stopping is not merely cancelling: it must leave the player reporting idle. A
// caller that asks "is anything playing" otherwise gets a stale yes until the
// cancelled goroutine happens to wake up.
func TestStopLeavesThePlayerIdle(t *testing.T) {
	player := &Player{finished: make(chan struct{}, 1), silent: true, volume: fullVolume}
	cancel := make(chan struct{})
	player.playing = true
	player.cancel = cancel

	player.Stop()

	if player.Playing() {
		t.Fatal("the player still reports playing after Stop")
	}
}

// Play stops whatever is running before starting the next sequence, so the goroutine
// it just cancelled finishes AFTERWARDS. Without a guard that late completion clears
// the state of the sequence that replaced it, so the caller is told nothing is
// playing while a clip is audibly playing.
func TestALateCompletionDoesNotClearTheSequenceThatReplacedIt(t *testing.T) {
	player := &Player{finished: make(chan struct{}, 1), silent: true, volume: fullVolume}
	superseded := make(chan struct{})
	player.playing = true
	player.cancel = superseded

	player.Stop()

	current := make(chan struct{})
	player.playing = true
	player.cancel = current

	player.complete(superseded)

	if !player.Playing() {
		t.Fatal("a superseded sequence's completion cleared the current one")
	}

	player.complete(current)

	if player.Playing() {
		t.Fatal("the current sequence's own completion left it playing")
	}
}

// The device asks for samples as it consumes them. If a whole buffer's worth of time
// passes between two requests, the buffer emptied before it was refilled, which is
// what the listener hears as the speech breaking up. Counting it is the only way that
// fact reaches anybody but the listener.
func TestARefillThatArrivesLateIsCountedAsTheDeviceRunningDry(t *testing.T) {
	t.Parallel()
	player := &Player{finished: make(chan struct{}, 1), volume: fullVolume}
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	pull := func(after time.Duration) {
		now = now.Add(after)
		player.observePull(now)
	}

	player.beginClip()
	pull(0)
	// Two refills well inside a buffer, which is what a fed device looks like.
	pull(stallInterval / 4)
	pull(stallInterval / 4)
	if count, _ := player.Stalls(); count != 0 {
		t.Fatalf("a device that was kept fed reported %d stalls", count)
	}

	// One that arrives after the buffer would have emptied.
	late := stallInterval * 3
	pull(late)
	count, worst := player.Stalls()
	if count != 1 {
		t.Fatalf("stalls = %d, want 1", count)
	}
	if worst != late {
		t.Fatalf("worst = %v, want %v", worst, late)
	}

	// A longer one replaces the worst; a shorter one does not.
	pull(stallInterval * 5)
	pull(stallInterval * 2)
	if count, worst = player.Stalls(); count != 3 || worst != stallInterval*5 {
		t.Fatalf("stalls = %d worst = %v, want 3 and %v", count, worst, stallInterval*5)
	}
}

// The silence between two clips is not a fault. Timing across it
// would report a stall on every gap and make the count worthless.
func TestTheGapBetweenClipsIsNotAStall(t *testing.T) {
	t.Parallel()
	player := &Player{finished: make(chan struct{}, 1), volume: fullVolume}
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)

	player.beginClip()
	player.observePull(now)

	// One clip ends, a long gap passes, the next clip starts.
	now = now.Add(stallInterval * 10)
	player.beginClip()
	player.observePull(now)

	if count, _ := player.Stalls(); count != 0 {
		t.Fatalf("the gap between two parts was counted as %d stalls", count)
	}
}
