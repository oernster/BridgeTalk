package audio

// A recording whose header lies is foreign input: a crashed recorder, a bad copy or a plugin's span
// can leave one. Whatever it states, the scan answers why it will not play and the player passes it
// over; nothing it states may end the run.

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/generators"

	"github.com/oernster/bridge-talk/internal/domain/take"
	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

// someFrames is enough samples behind a header for the decoder to have something to read.
const someFrames = 64

// samples answers bytes of sound behind a header: someFrames of them at the size its fields state.
func samples(header audiotest.Header) []byte {
	return make([]byte, someFrames*int(header.Channels)*int(header.BitsPerSample)/bitsPerByte)
}

// bitsPerByte turns a header's bits per sample into bytes.
const bitsPerByte = 8

// lying writes a WAV take stating header into a temporary folder and answers its path.
func lying(t *testing.T, name string, header audiotest.Header) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	audiotest.WriteFile(t, path, audiotest.WAVOf(t, header, samples(header)))
	return path
}

// B-1: a block align of 0, 1 or below 0 made the decoder index out of range inside the scan's own
// probe, which ended the process before the window opened, on every start. It is now the reason the
// take will not play.
func TestATakeWhoseBlockAlignLiesIsRefusedRatherThanEndingTheRun(t *testing.T) {
	t.Parallel()
	for _, blockAlign := range []int16{0, 1, -4} {
		header := audiotest.Standard()
		header.BlockAlign = blockAlign
		path := lying(t, "DockingDenied.wav", header)

		err := Playable(path)
		if !errors.Is(err, ErrDamaged) {
			t.Errorf("block align %d: got %v, want %v", blockAlign, err, ErrDamaged)
		}
		if _, err := load(take.File(path)); err == nil {
			t.Errorf("block align %d: the player loaded it, want it refused", blockAlign)
		}
	}
}

// B-3: a sample rate of 0 or below streamed its probe without complaint, so the scan offered it;
// the player's resampling then panicked on the ratio, on a goroutine with no recover, mid-game. It
// is refused as the part is opened, so the scan reports it and the player passes it over.
func TestATakeStatingNoSampleRateIsRefused(t *testing.T) {
	t.Parallel()
	for _, rate := range []int32{0, -44100} {
		header := audiotest.Standard()
		header.SampleRate = rate
		path := lying(t, "DockingDenied.wav", header)

		if err := Playable(path); !errors.Is(err, ErrNoSampleRate) {
			t.Errorf("rate %d: Playable answered %v, want %v", rate, err, ErrNoSampleRate)
		}
		if _, err := load(take.File(path)); !errors.Is(err, ErrNoSampleRate) {
			t.Errorf("rate %d: load answered %v, want %v", rate, err, ErrNoSampleRate)
		}
	}
}

// B-3: whatever a load meets, a fault inside it costs that part alone. The part is passed over with
// the fault as its reason and the take carries on.
func TestAPartWhoseLoadFaultsIsPassedOverAndTheTakeCarriesOn(t *testing.T) {
	t.Parallel()
	player := silentPlayer()
	var noted []string
	player.load = func(take.Part) (beep.Streamer, error) { panic("resample: invalid ratio: 0.000000") }
	player.record = func(line string) { noted = append(noted, line) }

	if !player.playOne(take.File("faulty.wav"), make(chan struct{})) {
		t.Fatal("a part whose load faulted ended the take, want the rest of it played")
	}
	if len(noted) != 1 || !strings.Contains(noted[0], "faulty.wav") || !strings.Contains(noted[0], "invalid ratio") {
		t.Errorf("noted %v, want the part named with the fault as its reason", noted)
	}
}

// A decoder that faulted reads nothing more and keeps answering the fault; seeking goes through the
// same guard, so a seek cannot end the run either.
func TestADecoderThatFaultedStaysEndedAndSeeksSafely(t *testing.T) {
	t.Parallel()
	header := audiotest.Standard()
	header.BlockAlign = 1
	streamer, _, closer, err := open(take.File(lying(t, "DockingDenied.wav", header)))
	if err != nil {
		t.Fatalf("opening: %v", err)
	}
	defer func() { _ = closer() }()
	room := make([][2]float64, someFrames)

	for range 2 {
		if filled, ok := streamer.Stream(room); filled != 0 || ok {
			t.Errorf("a faulted decoder read %d samples (%v), want none", filled, ok)
		}
	}
	if !errors.Is(streamer.Err(), ErrDamaged) {
		t.Errorf("it answered %v, want %v", streamer.Err(), ErrDamaged)
	}
	if err := streamer.Seek(0); err != nil {
		t.Errorf("seeking to the start answered %v, want it allowed", err)
	}
}

// B-6: a part's length is foreign input. One that would decode to more than a part is ever held for
// is refused from its header, before any of it is decoded; the scan reports it rather than offering
// it. 8 bit mono at a low rate expands many times over when it is brought up to the device's.
func TestATakeLongerThanAnyPartIsHeldForIsRefusedFromItsHeader(t *testing.T) {
	t.Parallel()
	const lowRate = 4000
	header := audiotest.Header{SampleRate: lowRate, Channels: 1, BitsPerSample: bitsPerByte, BlockAlign: 1}
	frames := int(beep.SampleRate(lowRate).N(maxPartDuration)) + 1
	path := filepath.Join(t.TempDir(), "Endless.wav")
	audiotest.WriteFile(t, path, audiotest.WAVOf(t, header, make([]byte, frames)))

	if err := Playable(path); !errors.Is(err, ErrTooLong) {
		t.Errorf("Playable answered %v, want %v", err, ErrTooLong)
	}
	started := time.Now()
	if _, err := load(take.File(path)); !errors.Is(err, ErrTooLong) {
		t.Errorf("load answered %v, want %v", err, ErrTooLong)
	}
	if spent := time.Since(started); spent > time.Second {
		t.Errorf("refusing it took %v, want it refused before decoding", spent)
	}
}

// B-6: a format whose header states no length is held to the same ceiling while it is read, so a
// part that runs on cannot fill memory however it describes itself.
func TestAPartThatRunsOnIsCutOffAtTheCeiling(t *testing.T) {
	t.Parallel()
	if _, err := readWhole(generators.Silence(-1)); !errors.Is(err, ErrTooLong) {
		t.Errorf("an endless part answered %v, want %v", err, ErrTooLong)
	}
	held, err := readWhole(generators.Silence(someFrames))
	if err != nil || held.Len() != someFrames {
		t.Errorf("a short part answered %v holding %d frames, want all %d", err, held.Len(), someFrames)
	}
}
