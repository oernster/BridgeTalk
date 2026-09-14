// Package making works out what a machine voice still has to make: the key each line's made line
// is stored under, which lines are current and how far making has got.
package making

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// keySeparator keeps the parts of a key apart, so a symbol moved from one part to the next still
// changes the key.
const keySeparator = "\x00"

// Files are the digests of what a voice's made lines are made from besides their speech sounds:
// the voice's style file and the model (FR-513).
type Files struct {
	Style string
	Model string
}

// Line is one line a voice speaks: its cue, its place among that cue's lines, its speech sounds
// in the voice's accent and the key its made line is stored under.
type Line struct {
	Cue    cue.ID
	Index  int
	Sounds string
	Key    string
}

// Key is the name a made line is stored under. It changes whenever the line's speech sounds, the
// style file or the model does, so an old rendering is never taken for current (FR-513). Two
// lines with the same sounds share a key, which is right: they sound the same.
func Key(sounds string, files Files) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{sounds, files.Style, files.Model}, keySeparator)))
	return hex.EncodeToString(sum[:])
}

// Plan is a voice's lines set against the keys of the made lines already on disk.
type Plan struct {
	lines   []Line
	current map[string]bool
}

// New sets every line of the script, in the voice's accent, against the keys already on disk.
func New(voiced script.Voiced, accent machinevoice.Accent, files Files, onDisk []string) Plan {
	current := make(map[string]bool, len(onDisk))
	for _, key := range onDisk {
		current[key] = true
	}
	var lines []Line
	for _, id := range voiced.Cues() {
		sounds, _ := voiced.Sounds(id, accent)
		for index, each := range sounds {
			lines = append(lines, Line{Cue: id, Index: index, Sounds: each, Key: Key(each, files)})
		}
	}
	return Plan{lines: lines, current: current}
}

// ToMake returns the lines with no current made line, cue by cue in order (FR-511, FR-512).
func (p Plan) ToMake() []Line {
	var toMake []Line
	for _, line := range p.lines {
		if !p.current[line.Key] {
			toMake = append(toMake, line)
		}
	}
	return toMake
}

// Total returns how many lines the voice speaks (FR-515).
func (p Plan) Total() int { return len(p.lines) }

// Current returns how many of those lines have a current made line (FR-515).
func (p Plan) Current() int { return p.Total() - len(p.ToMake()) }

// CuesServed returns how many cues have at least one current made line (FR-522).
func (p Plan) CuesServed() int {
	served := make(map[cue.ID]bool)
	for _, line := range p.lines {
		if p.current[line.Key] {
			served[line.Cue] = true
		}
	}
	return len(served)
}
