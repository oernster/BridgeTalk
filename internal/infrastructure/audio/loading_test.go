package audio

// A clip is read whole before any of it reaches the speaker, so a take can be stopped or
// replaced while it is still being read. Play and Stop clear the speaker at that moment; a
// read that finishes afterwards must not add the cancelled take behind the clear, where it
// would play over or after what replaced it (NFR-P-202).

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gopxl/beep/v2"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

// holdRead makes the player's read of held wait part way until release is closed, closing
// reading as the read reaches that point. Every other clip is read as usual.
func holdRead(player *Player, held string) (reading, release chan struct{}) {
	reading, release = make(chan struct{}), make(chan struct{})
	var once sync.Once
	player.load = func(path string) (beep.Streamer, error) {
		if path == held {
			once.Do(func() { close(reading) })
			<-release
		}
		return load(path)
	}
	return reading, release
}

// within waits for signal, failing the test when it has not come by feedWait.
func within(t *testing.T, signal <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(feedWait):
		t.Fatalf("%s never happened", what)
	}
}

// heldTakes reports how many takes the speaker holds.
func heldTakes(player *Player) int {
	player.out.mu.Lock()
	defer player.out.mu.Unlock()
	return player.out.mixer.Len()
}

func TestATakeStoppedWhileItsClipIsReadNeverReachesTheSpeaker(t *testing.T) {
	t.Parallel()
	player, _, take := fedPlayer(t)
	reading, release := holdRead(player, take)

	if err := player.Play([]string{take}, 0); err != nil {
		t.Fatalf("playing: %v", err)
	}
	within(t, reading, "the read of the take")
	player.Stop()
	close(release)
	within(t, player.Done(), "the stopped sequence ending")

	if held := heldTakes(player); held != 0 {
		t.Errorf("the speaker holds %d takes after Stop, want none: the take read after the stop was added", held)
	}
}

func TestATakeReplacedWhileItsClipIsReadNeverReachesTheSpeaker(t *testing.T) {
	t.Parallel()
	player, _, take := fedPlayer(t)
	slow := filepath.Join(t.TempDir(), "slow.wav")
	audiotest.WriteTake(t, slow)
	reading, release := holdRead(player, slow)

	if err := player.Play([]string{slow}, 0); err != nil {
		t.Fatalf("playing: %v", err)
	}
	within(t, reading, "the read of the first take")
	if err := player.Play([]string{take}, 0); err != nil {
		t.Fatalf("replacing: %v", err)
	}
	awaitHeld(t, player)
	close(release)
	within(t, player.Done(), "the replaced sequence ending")

	if held := heldTakes(player); held != 1 {
		t.Errorf("the speaker holds %d takes, want only the one that replaced the take being read", held)
	}
}
