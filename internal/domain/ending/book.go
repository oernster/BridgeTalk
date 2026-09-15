// Package ending holds the hiss a machine voice adds after a final nasal: where each line ending on a
// nasal fades, the book the fades ship in, the check that they are not stale and the fade itself
// (FR-555 to FR-557).
package ending

import (
	"errors"
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/measured"
)

// ErrInvalidBook is returned for endings that cannot be shipped as they stand (FR-555).
var ErrInvalidBook = errors.New("invalid endings")

// kind is what an ending says of itself where a book is refused or found stale (FR-555, FR-557).
var kind = measured.Kind{
	Invalid: ErrInvalidBook, Stale: ErrStale, Found: "FR-555", Checked: "FR-557",
	Plural: "endings", Missing: "ends on a nasal with no ending", MeasuredIn: "had its ending found in speech sounds",
	Gone: "has an ending but no longer ends on a nasal",
}

// Entry is one line's ending for one voice: where the line's fade starts, if it ends on a burst
// (FR-555).
type Entry struct {
	// Cue is the cue the line is spoken for.
	Cue cue.ID
	// Index is the line's place among its cue's lines, from zero.
	Index int
	// Sounds is the saved speech sounds the line was made from.
	Sounds string
	// Digest is the digest of the samples the burst was looked for in (pause.Digest).
	Digest string
	// Sample is the sample the fade starts at; zero where the line ends on no burst.
	Sample int
}

// Fades reports whether the line is faded: it ends on a burst, so its fade has a sample (FR-556).
func (e Entry) Fades() bool { return e.Sample > 0 }

// Measured answers the line the ending was found in.
func (e Entry) Measured() measured.Line {
	return measured.Line{Cue: e.Cue, Index: e.Index, Sounds: e.Sounds, Digest: e.Digest}
}

// Voice is one voice's endings with the digest of the style file its lines were made with.
type Voice = measured.Voice[Entry]

// Book is every voice's endings with how many samples a fade lasts and the digest of the model they
// were found with. Only NewBook makes one, so every Book holds endings the rules accept.
type Book struct {
	measured.Book[Entry]
	fade int
}

// NewBook builds a book from each voice's endings, refusing every problem together, each naming its
// voice, its cue and its line (FR-555): a fade before the first sample, a fade with no length while a
// line fades, as well as what every measured book refuses. A book with no voices and no fade is the
// empty book shipped before the pauses tool first finds the endings.
func NewBook(fade int, model string, voices map[string]Voice) (Book, error) {
	var problems []error
	if name, needed := measured.FirstWhere(voices, Entry.Fades); needed && fade <= 0 {
		problems = append(problems, fmt.Errorf(
			"%w: a fade of %d samples cannot be applied while %s fades (FR-556)", ErrInvalidBook, fade, name,
		))
	}
	book, err := measured.NewBook(kind, model, voices, samplePlace)
	if err != nil {
		problems = append(problems, err)
	}
	if len(problems) > 0 {
		return Book{}, errors.Join(problems...)
	}
	return Book{Book: book, fade: fade}, nil
}

// samplePlace refuses a fade starting before the first sample (FR-555).
func samplePlace(name string, entry Entry) error {
	if entry.Sample < 0 {
		return fmt.Errorf("%w: %s has its fade at sample %d, before the first (FR-555)", ErrInvalidBook, name, entry.Sample)
	}
	return nil
}

// Fade returns how many samples a fade lasts (FR-556).
func (b Book) Fade() int { return b.fade }
