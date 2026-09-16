package take_test

// A take is one utterance, whether it arrives as one file or as several. These pin the two
// things the rest of the application asks of it: which take this is; whether it is a take
// at all.

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/take"
)

func TestATakeOfOnePartIsIdentifiedByThatPart(t *testing.T) {
	t.Parallel()

	one := take.Of(`C:\audio\docking.granted.mp3`)

	if got := one.Key(); got != `C:\audio\docking.granted.mp3` {
		t.Errorf("a take of one part identifies itself as %q", got)
	}
}

func TestATakeOfSeveralPartsIsIdentifiedByItsFirst(t *testing.T) {
	t.Parallel()

	several := take.Of(`C:\audio\one.mp3`, `C:\audio\two.mp3`, `C:\audio\three.mp3`)

	if got := several.Key(); got != `C:\audio\one.mp3` {
		t.Errorf("a take of several parts identifies itself as %q, not by its first part", got)
	}
	if len(several) != 3 {
		t.Errorf("a take of three parts holds %d", len(several))
	}
}

func TestAnEmptyTakeHasNoIdentityAndIsEmpty(t *testing.T) {
	t.Parallel()

	var none take.Take

	if got := none.Key(); got != "" {
		t.Errorf("an empty take identifies itself as %q rather than as nothing", got)
	}
	if !none.Empty() {
		t.Error("an empty take does not report itself empty")
	}
}

func TestATakeWithAPartIsNotEmpty(t *testing.T) {
	t.Parallel()

	if take.Of(`C:\audio\one.mp3`).Empty() {
		t.Error("a take holding a part reports itself empty")
	}
}

// FR-591: two spans of one file are two takes, so the picker can avoid the one it chose last.
func TestTwoSpansOfOneFileAreToldApartByTheirKeys(t *testing.T) {
	t.Parallel()

	keys := map[string]bool{}
	for _, each := range []take.Take{
		{take.SpanOf(`C:\audio\many.bin`, "mp3", 0, 4000)},
		{take.SpanOf(`C:\audio\many.bin`, "mp3", 4000, 4000)},
		{take.SpanOf(`C:\audio\many.bin`, "mp3", 0, 8000)},
		take.Of(`C:\audio\many.bin`),
	} {
		keys[each.Key()] = true
	}

	if len(keys) != 4 {
		t.Errorf("four different parts of one file gave %d keys, want four", len(keys))
	}
}

// A take is shown by the file it begins in, whether that part is the whole file or a span of it;
// an empty take begins nowhere.
func TestATakeIsShownByTheFileItBeginsIn(t *testing.T) {
	t.Parallel()

	spanned := take.Take{take.SpanOf(`C:\audio\many.bin`, "mp3", 4000, 4000), take.File(`C:\audio\b.mp3`)}

	if got := spanned.Source(); got != `C:\audio\many.bin` {
		t.Errorf("a take beginning with a span is shown as %q", got)
	}
	if got := (take.Take{}).Source(); got != "" {
		t.Errorf("an empty take is shown as %q, want nothing", got)
	}
}

func TestTwoTakesOfACueAreToldApartByTheirKeys(t *testing.T) {
	t.Parallel()

	first := take.Of(`C:\audio\one.mp3`, `C:\audio\shared.mp3`)
	second := take.Of(`C:\audio\two.mp3`, `C:\audio\shared.mp3`)

	if first.Key() == second.Key() {
		t.Error("two takes sharing a later part are not told apart, so the picker could not avoid one")
	}
}
