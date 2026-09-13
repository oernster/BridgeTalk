package tomlfile

import (
	"errors"
	"strings"
	"testing"
)

// shape is a small stand-in for the shapes the application reads.
type shape struct {
	Name  string              `toml:"name"`
	Takes map[string][]string `toml:"takes"`
}

func TestAShapeItHoldsIsDecoded(t *testing.T) {
	t.Parallel()
	var got shape
	raw := "name = \"Alice Hart\"\n[takes]\n\"Docked\" = [\"a.wav\"]\n"
	if err := Decode([]byte(raw), &got); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if got.Name != "Alice Hart" || len(got.Takes["Docked"]) != 1 {
		t.Errorf("got %+v", got)
	}
}

// FR-211: a misspelled key is refused and named, rather than dropped in silence.
func TestAKeyTheShapeDoesNotHoldIsRefused(t *testing.T) {
	t.Parallel()
	var got shape
	err := Decode([]byte("nmae = \"Alice Hart\"\n"), &got)
	if !errors.Is(err, ErrUnknownKey) || !strings.Contains(err.Error(), "nmae") {
		t.Fatalf("got %v, want the unknown key nmae named", err)
	}
}

// Text that is not TOML and a value of the wrong type are the parser's to describe.
func TestWhatIsNotTomlOrHasTheWrongTypeIsRefused(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"name = \n", "name = 3\n"} {
		var got shape
		err := Decode([]byte(raw), &got)
		if err == nil || errors.Is(err, ErrUnknownKey) {
			t.Errorf("%q: got %v, want the parser's refusal", raw, err)
		}
	}
}
