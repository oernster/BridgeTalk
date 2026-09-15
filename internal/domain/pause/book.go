package pause

import (
	"errors"
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/measured"
)

// ErrInvalidBook is returned for pauses that cannot be shipped as they stand (FR-551).
var ErrInvalidBook = errors.New("invalid pauses")

// firstSample is the earliest sample a pause may go at: at sample zero its silence would come
// before the line rather than inside it (FR-551).
const firstSample = 1

// kind is what a pause says of itself where a book is refused or found stale (FR-551, FR-554).
var kind = measured.Kind{
	Invalid: ErrInvalidBook, Stale: ErrStale, Found: "FR-551", Checked: "FR-554",
	Plural: "pauses", Missing: "joins with no pause", MeasuredIn: "was paused in speech sounds",
	Gone: "has a pause but no longer joins",
}

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

// Measured answers the line the pause was found in.
func (e Entry) Measured() measured.Line {
	return measured.Line{Cue: e.Cue, Index: e.Index, Sounds: e.Sounds, Digest: e.Digest}
}

// paused reports whether the entry gives its line a pause: its break was not doubtful (FR-552).
func (e Entry) paused() bool { return !e.Doubtful }

// Voice is one voice's pauses with the digest of the style file its lines were made with.
type Voice = measured.Voice[Entry]

// Book is every voice's pauses with the silence a pause inserts and the digest of the model they
// were found with. Only NewBook makes one, so every Book holds pauses the rules accept.
type Book struct {
	measured.Book[Entry]
	silence int
}

// NewBook builds a book from each voice's pauses, refusing every problem together, each naming its
// voice, its cue and its line (FR-551): a pause before sample 1 as well as what every measured book
// refuses. A book with no voices and no silence is the empty book shipped before the pauses tool first
// runs.
func NewBook(silence int, model string, voices map[string]Voice) (Book, error) {
	var problems []error
	if name, needed := measured.FirstWhere(voices, Entry.paused); needed && silence <= 0 {
		problems = append(problems, fmt.Errorf(
			"%w: %d samples of silence cannot be inserted while %s has a pause (FR-553)", ErrInvalidBook, silence, name,
		))
	}
	book, err := measured.NewBook(kind, model, voices, samplePlace)
	if err != nil {
		problems = append(problems, err)
	}
	if len(problems) > 0 {
		return Book{}, errors.Join(problems...)
	}
	return Book{Book: book, silence: silence}, nil
}

// samplePlace refuses a pause before sample 1; a doubtful line needs no sample (FR-551).
func samplePlace(name string, entry Entry) error {
	if entry.paused() && entry.Sample < firstSample {
		return fmt.Errorf("%w: %s has its pause at sample %d, before sample %d (FR-551)", ErrInvalidBook, name, entry.Sample, firstSample)
	}
	return nil
}

// Silence returns how many samples of silence a pause inserts (FR-553).
func (b Book) Silence() int { return b.silence }
