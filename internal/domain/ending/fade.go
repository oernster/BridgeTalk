package ending

import (
	"errors"
	"fmt"
	"slices"
)

// ErrOutOfRange is returned for a fade that cannot be applied where it was asked for (FR-556).
var ErrOutOfRange = errors.New("fade out of range")

// FadeOut answers samples faded from the sample at over length samples: sample at + i multiplied by
// (length - i) / length, then zero to the end, with the samples before at unchanged and the length
// kept; the samples handed in are left alone (FR-556). A fade starting at the end of the line changes
// nothing. A fade starting before the first sample or beyond the end is refused; so is one lasting
// no samples.
func FadeOut(samples []float32, at, length int) ([]float32, error) {
	if at < 0 || at > len(samples) {
		return nil, fmt.Errorf("%w: sample %d is outside a line of %d samples (FR-556)", ErrOutOfRange, at, len(samples))
	}
	if length < 1 {
		return nil, fmt.Errorf("%w: a fade of %d samples (FR-556)", ErrOutOfRange, length)
	}
	faded := slices.Clone(samples)
	for step := range len(faded) - at {
		if step >= length {
			faded[at+step] = 0
			continue
		}
		faded[at+step] *= float32(length-step) / float32(length)
	}
	return faded, nil
}
