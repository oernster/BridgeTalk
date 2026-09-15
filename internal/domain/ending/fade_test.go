package ending_test

// FR-556: a made line faded from the sample its ending was found at, falling linearly to zero and then
// silent to its end, so it keeps its length.

import (
	"errors"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/ending"
)

// FR-556's acceptance in the domain: the samples before the start are unchanged, sample start + i is
// multiplied by (length - i) / length and every sample from start + length is zero. The line keeps its
// length; the samples handed in are left as they were.
func TestAFadeFallsLinearlyToZeroFromItsStartThenStaysSilent(t *testing.T) {
	t.Parallel()
	samples := []float32{8, 8, 8, 8, 8, 8, 8}
	got, err := ending.FadeOut(samples, 2, 4)
	if want := []float32{8, 8, 8, 6, 4, 2, 0}; err != nil || !slices.Equal(got, want) {
		t.Errorf("FadeOut = %v, %v; want %v", got, err, want)
	}
	if !slices.Equal(samples, []float32{8, 8, 8, 8, 8, 8, 8}) {
		t.Errorf("the samples handed in became %v", samples)
	}
}

// A fade running past the end of the line falls as far as the line goes; a fade starting at the end
// changes nothing. Either answers the caller's own copy.
func TestAFadeCutShortByTheEndOfTheLineFallsAsFarAsTheLineGoes(t *testing.T) {
	t.Parallel()
	for _, each := range []struct {
		samples    []float32
		at, length int
		want       []float32
	}{
		{[]float32{8, 8, 8}, 1, 4, []float32{8, 8, 6}},
		{[]float32{1, 2}, 2, 4, []float32{1, 2}},
	} {
		got, err := ending.FadeOut(each.samples, each.at, each.length)
		if err != nil || !slices.Equal(got, each.want) {
			t.Errorf("FadeOut(%v, %d, %d) = %v, %v; want %v", each.samples, each.at, each.length, got, err, each.want)
			continue
		}
		if len(got) > 0 {
			got[0] = 9
			if each.samples[0] == 9 {
				t.Errorf("changing the answer changed the samples handed in to %v", each.samples)
			}
		}
	}
}

// A fade starting before the first sample or beyond the end of the line is refused with nothing
// answered; so is a fade lasting no samples.
func TestAFadeOutsideTheLineOrOfNoLengthIsRefused(t *testing.T) {
	t.Parallel()
	for _, each := range []struct{ at, length int }{{-1, 4}, {3, 4}, {1, 0}, {1, -1}} {
		got, err := ending.FadeOut([]float32{1, 2}, each.at, each.length)
		if !errors.Is(err, ending.ErrOutOfRange) || got != nil {
			t.Errorf("FadeOut at %d over %d = %v, %v; want ErrOutOfRange", each.at, each.length, got, err)
		}
	}
}
