package audio

// NFR-P-202 on a real device: a take begins within 150 milliseconds at the 95th percentile, over
// 100 firings. Output is counted from the call that plays the take to its first samples being
// taken, plus the audio queued ahead of them in the player then, plus the Windows buffer at its
// full size, since how full that buffer is cannot be read. A machine with no device skips it.

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

// The figures NFR-P-202 states.
const (
	latencyBudget     = 150 * time.Millisecond
	latencyFirings    = 100
	latencyPercentile = 95
)

// firstPullWait is how long one firing may take to be asked for before the run gives up on it.
const firstPullWait = 2 * time.Second

// awaitFirstPull waits for the clip just played to be asked for, answering with when that
// happened and how much audio sat ahead of it.
//
// It waits for the record to be set rather than for a time after the call. The Windows clock
// ticks coarsely enough that a pull straight after the call reads as the same instant: measured,
// a pull recorded at exactly the call's time was never seen as later than it.
func awaitFirstPull(t *testing.T, player *Player) (time.Time, time.Duration) {
	t.Helper()
	deadline := time.Now().Add(firstPullWait)
	for time.Now().Before(deadline) {
		player.mu.Lock()
		at, ahead := player.firstPull, player.queuedAhead
		player.mu.Unlock()
		if !at.IsZero() {
			return at, ahead
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("a take that was played was never asked for")
	return time.Time{}, 0
}

func TestPlaybackBeginsWithinTheLatencyBudget(t *testing.T) {
	player, err := NewPlayer()
	if err != nil {
		t.Skipf("no audio device: %v", err)
	}
	defer func() { _ = player.Close() }()
	// Heard as silence: the run measures when output begins, not what it says.
	player.SetVolume(0)

	take := filepath.Join(t.TempDir(), "take.wav")
	audiotest.WriteTake(t, take)

	latencies := make([]time.Duration, 0, latencyFirings)
	for range latencyFirings {
		player.mu.Lock()
		player.firstPull = time.Time{}
		player.mu.Unlock()
		called := time.Now()
		if err := player.Play([]string{take}, 0); err != nil {
			t.Fatalf("playing: %v", err)
		}
		at, ahead := awaitFirstPull(t, player)
		latencies = append(latencies, at.Sub(called)+ahead+driverBuffer)
	}
	slices.Sort(latencies)
	at := latencies[latencyFirings*latencyPercentile/100-1]
	t.Logf("latency at the %dth percentile %v, best %v, worst %v",
		latencyPercentile, at, latencies[0], latencies[len(latencies)-1])
	if at > latencyBudget {
		t.Errorf("output begins in %v at the %dth percentile, over the %v budget", at, latencyPercentile, latencyBudget)
	}
}
