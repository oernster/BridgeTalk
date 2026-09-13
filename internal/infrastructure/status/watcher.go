package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// fileName is the status file the game rewrites in place.
const fileName = "Status.json"

// reading is one parse of the status file.
type reading struct {
	Flags    uint32 `json:"Flags"`
	Flags2   uint32 `json:"Flags2"`
	GuiFocus int    `json:"GuiFocus"`
	FireGrp  int    `json:"FireGroup"`
	Pips     []int  `json:"Pips"`
	Stamp    string `json:"timestamp"`
}

// Watcher emits an event for every status value that changes.
type Watcher struct {
	path    string
	clock   func() time.Time
	primed  bool
	current reading
}

// NewWatcher builds a watcher over the status file in a journal directory.
//
// The first poll primes the baseline and emits nothing. Emitting on the first read
// would fire a cue for every flag that merely happened to be set when the
// application started, which is state rather than news.
func NewWatcher(directory string, clock func() time.Time) (*Watcher, error) {
	path := filepath.Join(directory, fileName)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, refusal.Reason(err))
	}
	return &Watcher{path: path, clock: clock}, nil
}

// Name identifies the source in logs and on the home pane.
func (w *Watcher) Name() string { return "status" }

// Path returns the file being watched.
func (w *Watcher) Path() string { return w.path }

// Poll reads the status file and returns one event per changed value.
//
// A read that fails or does not parse is discarded rather than reported. The game
// rewrites this file in place and far more often than its contents actually change,
// so catching it mid-write is expected rather than exceptional.
func (w *Watcher) Poll() ([]event.Event, error) {
	next, ok := w.read()
	if !ok {
		return nil, nil
	}
	if !inSession(next) {
		// The baseline goes with the session it described. The next one may be a
		// different ship in a different place, so its first reading is state rather
		// than change, exactly as the first reading of the run is; keeping the old
		// baseline would announce the difference between two sessions as though the
		// ship had done something between them.
		w.primed = false
		w.current = reading{}
		return nil, nil
	}
	if !w.primed {
		w.current = next
		w.primed = true
		return nil, nil
	}
	previous := w.current
	w.current = next
	if unchanged(previous, next) {
		return nil, nil
	}
	return w.diff(previous, next), nil
}

// inSession reports whether a reading describes a commander who is actually playing.
//
// On exit the game rewrites the file as a single line carrying nothing but a zeroed
// Flags, with no Flags2, GuiFocus, FireGroup or Pips at all. Diffed against the state
// the commander left, every bit that was set reads as falling, so quitting from a
// landed ship announced the gear being raised, the lights going out and the shields
// dropping, none of which happened. It is the same mistake the priming poll avoids at
// the other end: state, arriving as though it were news.
//
// The test is that no flag is set in either word. Wherever a commander can be, the
// game says which: in the main ship, in a fighter, in an SRV or on foot, one of those
// is always set while there is a session to be in. Both words at zero is the game
// saying there is nobody anywhere, which is not a place a ship can do anything from.
func inSession(state reading) bool {
	return state.Flags != 0 || state.Flags2 != 0
}

// unchanged reports whether two readings describe the same state.
//
// The timestamp is ignored because the game rewrites the file on a cadence of its
// own whether or not anything moved; comparing it would make every rewrite look
// like news. Pips are compared element by element, since a slice is not comparable.
func unchanged(previous, next reading) bool {
	if previous.Flags != next.Flags || previous.Flags2 != next.Flags2 {
		return false
	}
	if previous.GuiFocus != next.GuiFocus || previous.FireGrp != next.FireGrp {
		return false
	}
	if len(previous.Pips) != len(next.Pips) {
		return false
	}
	for index := range previous.Pips {
		if previous.Pips[index] != next.Pips[index] {
			return false
		}
	}
	return true
}

// read parses the status file, reporting failure as "nothing to say".
func (w *Watcher) read() (reading, bool) {
	raw, err := os.ReadFile(w.path)
	if err != nil || len(raw) == 0 {
		return reading{}, false
	}
	var parsed reading
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return reading{}, false
	}
	return parsed, true
}

// diff builds the events describing what changed between two readings.
func (w *Watcher) diff(previous, next reading) []event.Event {
	at := w.clock()
	if stamp, err := time.Parse(time.RFC3339, next.Stamp); err == nil {
		at = stamp
	}

	var events []event.Event
	events = append(events, bitEvents(flagBits, previous.Flags, next.Flags, at)...)
	events = append(events, bitEvents(flag2Bits, previous.Flags2, next.Flags2, at)...)

	if previous.GuiFocus != next.GuiFocus {
		events = append(events, valueEvent(ValueGuiFocus, guiFocusName(next.GuiFocus), at))
	}
	if previous.FireGrp != next.FireGrp {
		events = append(events, valueEvent(ValueFireGroup, fmt.Sprintf("%d", next.FireGrp), at))
	}
	if before, after := dominantPips(previous.Pips), dominantPips(next.Pips); before != after {
		events = append(events, valueEvent(ValuePips, after, at))
	}
	return events
}

// bitEvents emits a rising or falling event for each bit that changed.
//
// Bits are emitted in ascending numeric order so a burst of simultaneous changes is
// reported deterministically, which keeps the reaction log readable and the tests
// stable.
func bitEvents(table map[uint32]string, previous, next uint32, at time.Time) []event.Event {
	changed := previous ^ next
	if changed == 0 {
		return nil
	}
	bits := make([]uint32, 0, len(table))
	for bit := range table {
		if changed&bit != 0 {
			bits = append(bits, bit)
		}
	}
	sort.Slice(bits, func(a, b int) bool { return bits[a] < bits[b] })

	events := make([]event.Event, 0, len(bits))
	for _, bit := range bits {
		edge := event.EdgeFalling
		if next&bit != 0 {
			edge = event.EdgeRising
		}
		events = append(events, event.New(
			event.SourceStatus,
			table[bit],
			edge,
			map[string]any{"value": next&bit != 0},
			at,
		))
	}
	return events
}

// valueEvent builds a rising event for a non-bit status value.
func valueEvent(name, value string, at time.Time) event.Event {
	return event.New(
		event.SourceStatus,
		name,
		event.EdgeRising,
		map[string]any{"value": value},
		at,
	)
}
