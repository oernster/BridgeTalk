package tomlfile_test

import (
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
)

// A shape is written as TOML that reads back into the same shape, a key holding a dot and a
// speech sound symbol included.
func TestAShapeIsWrittenAsTomlThatReadsBack(t *testing.T) {
	type shape struct {
		Name string              `toml:"name"`
		Keys map[string][]string `toml:"keys"`
	}
	written, err := tomlfile.Encode(shape{Name: "x", Keys: map[string][]string{"Cast.Confirmed": {"ˈa"}}})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var back shape
	if err := tomlfile.Decode(written, &back); err != nil {
		t.Fatalf("Decode of %q: %v", written, err)
	}
	if got := back.Keys["Cast.Confirmed"]; back.Name != "x" || len(got) != 1 || got[0] != "ˈa" {
		t.Errorf("read back %+v from %q", back, written)
	}
}

// A value TOML cannot hold is refused.
func TestAValueTomlCannotHoldIsRefused(t *testing.T) {
	if _, err := tomlfile.Encode(map[string]any{"c": make(chan int)}); err == nil {
		t.Error("a channel was written as TOML")
	}
}
