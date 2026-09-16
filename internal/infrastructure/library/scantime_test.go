package library

// NFR-P-201: a library root holding 10 voices and 5,000 audio files between them is scanned within 3
// seconds. The scan decodes the start of every take (FR-204), so the files here are real recordings
// rather than names alone, which would measure a directory listing. They are WAV and MP3, the two
// formats audiotest can make; an OGG or FLAC take is not timed here.

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
)

const (
	// largestLibraryVoices and largestLibraryFiles are the library NFR-P-201 names.
	largestLibraryVoices = 10
	largestLibraryFiles  = 5000
	// scanBudget is the time NFR-P-201 allows for scanning it.
	scanBudget = 3 * time.Second
	// takesPerCue is how many takes each voice holds for one moment: one named for it at the root
	// of the voice, one in the moment's own folder, which are the two forms a take is found in.
	takesPerCue = 2
)

// formats are the extensions audiotest can make a recording in, taken in turn so each decoder
// carries its share.
var formats = []string{".wav", ".mp3"}

// cueName answers a moment id for index that no other index shares. It is built of letters alone,
// since a cue id ending in a segment of digits would read as a take number (flatTake).
func cueName(index int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return "Moment" + string(letters[index/len(letters)]) + string(letters[index%len(letters)])
}

// TestTheLargestLibraryIsScannedWithinTheBudget lays out the library NFR-P-201 names, then scans it
// and holds the time the scan took to the budget. The laying out is not timed.
func TestTheLargestLibraryIsScannedWithinTheBudget(t *testing.T) {
	root := t.TempDir()
	cuesPerVoice := largestLibraryFiles / largestLibraryVoices / takesPerCue
	ids := make([]string, 0, cuesPerVoice)
	for index := range cuesPerVoice {
		ids = append(ids, cueName(index))
	}
	written := 0
	for voice := range largestLibraryVoices {
		dir := filepath.Join(root, fmt.Sprintf("Voice %d", voice+1))
		for index, id := range ids {
			format := formats[(voice+index)%len(formats)]
			audiotest.WriteTake(t, filepath.Join(dir, id+format))
			audiotest.WriteTake(t, filepath.Join(dir, id, "take"+format))
			written += takesPerCue
		}
	}
	if written != largestLibraryFiles {
		t.Fatalf("laid out %d files, want the %d NFR-P-201 names", written, largestLibraryFiles)
	}
	table := journalTable(t, ids...)

	started := time.Now()
	voices, _ := scanned(t, root, table)
	took := time.Since(started)

	t.Logf("scanned %d voices holding %d files in %v", len(voices), written, took)
	if len(voices) != largestLibraryVoices {
		t.Fatalf("found %d voices, want all %d, so the scan did not read what it was timed over", len(voices), largestLibraryVoices)
	}
	for _, voice := range voices {
		if voice.Takes != len(ids)*takesPerCue {
			t.Fatalf("%s holds %d takes, want %d, so not every file was read", voice.Name, voice.Takes, len(ids)*takesPerCue)
		}
	}
	if took > scanBudget {
		t.Errorf("the scan took %v, over the %v NFR-P-201 allows", took, scanBudget)
	}
}
