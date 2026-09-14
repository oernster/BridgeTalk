package pause

import (
	"math"
	"slices"
)

// doubtfulShare is how far a final voiced stretch may lie from its voice's median before its break
// is doubtful: a quarter of the median, longer or shorter (FR-552).
const doubtfulShare = 0.25

// Doubtful answers, for a voice's lines in order, whether each final voiced stretch is more than a
// quarter longer or shorter than the median over them all (FR-552). A final that is not positive is
// doubtful whatever the median.
func Doubtful(finals []float64) []bool {
	doubtful := make([]bool, len(finals))
	if len(finals) == 0 {
		return doubtful
	}
	middle := median(finals)
	for index, final := range finals {
		doubtful[index] = final <= 0 || math.Abs(final-middle) > middle*doubtfulShare
	}
	return doubtful
}

// median answers the middle of values, which holds at least one; for an even count, the mean of the
// middle two.
func median(values []float64) float64 {
	sorted := slices.Sorted(slices.Values(values))
	half := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[half-1] + sorted[half]) / 2
	}
	return sorted[half]
}
