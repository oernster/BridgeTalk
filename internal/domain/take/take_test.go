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

func TestTwoTakesOfACueAreToldApartByTheirKeys(t *testing.T) {
	t.Parallel()

	first := take.Of(`C:\audio\one.mp3`, `C:\audio\shared.mp3`)
	second := take.Of(`C:\audio\two.mp3`, `C:\audio\shared.mp3`)

	if first.Key() == second.Key() {
		t.Error("two takes sharing a later part are not told apart, so the picker could not avoid one")
	}
}
