//go:build windows

package plugin

// The Windows half: the thread every plugin call arrives on.

import (
	"testing"

	"golang.org/x/sys/windows"
)

// Every call is made from one thread that belongs to nothing else, which is what lets a
// plugin keep state belonging to a thread, a COM apartment being the usual one.
func TestEveryCallArrivesOnOneThreadOfItsOwn(t *testing.T) {
	t.Parallel()

	on := newRunner()
	defer on.close()

	var seen []uint32
	for range 8 {
		on.do(func() { seen = append(seen, windows.GetCurrentThreadId()) })
	}

	if len(seen) != 8 {
		t.Fatalf("saw %d thread ids, want 8", len(seen))
	}
	for _, id := range seen {
		if id != seen[0] {
			t.Fatalf("calls arrived on %v, want one thread throughout", seen)
		}
	}
	if seen[0] == windows.GetCurrentThreadId() {
		t.Error("calls arrived on the calling thread, so the thread is not the plugin's own")
	}
}
