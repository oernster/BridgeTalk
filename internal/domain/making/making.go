// Package making works out what a machine voice still has to make: the key each line's made line
// is stored under, which lines are current and how far making has got.
package making

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/pause"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// keySeparator keeps the parts of a key apart, so a symbol moved from one part to the next still
// changes the key.
const keySeparator = "\x00"

// fadeMark goes before a fade's parts of a key, so a pause and a fade at the same sample found in the
// same samples are keyed apart (FR-513).
const fadeMark = "fade"

// Files are the digests of what a voice's made lines are made from besides their speech sounds:
// the voice's style file and the model (FR-513).
type Files struct {
	Style string
	Model string
}

// Line is one line a voice speaks: its cue, its place among that cue's lines, its speech sounds in
// the voice's accent, the pause and the ending the voice's books give it and the key its made line is
// stored under.
type Line struct {
	Cue    cue.ID
	Index  int
	Sounds string
	// Pause is the voice's pause for the line (FR-553); it means nothing unless PauseGiven.
	Pause pause.Entry
	// PauseGiven reports whether the book gives the voice a pause for the line, doubtful or not.
	PauseGiven bool
	// Ending is the voice's ending for the line (FR-556); it means nothing unless EndingGiven.
	Ending ending.Entry
	// EndingGiven reports whether the book gives the voice an ending for the line, fading or not.
	EndingGiven bool
	Key         string
}

// Paused reports whether the line's made line gets its pause: the book gives one that is not
// doubtful (FR-552, FR-553).
func (l Line) Paused() bool { return l.PauseGiven && !l.Pause.Doubtful }

// Faded reports whether the line's made line is faded: the book gives it an ending that fades
// (FR-556).
func (l Line) Faded() bool { return l.EndingGiven && l.Ending.Fades() }

// Key is the name a made line is stored under. It changes whenever the line's speech sounds, the
// style file or the model does, so an old rendering is never taken for current (FR-513). Two
// lines with the same sounds share a key, which is right: they sound the same.
func Key(sounds string, files Files) string { return keyOf(sounds, files.Style, files.Model) }

// PausedKey is the key of a line the book gives a pause: Key's parts followed by the pause's sample,
// the digest it was found in and how many samples of silence the book inserts, so a changed pause is
// made again (FR-513). A doubtful pause adds nothing, answering Key.
func PausedKey(sounds string, files Files, entry pause.Entry, silence int) string {
	return lineKey(Line{Sounds: sounds, Pause: entry, PauseGiven: true}, files, lengths{silence: silence})
}

// lengths are how many samples the books' changes to a line last: the pause's silence and the fade.
type lengths struct {
	silence int
	fade    int
}

// lineKey is the key of a line with what its books give it: Key's parts, then the pause's sample,
// digest and silence where it is paused, then fadeMark with the fade's sample, digest and length where
// it is faded. A line neither paused nor faded answers Key, so no made line keyed before pauses and
// endings goes stale (FR-513).
func lineKey(line Line, files Files, given lengths) string {
	parts := []string{line.Sounds, files.Style, files.Model}
	if line.Paused() {
		parts = append(parts, strconv.Itoa(line.Pause.Sample), line.Pause.Digest, strconv.Itoa(given.silence))
	}
	if line.Faded() {
		parts = append(parts, fadeMark, strconv.Itoa(line.Ending.Sample), line.Ending.Digest, strconv.Itoa(given.fade))
	}
	return keyOf(parts...)
}

// keyOf is the SHA-256 of a key's parts kept apart by keySeparator, written as hexadecimal.
func keyOf(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, keySeparator)))
	return hex.EncodeToString(sum[:])
}

// Plan is a voice's lines set against the keys of the made lines already on disk.
type Plan struct {
	lines   []Line
	current map[string]bool
	onDisk  []string
}

// New sets every line of the script, in the voice's accent with the pauses and the endings the books
// give the voice, against the keys already on disk.
func New(voiced script.Voiced, voice machinevoice.Voice, files Files, pauses pause.Book, endings ending.Book, onDisk []string) Plan {
	current := make(map[string]bool, len(onDisk))
	for _, key := range onDisk {
		current[key] = true
	}
	given := lengths{silence: pauses.Silence(), fade: endings.Fade()}
	var lines []Line
	for _, id := range voiced.Cues() {
		sounds, _ := voiced.Sounds(id, voice.Accent())
		for index, each := range sounds {
			line := Line{Cue: id, Index: index, Sounds: each}
			line.Pause, line.PauseGiven = pauses.Entry(voice.ID(), id, index)
			line.Ending, line.EndingGiven = endings.Entry(voice.ID(), id, index)
			line.Key = lineKey(line, files, given)
			lines = append(lines, line)
		}
	}
	return Plan{lines: lines, current: current, onDisk: slices.Clone(onDisk)}
}

// ToMake returns the lines with no current made line, cue by cue in the plan's order (FR-511, FR-512).
func (p Plan) ToMake() []Line {
	var toMake []Line
	for _, line := range p.lines {
		if !p.current[line.Key] {
			toMake = append(toMake, line)
		}
	}
	return toMake
}

// Unmade returns a cue's lines with no current made line in line order: what is made next when the cue
// fires with none (FR-514).
func (p Plan) Unmade(id cue.ID) []Line {
	var unmade []Line
	for _, line := range p.lines {
		if line.Cue == id && !p.current[line.Key] {
			unmade = append(unmade, line)
		}
	}
	return unmade
}

// Stale returns the keys on disk that no line holds, in the order they were given: made lines no
// longer current, which casting the voice deletes (FR-513, FR-527).
func (p Plan) Stale() []string {
	held := make(map[string]bool, len(p.lines))
	for _, line := range p.lines {
		held[line.Key] = true
	}
	var stale []string
	for _, key := range p.onDisk {
		if !held[key] {
			stale = append(stale, key)
		}
	}
	return stale
}

// Total returns how many lines the voice speaks (FR-515).
func (p Plan) Total() int { return len(p.lines) }

// Current returns how many of those lines have a current made line (FR-515).
func (p Plan) Current() int { return p.Total() - len(p.ToMake()) }

// WithMade returns the plan with the made line under key current, leaving this plan unchanged.
// Making asks for it as each line is written, so what is counted and answered rises as making goes
// on (FR-514, FR-515).
func (p Plan) WithMade(key string) Plan {
	current := make(map[string]bool, len(p.current)+1)
	maps.Copy(current, p.current)
	current[key] = true
	return Plan{lines: p.lines, current: current, onDisk: p.onDisk}
}

// Group returns the lines of every cue in a group, made or not, in the plan's order: what a machine
// voice is auditioned on for that group (FR-546).
func (p Plan) Group(key string) []Line {
	var lines []Line
	for _, line := range p.lines {
		if line.Cue.Group() == key {
			lines = append(lines, line)
		}
	}
	return lines
}

// Made reports whether the made line under key is current.
func (p Plan) Made(key string) bool { return p.current[key] }

// Takes returns the keys of a cue's current made lines in line order, each once: two lines with the
// same sounds share one made line (FR-514).
func (p Plan) Takes(id cue.ID) []string {
	var keys []string
	for _, line := range p.lines {
		if line.Cue == id && p.current[line.Key] && !slices.Contains(keys, line.Key) {
			keys = append(keys, line.Key)
		}
	}
	return keys
}

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
