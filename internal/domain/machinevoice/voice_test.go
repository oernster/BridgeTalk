package machinevoice_test

// FR-508, FR-510 and FR-528: the machine voices offered, the accent each speaks with and the
// name each is shown by.

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
)

// FR-508: four British female, four British male, eleven American female and nine American
// male, none of them offered twice.
func TestTheVoicesOfferedAreTheTwentyEightOfFR508(t *testing.T) {
	want := map[machinevoice.Accent]map[machinevoice.Sex]int{
		machinevoice.British:  {machinevoice.Female: 4, machinevoice.Male: 4},
		machinevoice.American: {machinevoice.Female: 11, machinevoice.Male: 9},
	}
	got := map[machinevoice.Accent]map[machinevoice.Sex]int{
		machinevoice.British:  {},
		machinevoice.American: {},
	}
	seen := map[string]bool{}
	for _, voice := range machinevoice.All() {
		if seen[voice.ID()] {
			t.Errorf("%s is offered twice", voice.ID())
		}
		seen[voice.ID()] = true
		got[voice.Accent()][voice.Sex()]++
	}
	for accent, bySex := range want {
		for sex, count := range bySex {
			if got[accent][sex] != count {
				t.Errorf("%s %s voices: got %d, want %d", accent, sex, got[accent][sex], count)
			}
		}
	}
	if len(seen) != 28 {
		t.Errorf("got %d distinct voices, want 28", len(seen))
	}
}

// FR-510: the accent a voice speaks with is the one its id names.
func TestAVoiceSpeaksWithTheAccentItsIdNames(t *testing.T) {
	for id, want := range map[string]machinevoice.Accent{
		"bf_emma":    machinevoice.British,
		"am_michael": machinevoice.American,
	} {
		voice, err := machinevoice.Parse(id)
		if err != nil {
			t.Fatalf("Parse(%q): %v", id, err)
		}
		if voice.Accent() != want {
			t.Errorf("%s speaks %s, want %s", id, voice.Accent(), want)
		}
	}
}

// FR-528: the name in the id, capitalised, then the accent and sex in brackets. One voice
// from each of the four groups.
func TestAVoiceIsNamedByItsNameThenItsAccentAndSex(t *testing.T) {
	for id, want := range map[string]string{
		"bf_emma":    "Emma (British, female)",
		"bm_george":  "George (British, male)",
		"af_heart":   "Heart (American, female)",
		"am_michael": "Michael (American, male)",
	} {
		voice, err := machinevoice.Parse(id)
		if err != nil {
			t.Fatalf("Parse(%q): %v", id, err)
		}
		if got := voice.Name(); got != want {
			t.Errorf("%s is named %q, want %q", id, got, want)
		}
	}
}

// Every voice offered is found again by its own id.
func TestEveryOfferedVoiceIsFoundByItsId(t *testing.T) {
	for _, voice := range machinevoice.All() {
		got, err := machinevoice.Parse(voice.ID())
		if err != nil || got != voice {
			t.Errorf("Parse(%q) = %v, %v; want the voice offered", voice.ID(), got, err)
		}
	}
}

// An id that is not one of the voices offered is refused, naming the id as given.
func TestAnIdThatIsNotOfferedIsRefusedNamingIt(t *testing.T) {
	for _, id := range []string{"bf_nobody", "xf_emma", "Emma", "BF_EMMA", ""} {
		_, err := machinevoice.Parse(id)
		if !errors.Is(err, machinevoice.ErrUnknownVoice) || !strings.Contains(err.Error(), strconv.Quote(id)) {
			t.Errorf("Parse(%q) = %v, want ErrUnknownVoice naming the id", id, err)
		}
	}
}

// The list handed out is a copy, so no caller can change what the next one is offered.
func TestTheVoicesOfferedCannotBeChangedByACaller(t *testing.T) {
	first := machinevoice.All()
	offered := first[0]
	first[0] = machinevoice.Voice{}
	if machinevoice.All()[0] != offered {
		t.Error("changing the returned list changed the voices offered")
	}
}

// Every accent is listed, British first; the list handed out is the caller's own.
func TestEveryAccentIsListedBritishFirst(t *testing.T) {
	got := machinevoice.Accents()
	if len(got) != 2 || got[0] != machinevoice.British || got[1] != machinevoice.American {
		t.Fatalf("Accents() = %v, want British then American", got)
	}
	got[0] = machinevoice.American
	if machinevoice.Accents()[0] != machinevoice.British {
		t.Error("changing the returned accents changed them")
	}
}
