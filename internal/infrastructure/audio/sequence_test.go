package audio

// The player's sequencing is tested through a SILENT player: one built as though the
// output device could not be opened. That is the same object with the same state
// machine, minus the calls into the speaker, so every decision it makes is reachable
// while nothing here needs a sound card.
//
// What is not reachable that way is the part that actually sounds: playOne and the
// loop around it call into the speaker package and cannot be exercised without a real
// device and a real clip to hear. TESTING.md names that as the gap.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// silentPlayer builds a player as though the device had refused to open.
func silentPlayer() *Player {
	return &Player{finished: make(chan struct{}, 1), silent: true, volume: fullVolume}
}

// waitForFinish reports whether the player signalled the end of a sequence.
func waitForFinish(t *testing.T, player *Player) bool {
	t.Helper()
	select {
	case <-player.Done():
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

// wavBytes builds the smallest real WAV file: a header the decoder accepts and a few
// samples behind it. Writing one is cheaper than shipping a binary fixture and it
// keeps the format the decoder is asked to read visible in the test.
func wavBytes(t *testing.T, frames int) []byte {
	t.Helper()
	const (
		channels      = 2
		bitsPerSample = 16
		sampleRate    = 44100
	)
	dataSize := frames * channels * bitsPerSample / 8

	var out bytes.Buffer
	write := func(values ...any) {
		for _, value := range values {
			if err := binary.Write(&out, binary.LittleEndian, value); err != nil {
				t.Fatalf("building the wav: %v", err)
			}
		}
	}
	out.WriteString("RIFF")
	write(uint32(36 + dataSize))
	out.WriteString("WAVE")
	out.WriteString("fmt ")
	write(uint32(16), uint16(1), uint16(channels), uint32(sampleRate))
	write(uint32(sampleRate*channels*bitsPerSample/8), uint16(channels*bitsPerSample/8))
	write(uint16(bitsPerSample))
	out.WriteString("data")
	write(uint32(dataSize))
	for index := 0; index < frames*channels; index++ {
		write(int16(index))
	}
	return out.Bytes()
}

// A sequence with nothing in it is a caller mistake rather than a silent success;
// a cue bound to an empty folder would otherwise look as though it had spoken.
func TestPlayingNothingIsRefused(t *testing.T) {
	t.Parallel()
	if err := silentPlayer().Play(nil, 0); !errors.Is(err, ErrNoClips) {
		t.Fatalf("got %v, want %v", err, ErrNoClips)
	}
}

// With no device the sequence completes at once and still signals, because the caller
// is counting sequences: a scheduler that never heard the end of one would wait for
// ever and the application would go quiet on a machine with no sound card.
func TestASilentPlayerCompletesTheSequenceImmediately(t *testing.T) {
	player := silentPlayer()

	if !player.Silent() {
		t.Fatal("a player built with no device does not report itself silent")
	}
	if err := player.Play([]string{"anything.mp3"}, 0); err != nil {
		t.Fatalf("playing: %v", err)
	}
	if !waitForFinish(t, player) {
		t.Fatal("a silent sequence never signalled that it had finished")
	}
	if player.Playing() {
		t.Fatal("a silent sequence is still reported as playing")
	}
}

// Stopping is safe with nothing playing, which is the state the application is in for
// most of a session; so is closing a player that never had a device.
func TestStoppingAndClosingASilentPlayerAreSafe(t *testing.T) {
	player := silentPlayer()

	player.Stop()
	if player.Playing() {
		t.Fatal("stopping an idle player left it playing")
	}
	if err := player.Close(); err != nil {
		t.Fatalf("closing: %v", err)
	}
}

// A gap is honoured; a cancel during one cuts it short rather than making the
// caller wait out a pause nobody is going to hear the end of.
func TestAGapIsWaitedOutUnlessItIsCancelled(t *testing.T) {
	t.Parallel()
	cancel := make(chan struct{})
	if !sleepOrCancel(time.Millisecond, cancel) {
		t.Fatal("an uncancelled gap reported as cancelled")
	}

	close(cancel)
	started := time.Now()
	if sleepOrCancel(time.Hour, cancel) {
		t.Fatal("a cancelled gap was waited out")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("a cancelled gap took %v to return", elapsed)
	}
}

func TestAClipThatIsNotThereCannotBeDecoded(t *testing.T) {
	t.Parallel()
	_, _, closer, err := decode(filepath.Join(t.TempDir(), "absent.mp3"))
	if err == nil {
		t.Fatal("a clip that is not there decoded")
	}
	if closer == nil {
		t.Fatal("no closer was returned; the caller defers it either way")
	}
	if err := closer(); err != nil {
		t.Fatalf("the closer for a failed decode returned %v", err)
	}
}

// The extension chooses the decoder, so a file the decoders do not recognise is named
// as unsupported rather than reported as a corrupt clip.
func TestAFormatTheDecodersDoNotKnowIsNamedAsUnsupported(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("this is not audio"), 0o644); err != nil {
		t.Fatalf("planting: %v", err)
	}

	if _, _, _, err := decode(path); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("got %v, want %v", err, ErrUnsupportedFormat)
	}
}

