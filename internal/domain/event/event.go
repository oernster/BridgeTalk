// Package event holds the source-neutral event the cue engine reasons about.
//
// Both the journal tail reader and the status-flag watcher produce values of this
// type, so nothing above infrastructure knows which file an event came from or how
// that file is laid out.
package event

import "time"

// Source names where an event originated.
type Source string

const (
	// SourceJournal is an appended line in a Journal.*.log file.
	SourceJournal Source = "journal"
	// SourceStatus is a changed bit or value in Status.json.
	SourceStatus Source = "status"
	// SourceApplication is a moment in the application rather than in the game, such
	// as a voice being chosen. The game never emits one, so a cue carrying this
	// source is never resolved by a journal line or a status change; it is asked for
	// by id at the moment it belongs to. It is a cue rather than a special case so
	// that it is recorded, counted and auditioned like every other line.
	SourceApplication Source = "application"
)

// Edge describes the direction of a status change. Journal events carry EdgeNone.
type Edge string

const (
	// EdgeNone is carried by every journal event, which has no direction.
	EdgeNone Edge = ""
	// EdgeRising is a status value that became set or true.
	EdgeRising Edge = "rising"
	// EdgeFalling is a status value that became clear or false.
	EdgeFalling Edge = "falling"
)

// Event is one thing that happened, reduced to what the cue engine needs.
//
// Fields is unexported so an Event cannot be altered after construction; read it
// through Field. The map is copied on construction, so the caller's map is not
// retained.
type Event struct {
	source Source
	name   string
	edge   Edge
	fields map[string]any
	at     time.Time
}

// New builds an Event. The fields map is copied, so later changes by the caller
// do not reach the returned value.
func New(source Source, name string, edge Edge, fields map[string]any, at time.Time) Event {
	copied := make(map[string]any, len(fields))
	for key, value := range fields {
		copied[key] = value
	}
	return Event{source: source, name: name, edge: edge, fields: copied, at: at}
}

// Source returns where the event came from.
func (e Event) Source() Source { return e.source }

// Name returns the journal event name; for a status event, the value that changed.
func (e Event) Name() string { return e.name }

// Edge returns the direction of a status change; EdgeNone for a journal event.
func (e Event) Edge() Edge { return e.edge }

// At returns the moment the event was observed.
func (e Event) At() time.Time { return e.at }

// Field returns one payload value and whether it was present.
func (e Event) Field(name string) (any, bool) {
	value, ok := e.fields[name]
	return value, ok
}

// FieldCount reports how many payload fields the event carries.
func (e Event) FieldCount() int { return len(e.fields) }
