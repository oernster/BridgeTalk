package pause

import (
	"errors"
	"fmt"
)

// ErrOutOfRange is returned for silence that cannot be inserted where it was asked for (FR-553).
var ErrOutOfRange = errors.New("pause out of range")

// Insert answers samples with silence zero samples inserted before the sample at, leaving samples
// unchanged; no silence answers a copy (FR-553). The end of the line may be asked for; a sample
// beyond it, a negative one or negative silence is refused.
func Insert(samples []float32, at, silence int) ([]float32, error) {
	if at < 0 || at > len(samples) {
		return nil, fmt.Errorf("%w: sample %d is outside a line of %d samples (FR-553)", ErrOutOfRange, at, len(samples))
	}
	if silence < 0 {
		return nil, fmt.Errorf("%w: %d samples of silence (FR-553)", ErrOutOfRange, silence)
	}
	paused := make([]float32, 0, len(samples)+silence)
	paused = append(paused, samples[:at]...)
	paused = append(paused, make([]float32, silence)...)
	return append(paused, samples[at:]...), nil
}
