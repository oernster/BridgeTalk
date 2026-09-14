package pause_test

// FR-553: silence inserted in a made line before the sample its pause goes at.

import (
	"errors"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/pause"
)

// FR-553: the silence goes in before the sample asked for with the line otherwise as made; the
// samples handed in are left as they were. The first sample and the end of the line may be asked
// for.
func TestSilenceGoesInBeforeTheSampleAskedForLeavingTheSamplesAlone(t *testing.T) {
	t.Parallel()
	samples := []float32{1, 2, 3}
	for _, each := range []struct {
		at, silence int
		want        []float32
	}{
		{1, 2, []float32{1, 0, 0, 2, 3}},
		{0, 1, []float32{0, 1, 2, 3}},
		{3, 1, []float32{1, 2, 3, 0}},
		{2, 0, []float32{1, 2, 3}},
	} {
		got, err := pause.Insert(samples, each.at, each.silence)
		if err != nil || !slices.Equal(got, each.want) {
			t.Errorf("Insert(%v, %d, %d) = %v, %v; want %v", samples, each.at, each.silence, got, err, each.want)
		}
	}
	if !slices.Equal(samples, []float32{1, 2, 3}) {
		t.Errorf("the samples handed in became %v", samples)
	}
}

// No silence answers the caller's own copy: changing it changes nothing handed in.
func TestNoSilenceAnswersTheCallersOwnCopy(t *testing.T) {
	t.Parallel()
	samples := []float32{1, 2}
	got, err := pause.Insert(samples, 1, 0)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got[0] = 9
	if samples[0] != 1 {
		t.Errorf("changing the answer changed the samples handed in to %v", samples)
	}
}

// Silence asked for before the first sample, beyond the end of the line or of a negative length is
// refused with nothing answered.
func TestSilenceOutsideTheLineIsRefused(t *testing.T) {
	t.Parallel()
	for _, each := range []struct{ at, silence int }{{-1, 1}, {4, 1}, {1, -1}} {
		got, err := pause.Insert([]float32{1, 2, 3}, each.at, each.silence)
		if !errors.Is(err, pause.ErrOutOfRange) || got != nil {
			t.Errorf("Insert at %d with %d = %v, %v; want ErrOutOfRange", each.at, each.silence, got, err)
		}
	}
}
