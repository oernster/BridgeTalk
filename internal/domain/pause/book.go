package pause

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// ErrInvalidBook is returned for pauses that cannot be shipped as they stand (FR-551).
var ErrInvalidBook = errors.New("invalid pauses")

// firstSample is the earliest sample a pause may go at: at sample zero its silence would come
// before the line rather than inside it (FR-551).
const firstSample = 1

// Entry is one joined line's pause for one voice (FR-551).
type Entry struct {
	// Cue is the cue the line is spoken for.
	Cue cue.ID
	// Index is the line's place among its cue's lines, from zero.
	Index int
	// Sounds is the saved speech sounds the line was made from.
	Sounds string
	// Digest is the Digest of the samples the break was found in.
	Digest string
	// Sample is the sample the pause goes at; it means nothing when Doubtful.
	Sample int
	// Doubtful says the break was doubtful, so the line gets no pause (FR-552).
	Doubtful bool
}

// Voice is one voice's pauses with the digest of the style file its lines were made with.
type Voice struct {
	// Style is the digest of the style file the voice's lines were made with (FR-551).
	Style string
	// Entries is the voice's pauses, one for each line the script joins.
	Entries []Entry
}

// place names one of a cue's lines.
type place struct {
	cue   cue.ID
	index int
}

// Book is every voice's pauses with the silence a pause inserts and the digest of the model they
// were found with. Only NewBook makes one, so every Book holds pauses the rules accept.
type Book struct {
	silence int
	model   string
	voices  map[string]Voice
	places  map[string]map[place]int
}

// NewBook builds a book from each voice's pauses, refusing every problem together, each naming its
// voice, its cue and its line (FR-551). A book with no voices and no silence is the empty book
// shipped before the pauses tool first runs.
func NewBook(silence int, model string, voices map[string]Voice) (Book, error) {
	book := Book{
		silence: silence,
		model:   model,
		voices:  make(map[string]Voice, len(voices)),
		places:  make(map[string]map[place]int, len(voices)),
	}
	var problems []error
	if err := silenceProblem(silence, voices); err != nil {
		problems = append(problems, err)
	}
	for _, id := range slices.Sorted(maps.Keys(voices)) {
		given := voices[id]
		places := make(map[place]int, len(given.Entries))
		for at, entry := range given.Entries {
			problems = append(problems, entryProblems(id, entry)...)
			key := place{cue: entry.Cue, index: entry.Index}
			if _, twice := places[key]; twice {
				problems = append(problems, fmt.Errorf(
					"%w: %s is given twice (FR-551)", ErrInvalidBook, LineName(id, entry.Cue, entry.Index),
				))
			}
			places[key] = at
		}
		book.voices[id] = Voice{Style: given.Style, Entries: slices.Clone(given.Entries)}
		book.places[id] = places
	}
	if len(problems) > 0 {
		return Book{}, errors.Join(problems...)
	}
	return book, nil
}

// silenceProblem returns the problem with a book's silence where there is one: none to insert while
// a line has a pause, naming the first such line (FR-553).
func silenceProblem(silence int, voices map[string]Voice) error {
	if silence > 0 {
		return nil
	}
	for _, id := range slices.Sorted(maps.Keys(voices)) {
		for _, entry := range voices[id].Entries {
			if !entry.Doubtful {
				return fmt.Errorf(
					"%w: %d samples of silence cannot be inserted while %s has a pause (FR-553)",
					ErrInvalidBook, silence, LineName(id, entry.Cue, entry.Index),
				)
			}
		}
	}
	return nil
}

// entryProblems returns every problem with one voice's pause for a line. A line at a negative index
// is refused for that alone, since it cannot be named as the script counts its lines.
func entryProblems(voice string, entry Entry) []error {
	if entry.Index < 0 {
		return []error{fmt.Errorf(
			"%w: %s %q has a line at index %d, before its first (FR-551)", ErrInvalidBook, voice, entry.Cue, entry.Index,
		)}
	}
	name := LineName(voice, entry.Cue, entry.Index)
	var problems []error
	if entry.Sounds == "" {
		problems = append(problems, fmt.Errorf("%w: %s has no speech sounds (FR-551)", ErrInvalidBook, name))
	}
	if entry.Digest == "" {
		problems = append(problems, fmt.Errorf("%w: %s has no digest (FR-551)", ErrInvalidBook, name))
	}
	if !entry.Doubtful && entry.Sample < firstSample {
		problems = append(problems, fmt.Errorf(
			"%w: %s has its pause at sample %d, before sample %d (FR-551)", ErrInvalidBook, name, entry.Sample, firstSample,
		))
	}
	return problems
}

// Silence returns how many samples of silence a pause inserts (FR-553).
func (b Book) Silence() int { return b.silence }

// Model returns the digest of the model file the pauses were found with (FR-551).
func (b Book) Model() string { return b.model }

// Voices returns the ids of the voices the book gives pauses, sorted. The slice is the caller's own.
func (b Book) Voices() []string { return slices.Sorted(maps.Keys(b.voices)) }

// Voice returns one voice's pauses in the order they were given, reporting whether the book gives
// the voice any. The entries are the caller's own.
func (b Book) Voice(id string) (Voice, bool) {
	voice, ok := b.voices[id]
	return Voice{Style: voice.Style, Entries: slices.Clone(voice.Entries)}, ok
}

// Entry returns a voice's pause for one of a cue's lines, reporting whether the book gives one.
func (b Book) Entry(voice string, id cue.ID, index int) (Entry, bool) {
	at, ok := b.places[voice][place{cue: id, index: index}]
	if !ok {
		return Entry{}, false
	}
	return b.voices[voice].Entries[at], true
}
