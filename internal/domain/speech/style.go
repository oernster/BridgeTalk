package speech

import (
	"errors"
	"fmt"
	"slices"
)

// StyleWidth is how many numbers one row of a voice's style file holds. Measured on 2026-09-14
// from the files shipped with the model: each is 522,240 bytes, which is MaxSymbols rows of
// StyleWidth four-byte numbers.
const StyleWidth = 256

// boundaries counts the markers Tokens puts around a line's symbols: one at each end.
const boundaries = 2

var (
	// ErrMisshapenStyle is returned for a style that is not MaxSymbols rows of StyleWidth numbers.
	ErrMisshapenStyle = errors.New("a style not shaped as the model reads it")
	// ErrNoSounds is returned for a line that comes to no symbols, which has no style row.
	ErrNoSounds = errors.New("no speech sounds to make")
)

// Style is a voice's style file read as numbers: one row for each count of symbols a line can
// come to, from one to MaxSymbols. Only NewStyle makes a usable one.
type Style struct {
	values []float32
}

// NewStyle holds a voice's style numbers, refusing any count but MaxSymbols rows of StyleWidth.
// The style keeps its own copy.
func NewStyle(values []float32) (Style, error) {
	if want := MaxSymbols * StyleWidth; len(values) != want {
		return Style{}, fmt.Errorf("%w: %d numbers, want %d", ErrMisshapenStyle, len(values), want)
	}
	return Style{values: slices.Clone(values)}, nil
}

// For returns the row the model reads beside a line's numbers: the row for how many symbols the
// line comes to, the boundary at each end not counted, as Kokoro chooses it. The row is the
// caller's own.
func (s Style) For(tokens []int64) ([]float32, error) {
	if len(s.values) == 0 {
		return nil, fmt.Errorf("%w: it holds no numbers", ErrMisshapenStyle)
	}
	symbols := len(tokens) - boundaries
	if symbols < 1 {
		return nil, ErrNoSounds
	}
	if symbols > MaxSymbols {
		return nil, fmt.Errorf("%w: %d symbols, at most %d", ErrTooLong, symbols, MaxSymbols)
	}
	row := symbols - 1
	return slices.Clone(s.values[row*StyleWidth : (row+1)*StyleWidth]), nil
}
