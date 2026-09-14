package speech

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
)

// formMarks are the brackets and slashes a spelling is written with in a line; a word or a
// spelling in the table of words holds none of them, since the table gives each apart (FR-549).
const formMarks = opening + joining + closing + slash

// Words is the script's table of words whose speech sounds are given once for every line holding
// them (FR-549).
type Words struct {
	byWord       map[string]Piece
	longestFirst []string
}

// NewWords builds the table of words from each word's spellings: British then American; one
// spelling serves both (FR-529). Every entry that cannot be read is refused together, each naming
// its word.
func NewWords(spellings map[string][]string) (Words, error) {
	words := Words{byWord: make(map[string]Piece, len(spellings))}
	var problems []error
	for _, word := range slices.Sorted(maps.Keys(spellings)) {
		piece, err := readWord(word, spellings[word])
		if err != nil {
			problems = append(problems, fmt.Errorf("word %q: %w", word, err))
			continue
		}
		words.byWord[word] = piece
		words.longestFirst = append(words.longestFirst, word)
	}
	if len(problems) > 0 {
		return Words{}, errors.Join(problems...)
	}
	slices.SortStableFunc(words.longestFirst, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	return words, nil
}

// readWord reads one entry of the table as the line [word](/spellings/) would be read, so a
// spelling is held to the same rules wherever it is given (FR-531).
func readWord(word string, spellings []string) (Piece, error) {
	switch {
	case len(spellings) == 0:
		return Piece{}, fmt.Errorf("%w: gives no spelling", ErrBrokenSpelling)
	case strings.ContainsAny(word+strings.Join(spellings, ""), formMarks):
		return Piece{}, fmt.Errorf("%w: holds a bracket or slash, which the table of words is written without", ErrBrokenSpelling)
	}
	line, err := Read(spell(word, spellings...))
	if err != nil {
		return Piece{}, err
	}
	return line.pieces[0], nil
}

// Sounds returns the speech sounds the table gives a word for an accent, reporting whether it
// gives the word any.
func (w Words) Sounds(word string, accent machinevoice.Accent) (string, bool) {
	piece, ok := w.byWord[word]
	return piece.Sounds(accent), ok
}

// Spell writes a line for one accent as ForAccent does, with every whole word of its plain words
// that the table holds spelled with the table's sounds. A word stands whole where no letter or
// digit of the spoken line touches it; a word the line spells keeps the line's spelling (FR-549).
func (w Words) Spell(line Line, accent machinevoice.Accent) string {
	spoken := line.Words()
	var written strings.Builder
	start := 0
	for _, piece := range line.pieces {
		end := start + len(piece.text)
		if piece.Given() {
			written.WriteString(spell(piece.text, piece.Sounds(accent)))
		} else {
			w.spellPlain(&written, spoken, start, end, accent)
		}
		start = end
	}
	return written.String()
}

// spellPlain writes the plain words between start and end of the spoken line, spelling each
// table word standing whole.
func (w Words) spellPlain(written *strings.Builder, spoken string, start, end int, accent machinevoice.Accent) {
	for at := start; at < end; {
		if word, ok := w.wordAt(spoken, at, end); ok {
			written.WriteString(spell(word, w.byWord[word].Sounds(accent)))
			at += len(word)
			continue
		}
		_, size := utf8.DecodeRuneInString(spoken[at:end])
		written.WriteString(spoken[at : at+size])
		at += size
	}
}

// wordAt returns the longest table word standing whole at a point of the spoken line, ending no
// later than end.
func (w Words) wordAt(spoken string, at, end int) (string, bool) {
	if before, _ := utf8.DecodeLastRuneInString(spoken[:at]); wordRune(before) {
		return "", false
	}
	for _, word := range w.longestFirst {
		if !strings.HasPrefix(spoken[at:end], word) {
			continue
		}
		if after, _ := utf8.DecodeRuneInString(spoken[at+len(word):]); !wordRune(after) {
			return word, true
		}
	}
	return "", false
}

// wordRune reports whether a character is part of a word: a letter or a digit.
func wordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
