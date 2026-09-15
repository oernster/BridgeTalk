// Package measured holds what the pauses and the endings share: a machine voice's lines measured
// before the build, each tied to the speech sounds and the samples it was measured in, kept voice by
// voice in a book, refused where they cannot be shipped and checked against the script (FR-551,
// FR-554, FR-555, FR-557).
package measured

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// Line is what every entry records of the line it was measured in.
type Line struct {
	// Cue is the cue the line is spoken for.
	Cue cue.ID
	// Index is the line's place among its cue's lines, from zero.
	Index int
	// Sounds is the saved speech sounds the line was made from.
	Sounds string
	// Digest is the digest of the samples the line was measured in.
	Digest string
}

// Entry is one voice's measure of one line: a pause or an ending.
type Entry interface {
	// Measured answers the line the entry was measured in.
	Measured() Line
}

// Voice is one voice's entries with the digest of the style file its lines were made with.
type Voice[E Entry] struct {
	// Style is the digest of the style file the voice's lines were made with.
	Style string
	// Entries is the voice's entries, one for each line measured.
	Entries []E
}

// Kind is what one kind of entry says of itself where a book is refused or found stale: the
// sentinels its problems wrap, the requirements they cite and the words naming its entries.
type Kind struct {
	// Invalid is wrapped by every refusal of a book.
	Invalid error
	// Stale is wrapped by every problem a check finds.
	Stale error
	// Found is the requirement a refusal of a book cites.
	Found string
	// Checked is the requirement a stale problem cites.
	Checked string
	// Plural names the kind's entries together, as in "has no pauses".
	Plural string
	// Missing says a line that should have an entry has none, as in "joins with no pause".
	Missing string
	// MeasuredIn says what an entry was found in, before the sounds it was found in.
	MeasuredIn string
	// Gone says an entry's line should no longer have one.
	Gone string
}

// place names one of a cue's lines.
type place struct {
	cue   cue.ID
	index int
}

// Book is every voice's entries with the digest of the model they were measured with. Only NewBook
// makes one, so every Book holds entries the rules accept.
type Book[E Entry] struct {
	model  string
	voices map[string]Voice[E]
	places map[string]map[place]int
}

// NewBook builds a book from each voice's entries, refusing every problem together, each naming its
// voice, its cue and its line: no speech sounds, no digest, a sample the kind's rule refuses and the
// same line twice in one voice. A line at a negative index is refused for that alone, since it cannot
// be named as the script counts its lines. A book with no voices is the empty book shipped before the
// pauses tool first runs.
func NewBook[E Entry](kind Kind, model string, voices map[string]Voice[E], sample func(name string, entry E) error) (Book[E], error) {
	book := Book[E]{
		model:  model,
		voices: make(map[string]Voice[E], len(voices)),
		places: make(map[string]map[place]int, len(voices)),
	}
	var problems []error
	for _, id := range slices.Sorted(maps.Keys(voices)) {
		given := voices[id]
		places := make(map[place]int, len(given.Entries))
		for at, entry := range given.Entries {
			line := entry.Measured()
			problems = append(problems, lineProblems(kind, id, entry, sample)...)
			key := place{cue: line.Cue, index: line.Index}
			if _, twice := places[key]; twice {
				problems = append(problems, fmt.Errorf(
					"%w: %s is given twice (%s)", kind.Invalid, LineName(id, line.Cue, line.Index), kind.Found,
				))
			}
			places[key] = at
		}
		book.voices[id] = Voice[E]{Style: given.Style, Entries: slices.Clone(given.Entries)}
		book.places[id] = places
	}
	if len(problems) > 0 {
		return Book[E]{}, errors.Join(problems...)
	}
	return book, nil
}

// lineProblems returns every problem with one voice's entry for a line.
func lineProblems[E Entry](kind Kind, voice string, entry E, sample func(name string, entry E) error) []error {
	line := entry.Measured()
	if line.Index < 0 {
		return []error{fmt.Errorf(
			"%w: %s %q has a line at index %d, before its first (%s)", kind.Invalid, voice, line.Cue, line.Index, kind.Found,
		)}
	}
	name := LineName(voice, line.Cue, line.Index)
	var problems []error
	if line.Sounds == "" {
		problems = append(problems, fmt.Errorf("%w: %s has no speech sounds (%s)", kind.Invalid, name, kind.Found))
	}
	if line.Digest == "" {
		problems = append(problems, fmt.Errorf("%w: %s has no digest (%s)", kind.Invalid, name, kind.Found))
	}
	if err := sample(name, entry); err != nil {
		problems = append(problems, err)
	}
	return problems
}

// FirstWhere answers the name of the first line whose entry matches, voices sorted and each voice's
// entries in the order given, reporting whether any does.
func FirstWhere[E Entry](voices map[string]Voice[E], match func(E) bool) (string, bool) {
	for _, id := range slices.Sorted(maps.Keys(voices)) {
		for _, entry := range voices[id].Entries {
			if match(entry) {
				line := entry.Measured()
				return LineName(id, line.Cue, line.Index), true
			}
		}
	}
	return "", false
}

// Model returns the digest of the model file the entries were measured with.
func (b Book[E]) Model() string { return b.model }

// Voices returns the ids of the voices the book gives entries, sorted. The slice is the caller's own.
func (b Book[E]) Voices() []string { return slices.Sorted(maps.Keys(b.voices)) }

// Voice returns one voice's entries in the order they were given, reporting whether the book gives the
// voice any. The entries are the caller's own.
func (b Book[E]) Voice(id string) (Voice[E], bool) {
	voice, ok := b.voices[id]
	return Voice[E]{Style: voice.Style, Entries: slices.Clone(voice.Entries)}, ok
}

// Entry returns a voice's entry for one of a cue's lines, reporting whether the book gives one.
func (b Book[E]) Entry(voice string, id cue.ID, index int) (E, bool) {
	at, ok := b.places[voice][place{cue: id, index: index}]
	if !ok {
		var none E
		return none, false
	}
	return b.voices[voice].Entries[at], true
}
