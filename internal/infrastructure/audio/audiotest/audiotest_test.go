package audiotest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio"
	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

// Every take this package writes is one the player reads as sound. A fixture that stopped
// playing would leave every scan test finding nothing, which would read as the scan broken.
func TestEveryTakeWrittenPlays(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, name := range []string{"take.mp3", "take.wav", "TAKE.MP3"} {
		path := filepath.Join(dir, "nested", name)
		audiotest.WriteTake(t, path)
		if err := audio.Playable(path); err != nil {
			t.Errorf("%s will not play: %v", name, err)
		}
	}
}

// A file that is no recording is written as it was given, directories and all.
func TestAFileIsWrittenAsGiven(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "a", "b", "notes.txt")
	audiotest.WriteFile(t, path, audiotest.NotARecording)
	if held, err := os.ReadFile(path); err != nil || string(held) != string(audiotest.NotARecording) {
		t.Fatalf("read back %q, %v", held, err)
	}
	if err := audio.Playable(path); err == nil {
		t.Fatal("a text file was called playable")
	}
}
