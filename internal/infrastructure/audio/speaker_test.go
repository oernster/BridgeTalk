package audio

// NFR-P-202's note, held without a sound card: the speaker runs over a fake of oto's queue that
// counts every time the queue is dropped, while the test itself asks for samples as the device
// would. Stop and a take that interrupts another drop what is queued; a take that starts after
// silence drops the queued silence; a take that follows another closely waits for its end.

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

// feedWait is how long a test asks for samples before giving up on what it waits for.
const feedWait = 2 * time.Second

// fakeQueue stands in for oto's player, counting the drops.
type fakeQueue struct {
	mu    sync.Mutex
	drops int
}

func (q *fakeQueue) Play() {}

func (q *fakeQueue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.drops++
}

func (q *fakeQueue) BufferedSize() int { return 0 }

func (q *fakeQueue) Close() error { return nil }

// dropped reports how many times the queue has been dropped.
func (q *fakeQueue) dropped() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.drops
}

// fedPlayer builds a player over a fake queue with a clock that never moves, so anything that
// sounded did so just now, together with a take it can play.
func fedPlayer(t *testing.T) (*Player, *fakeQueue, string) {
	t.Helper()
	queue := &fakeQueue{}
	now := time.Now()
	player := &Player{
		finished: make(chan struct{}, 1),
		volume:   fullVolume,
		out:      &speaker{player: queue, clock: func() time.Time { return now }},
	}
	take := filepath.Join(t.TempDir(), "take.wav")
	audiotest.WriteTake(t, take)
	t.Cleanup(player.Stop)
	return player, queue, take
}

// holding reports whether the speaker has a take to play.
func holding(player *Player) bool {
	player.out.mu.Lock()
	defer player.out.mu.Unlock()
	return player.out.mixer.Len() > 0
}

// awaitHeld waits for the take just played to reach the speaker.
func awaitHeld(t *testing.T, player *Player) {
	t.Helper()
	deadline := time.Now().Add(feedWait)
	for !holding(player) {
		if time.Now().After(deadline) {
			t.Fatal("a take that was played never reached the speaker")
		}
		time.Sleep(time.Millisecond)
	}
}

// feedToTheEnd asks for samples as the device would until the sequence reports it has ended.
func feedToTheEnd(t *testing.T, player *Player) {
	t.Helper()
	buf := make([]byte, deviceSampleRate.N(playerBuffer)*bytesPerFrame)
	deadline := time.Now().Add(feedWait)
	for time.Now().Before(deadline) {
		if _, err := player.out.Read(buf); err != nil {
			t.Fatalf("reading: %v", err)
		}
		select {
		case <-player.Done():
			return
		default:
			time.Sleep(time.Millisecond)
		}
	}
	t.Fatal("a take that was fed never ended")
}

// The scheduler starts the next take the moment the player reports the last one ended, which
// is while its end is still queued for the device. Dropping the queue then cuts off its last words.
func TestATakeThatFollowsAnotherCloselyWaitsForItsEnd(t *testing.T) {
	t.Parallel()
	player, queue, take := fedPlayer(t)
	if err := player.Play(whole(take), 0); err != nil {
		t.Fatalf("playing: %v", err)
	}
	feedToTheEnd(t, player)
	if player.Playing() {
		t.Fatal("a take that has ended still reports playing")
	}

	before := queue.dropped()
	if err := player.Play(whole(take), 0); err != nil {
		t.Fatalf("playing the next take: %v", err)
	}
	awaitHeld(t, player)
	if dropped := queue.dropped() - before; dropped != 0 {
		t.Errorf("a take that followed another closely dropped the queue %d times; it must wait for the end of the one before it", dropped)
	}
}

// A take that cuts in is heard at once, so what the one it replaced left queued goes.
func TestATakeThatInterruptsAnotherDropsWhatIsQueued(t *testing.T) {
	t.Parallel()
	player, queue, take := fedPlayer(t)
	if err := player.Play(whole(take), 0); err != nil {
		t.Fatalf("playing: %v", err)
	}
	awaitHeld(t, player)

	before := queue.dropped()
	if err := player.Play(whole(take), 0); err != nil {
		t.Fatalf("interrupting: %v", err)
	}
	if queue.dropped() == before {
		t.Error("a take that interrupted another left the queue behind it to play first")
	}
}

// Stop is heard at once: the take goes from the speaker and its audio from the queue.
func TestStopDropsWhatIsQueued(t *testing.T) {
	t.Parallel()
	player, queue, take := fedPlayer(t)
	if err := player.Play(whole(take), 0); err != nil {
		t.Fatalf("playing: %v", err)
	}
	awaitHeld(t, player)

	before := queue.dropped()
	player.Stop()
	if queue.dropped() == before {
		t.Error("Stop left the queue to play out")
	}
	if holding(player) {
		t.Error("Stop left the take in the speaker")
	}
}

// Nothing has sounded on a new speaker, so all it can have queued is silence, which goes.
func TestATakeThatStartsAfterSilenceDropsTheQueuedSilence(t *testing.T) {
	t.Parallel()
	player, queue, take := fedPlayer(t)
	if err := player.Play(whole(take), 0); err != nil {
		t.Fatalf("playing: %v", err)
	}
	awaitHeld(t, player)
	if queue.dropped() == 0 {
		t.Error("a take that started after silence played behind the queued silence")
	}
}
