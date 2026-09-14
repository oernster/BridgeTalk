package library

// FR-215's second figure: the files a recorded voice uses against the recordings present in
// its directory. Both belong to a voice held on disk, which is why they live on Voice rather
// than on the catalogue: an audio source answers takes and nothing else (FR-502).

import (
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/infrastructure/audio"
)

// Files reports the distinct files this voice uses against the recordings present in its
// directory (FR-215).
//
// The two numbers are expected to be equal, since a voice is recorded against the
// vocabulary and every file it holds should answer a cue. They are reported beside
// Coverage because they fail differently: a shortfall in Coverage means lines were never
// recorded, while a gap here means files are present that nothing can reach, which is a
// naming mistake rather than a missing performance.
func (v Voice) Files() (used int, present int) {
	seen := map[string]struct{}{}
	for _, clips := range v.byCue {
		for _, clip := range clips {
			seen[clip] = struct{}{}
		}
	}
	return len(seen), v.Present
}

// recordingsUnder counts the files with a recognised extension under a directory at any
// depth. It is the second figure's measure of what is there (FR-215), so it counts what the
// scan passed over as well as what it took: a recording nothing reaches is the shortfall
// that figure exists to show.
func (from disk) recordingsUnder(dir string) int {
	entries, err := from.read(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count += from.recordingsUnder(filepath.Join(dir, entry.Name()))
			continue
		}
		if audio.Recognised(entry.Name()) {
			count++
		}
	}
	return count
}