// A file carrying an audio extension but not the audio behind it is a real case: a
// voice can hold a placeholder or a truncated download. It fails as a decode rather
// than as an unsupported format, since the extension was recognised.
func TestAFileWithTheRightNameAndTheWrongContentsFailsToDecode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, name := range []string{"a.mp3", "a.wav", "a.flac", "a.ogg"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("not audio at all"), 0o644); err != nil {
			t.Fatalf("planting %q: %v", name, err)
		}
		_, _, _, err := decode(path)
		if err == nil {
			t.Fatalf("%q decoded although it holds no audio", name)
		}
		if errors.Is(err, ErrUnsupportedFormat) {
			t.Fatalf("%q was called unsupported; its extension is one of the four", name)
		}
	}
}

// A real clip decodes to a stream and a format the player can hand to the device.
func TestARealClipDecodesToAStreamAndItsFormat(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "clip.wav")
	if err := os.WriteFile(path, wavBytes(t, 64), 0o644); err != nil {
		t.Fatalf("planting: %v", err)
	}

	streamer, format, closer, err := decode(path)
	if err != nil {
		t.Fatalf("decoding: %v", err)
	}
	defer func() { _ = closer() }()
	if streamer == nil {
		t.Fatal("no streamer was returned")
	}
	if format.SampleRate == 0 || format.NumChannels == 0 {
		t.Fatalf("format = %+v, want the rate and channels the file declares", format)
	}
}

// A device that cannot be opened is not fatal: the application still watches the
// journal and still reports what it would have said, which is far more useful than
// refusing to start on a machine with no sound. Whether this machine has one is not
// the test's business; that a player comes back either way is.
func TestAPlayerIsReturnedWhetherOrNotThereIsADevice(t *testing.T) {
	player, err := NewPlayer()
	if player == nil {
		t.Fatal("no player was returned")
	}
	defer func() { _ = player.Close() }()

	if err != nil && !player.Silent() {
		t.Fatal("the device failed to open and the player does not report itself silent")
	}
	if err == nil && player.Silent() {
		t.Fatal("the device opened and the player reports itself silent")
	}
	if player.Volume() != fullVolume {
		t.Fatalf("volume = %v, want a new player at full level", player.Volume())
	}
}

// The clip is read whole before the device is given anything.
//
// This is the point of load: the decoding and any resampling happen on the player's
// own goroutine, in the gap before the part is due, rather than inside the device's
// request for its next buffer where being slow is heard rather than merely slow. What
// comes back must therefore be a plain buffer holding every sample, not a decoder
// that will read the file later.
func TestAClipIsReadWholeBeforeItReachesTheDevice(t *testing.T) {
	t.Parallel()
	const frames = 512
	dir := t.TempDir()
	path := filepath.Join(dir, "line.wav")
	if err := os.WriteFile(path, wavBytes(t, frames), 0o644); err != nil {
		t.Fatalf("writing the clip: %v", err)
	}

	source, err := load(path)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	// The file is gone. A streamer that still had reading to do would fail here,
	// which is exactly the reading that must not happen on the device's thread.
	if err := os.Remove(path); err != nil {
		t.Fatalf("removing the clip: %v", err)
	}

	total := 0
	chunk := make([][2]float64, 64)
	for {
		filled, ok := source.Stream(chunk)
		total += filled
		if !ok {
			break
		}
	}
	if total != frames {
		t.Fatalf("streamed %d frames, want %d", total, frames)
	}
	if err := source.Err(); err != nil {
		t.Fatalf("Err returned %v, want nil", err)
	}
}

// A file the decoders cannot read is reported rather than returned as an empty
// buffer, so one unreadable clip is skipped by the caller instead of playing as a
// moment of silence nobody can account for.
func TestAClipThatCannotBeReadIsReported(t *testing.T) {
	t.Parallel()
	if _, err := load(filepath.Join(t.TempDir(), "absent.mp3")); err == nil {
		t.Fatal("a clip that is not there loaded without error")
	}

	path := filepath.Join(t.TempDir(), "line.txt")
	if err := os.WriteFile(path, []byte("not audio"), 0o644); err != nil {
		t.Fatalf("writing: %v", err)
	}
	if _, err := load(path); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("got %v, want %v", err, ErrUnsupportedFormat)
	}
}
