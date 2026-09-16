package audio

// The parts a player is handed: one that will not open is recorded and passed over (FR-574), a take
// none of whose parts open plays nothing (FR-575) and a part is only ever read where it stands
// (FR-572). Each is reachable with no sound card, because each answers before the device is.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gopxl/beep/v2"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

// A part of a take that will not open is passed over so the rest of the take still plays; it
// is recorded with the reason (FR-574). The audio belongs to the user and can go at any time, so
// the note is the only thing that would ever say a part is missing: a take short of one still
// sounds like a take.
//
// The read fails before the device is reached, which is what makes this reachable with no sound
// card: the player answers that the sequence carries on without anything having been played.
func TestAPartThatWillNotOpenIsRecordedAndTheTakeCarriesOn(t *testing.T) {
	t.Parallel()
	player := silentPlayer()
	var noted []string
	player.load = func(string) (beep.Streamer, error) { return nil, errors.New("it is not there") }
	player.record = func(line string) { noted = append(noted, line) }

	if !player.playOne(filepath.Join("C:", "Recordings", "part two.wav"), make(chan struct{})) {
		t.Fatal("a part that would not open ended the take, want the rest of it played")
	}

	if len(noted) != 1 {
		t.Fatalf("%d parts were recorded, want one: %v", len(noted), noted)
	}
	if !strings.Contains(noted[0], "part two.wav") || !strings.Contains(noted[0], "it is not there") {
		t.Errorf("the note reads %q, want the part named with the reason", noted[0])
	}
	if !strings.Contains(noted[0], "passed over") {
		t.Errorf("the note reads %q, want it to say the part was passed over", noted[0])
	}
}

// A take none of whose parts will open plays nothing for its cue and records every part it tried,
// in the order it tried them (FR-575). Silence is the answer when the alternative is a wrong line.
// The sequence still ends and says so, since a scheduler that never heard the end of one would wait
// for ever.
//
// Nothing reaches the device: this player has none, so a part handed to it would end the test with
// a nil pointer rather than pass.
func TestATakeWhosePartsWillNotOpenPlaysNothingAndRecordsEachOne(t *testing.T) {
	t.Parallel()
	player := silentPlayer()
	var noted []string
	player.load = func(string) (beep.Streamer, error) { return nil, errors.New("it is not there") }
	player.record = func(line string) { noted = append(noted, line) }
	parts := []string{"one.wav", "two.wav", "three.wav"}
	cancel := make(chan struct{})
	player.playing, player.cancel = true, cancel

	player.run(parts, 0, cancel)

	if len(noted) != len(parts) {
		t.Fatalf("%d parts were recorded, want all %d: %v", len(noted), len(parts), noted)
	}
	for index, part := range parts {
		if !strings.Contains(noted[index], part) {
			t.Errorf("note %d reads %q, want %s named", index+1, noted[index], part)
		}
	}
	if !waitForFinish(t, player) || player.Playing() {
		t.Error("a take that played nothing did not end, so its cue would never be let go")
	}
}

// A part is read where it stands and never opened for writing, so one its owner has marked
// read-only plays the same as any other and is left exactly as it was (FR-572). A plugin's audio is
// more likely than a recording to belong to somebody else.
func TestAPartMarkedReadOnlyIsReadAndLeftAsItWas(t *testing.T) {
	t.Parallel()
	const frames = 256
	path := filepath.Join(t.TempDir(), "docked.wav")
	body := audiotest.WAV(t, frames)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("writing the part: %v", err)
	}
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatalf("marking the part read-only: %v", err)
	}
	// The temporary folder cannot be emptied around a read-only file on Windows.
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("reading the part's state: %v", err)
	}

	if _, err := load(path); err != nil {
		t.Fatalf("a part marked read-only would not load: %v", err)
	}

	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("reading the part's state again: %v", err)
	}
	if held, err := os.ReadFile(path); err != nil || string(held) != string(body) {
		t.Errorf("the part changed on being read: %v", err)
	}
	if !after.ModTime().Equal(before.ModTime()) || after.Mode() != before.Mode() {
		t.Errorf("the part was %v %v and is now %v %v", before.Mode(), before.ModTime(), after.Mode(), after.ModTime())
	}
}
