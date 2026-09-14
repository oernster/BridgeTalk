package pause_test

// FR-552: a line whose final voiced stretch lies far from its voice's median has a doubtful break.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/pause"
)

// doubtfulAre asserts which of a voice's finals are doubtful.
func doubtfulAre(t *testing.T, finals []float64, want []bool) {
	t.Helper()
	if got := pause.Doubtful(finals); !slices.Equal(got, want) {
		t.Errorf("Doubtful(%v) = %v, want %v", finals, got, want)
	}
}

// FR-552's acceptance: with a median of 480 ms, a line at 290 ms is doubtful while one at 520 ms is
// not.
func TestALineFarFromItsVoicesMedianIsDoubtful(t *testing.T) {
	t.Parallel()
	doubtfulAre(t, []float64{480, 290, 520, 470, 480}, []bool{false, true, false, false, false})
}

// The median of an even count is the mean of the middle two: 250 here, so 200 and 300 lie near it
// while 100 and 400 do not. Either middle value alone would judge 200 or 300 doubtful.
func TestTheMedianOfAnEvenCountIsTheMeanOfTheMiddleTwo(t *testing.T) {
	t.Parallel()
	doubtfulAre(t, []float64{400, 100, 300, 200}, []bool{true, true, false, false})
}

// A line exactly a quarter longer or shorter than the median is not doubtful; only more than a
// quarter is.
func TestALineExactlyAQuarterFromTheMedianIsNotDoubtful(t *testing.T) {
	t.Parallel()
	doubtfulAre(t, []float64{400, 500, 300, 400}, []bool{false, false, false, false})
}

// A final voiced stretch that is not positive is doubtful, even where the median is the same.
func TestAFinalThatIsNotPositiveIsDoubtful(t *testing.T) {
	t.Parallel()
	doubtfulAre(t, []float64{0, 0, -1}, []bool{true, true, true})
}

// No lines are judged as none.
func TestNoLinesAreJudgedAsNone(t *testing.T) {
	t.Parallel()
	if got := pause.Doubtful(nil); len(got) != 0 {
		t.Errorf("Doubtful(nil) = %v, want none", got)
	}
}
