// Package speech holds speech sounds as the model reads them: the symbols it knows, the
// numbers it reads them as and the spellings a script line may give for a word.
package speech

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// MaxSymbols is the most speech sound symbols one line may come to (FR-506). The model refuses
// more; the boundary at each end is read on top of them.
const MaxSymbols = 510

var (
	// ErrUnreadableSymbol is returned for a symbol the model does not read.
	ErrUnreadableSymbol = errors.New("a symbol the model does not read")
	// ErrTooLong is returned for speech sounds longer than the model reads.
	ErrTooLong = errors.New("more speech sounds than the model reads")
)

// Tokens turns speech sounds into the numbers the model reads, with the boundary at each end.
func Tokens(sounds string) ([]int64, error) {
	if count := utf8.RuneCountInString(sounds); count > MaxSymbols {
		return nil, fmt.Errorf("%w: %d symbols, at most %d", ErrTooLong, count, MaxSymbols)
	}
	if err := readable(sounds); err != nil {
		return nil, err
	}
	numbers := []int64{boundary}
	for _, symbol := range sounds {
		numbers = append(numbers, symbols[symbol])
	}
	return append(numbers, boundary), nil
}

// readable refuses the first symbol in sounds that the model does not read.
func readable(sounds string) error {
	for _, symbol := range sounds {
		if _, ok := symbols[symbol]; !ok {
			return fmt.Errorf("%w: %q", ErrUnreadableSymbol, symbol)
		}
	}
	return nil
}
