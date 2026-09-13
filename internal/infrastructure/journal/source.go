package journal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/event"
)

// journalPattern matches the game's journal file names.
const journalPattern = "Journal.*.log"

// eventField is the key every journal line carries naming what happened.
const eventField = "event"

// timestampField is the key every journal line carries dating what happened.
const timestampField = "timestamp"

// Source is a live, forward-only view of the newest journal file.
//
// It never replays history. On construction it seeks to the end of the newest file,
// so launching the application mid-session does not fire hundreds of cues at once.
// When the game opens a new journal the source follows it from byte zero, because
// the contents of a newly created file genuinely are new.
type Source struct {
	directory string
	path      string
	offset    int64
	partial   []byte
	clock     func() time.Time
}

// NewSource opens a source positioned at the end of the newest journal.
func NewSource(directory string, clock func() time.Time) (*Source, error) {
	if _, err := os.Stat(directory); err != nil {
		return nil, fmt.Errorf("journal directory %q: %w", directory, err)
	}
	source := &Source{directory: directory, clock: clock}
	newest, err := source.newestPath()
	if err != nil {
		return nil, err
	}
	source.path = newest
	source.offset = FileSize(newest)
	return source, nil
}

// Name identifies the source in logs and on the home pane.
func (s *Source) Name() string { return "journal" }

// Path returns the file currently being followed.
func (s *Source) Path() string { return s.path }

// Poll returns the events appended since the last call.
func (s *Source) Poll() ([]event.Event, error) {
	newest, err := s.newestPath()
	if err != nil {
		return nil, err
	}
	if newest != s.path {
		// A new journal means a new play session. Read it from the start.
		s.path = newest
		s.offset = emptyOffset
		s.partial = nil
	}

	result := ReadNewBytes(s.path, s.offset, s.partial)
	s.offset = result.Offset
	s.partial = result.Partial

	events := make([]event.Event, 0, len(result.Lines))
	for _, line := range result.Lines {
		parsed, ok := s.parse(line)
		if ok {
			events = append(events, parsed)
		}
	}
	return events, nil
}

// parse turns one journal line into an Event, dropping anything malformed.
//
// A single unreadable line is not an error worth surfacing: the game is the writer
// and it occasionally emits something this application has no schema for.
func (s *Source) parse(line string) (event.Event, bool) {
	var fields map[string]any
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		return event.Event{}, false
	}
	name, ok := fields[eventField].(string)
	if !ok || name == "" {
		return event.Event{}, false
	}
	at := s.clock()
	if stamp, ok := fields[timestampField].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, stamp); err == nil {
			at = parsed
		}
	}
	return event.New(event.SourceJournal, name, event.EdgeNone, fields, at), true
}

// newestPath returns the most recently modified journal file.
func (s *Source) newestPath() (string, error) {
	matches, err := filepath.Glob(filepath.Join(s.directory, journalPattern))
	if err != nil {
		return "", fmt.Errorf("scanning %q: %w", s.directory, err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no journal files in %q", s.directory)
	}
	// Journal names embed a sortable timestamp, so lexical order is time order and
	// a name comparison avoids stat-ing every file on every poll.
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

// StandardLocation returns the game's usual journal location for this user.
func StandardLocation() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	candidate := filepath.Join(
		home, "Saved Games", "Frontier Developments", "Elite Dangerous",
	)
	if _, err := os.Stat(candidate); err != nil {
		return "", fmt.Errorf("journal directory not found at %q: %w", candidate, err)
	}
	return candidate, nil
}

// IsJournalFile reports whether a name looks like a journal file, used by tests and
// by the directory chooser to validate what the user picked.
func IsJournalFile(name string) bool {
	base := filepath.Base(name)
	return strings.HasPrefix(base, "Journal.") && strings.HasSuffix(base, ".log")
}
