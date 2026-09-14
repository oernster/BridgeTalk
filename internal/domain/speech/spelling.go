package speech

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
)

// ErrBrokenSpelling is returned for a line whose given speech sounds cannot be read (FR-531).
var ErrBrokenSpelling = errors.New("broken spelling")

// A line gives a word's speech sounds as [word](/sounds/); where the two accents differ it gives
// [word](/British/American/) (FR-529).
const (
	opening      = "["
	closingWord  = "]"
	joining      = "]("
	closing      = ")"
	slash        = "/"
	maxSpellings = 2
)

// Piece is a stretch of a line: plain words or one word with its speech sounds given.
type Piece struct {
	text     string
	british  string
	american string
}

// Text returns the piece's words as they are spoken.
func (p Piece) Text() string { return p.text }

// Given reports whether the line gives this piece's speech sounds.
func (p Piece) Given() bool { return p.british != "" }

// Sounds returns the speech sounds given for an accent, empty for plain words. A single
// spelling serves both accents.
func (p Piece) Sounds(accent machinevoice.Accent) string {
	if accent == machinevoice.American {
		return p.american
	}
	return p.british
}

// Line is a script line read into its pieces.
type Line struct {
	pieces []Piece
}

// Read reads a line into its pieces, refusing a spelling it cannot read (FR-531).
func Read(text string) (Line, error) {
	var pieces []Piece
	rest := text
	for start := strings.Index(rest, opening); start >= 0; start = strings.Index(rest, opening) {
		if start > 0 {
			pieces = append(pieces, Piece{text: rest[:start]})
		}
		piece, length, err := readSpelling(rest[start:])
		if err != nil {
			return Line{}, err
		}
		pieces = append(pieces, piece)
		rest = rest[start+length:]
	}
	if rest != "" {
		pieces = append(pieces, Piece{text: rest})
	}
	return Line{pieces: pieces}, nil
}

// Pieces returns the line's pieces in order. The slice is the caller's own.
func (l Line) Pieces() []Piece { return slices.Clone(l.pieces) }

// Words returns the line as it is spoken: its words, with every spelling left out.
func (l Line) Words() string {
	var words strings.Builder
	for _, piece := range l.pieces {
		words.WriteString(piece.text)
	}
	return words.String()
}

// readSpelling reads the [word](/sounds/) that text begins with, returning it with its length.
func readSpelling(text string) (Piece, int, error) {
	join := strings.Index(text, joining)
	if join < 0 {
		return broken(text, "is not closed")
	}
	end := strings.Index(text[join:], closing)
	if end < 0 {
		return broken(text, "is not closed")
	}
	length := join + end + len(closing)
	spelling := text[:length]
	word := text[len(opening):join]
	inside := text[join+len(joining) : join+end]
	switch {
	case strings.ContainsAny(word, opening+closingWord):
		return broken(spelling, "is not closed")
	case word == "":
		return broken(spelling, "gives speech sounds for no word")
	case strings.IndexFunc(word, unicode.IsSpace) >= 0:
		return broken(spelling, "gives one spelling for more than one word")
	case len(inside) < len(slash+slash) || !strings.HasPrefix(inside, slash) || !strings.HasSuffix(inside, slash):
		return broken(spelling, "does not hold its speech sounds between slashes")
	}
	spellings := strings.Split(inside[len(slash):len(inside)-len(slash)], slash)
	if len(spellings) > maxSpellings {
		return broken(spelling, "gives more than two spellings")
	}
	for _, sounds := range spellings {
		if sounds == "" {
			return broken(spelling, "gives an empty spelling")
		}
		if err := readable(sounds); err != nil {
			return Piece{}, 0, fmt.Errorf("%w: %q gives %w", ErrBrokenSpelling, spelling, err)
		}
	}
	return Piece{text: word, british: spellings[0], american: spellings[len(spellings)-1]}, length, nil
}

// broken refuses a spelling, saying why.
func broken(spelling, reason string) (Piece, int, error) {
	return Piece{}, 0, fmt.Errorf("%w: %q %s", ErrBrokenSpelling, spelling, reason)
}
