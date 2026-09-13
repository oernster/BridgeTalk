// Package cue holds the cue vocabulary: what a cue is, how urgent it is and how a
// cue decides that an event belongs to it.
package cue

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/event"
)

// ErrInvalidCue is returned when a cue definition cannot be honoured.
var ErrInvalidCue = errors.New("invalid cue")

// ID identifies a cue across the cue table and every voice.
//
// Ids are written in the game's own words: the journal event or status value first,
// then whatever narrows it, "ShieldState.ShieldsUp.false" or "LightsOn.Set". That
// first segment is the only grouping the vocabulary needs, so Group reads it rather
// than a second table being kept in step with this one.
type ID string

// Group returns the part of the id before the first dot, which names the area of
// the game the cue belongs to. An id with no dot is its own group.
func (i ID) Group() string {
	text := string(i)
	if cut := strings.IndexByte(text, groupSeparator); cut >= 0 {
		return text[:cut]
	}
	return text
}

// groupSeparator divides the segments of a cue id.
const groupSeparator = '.'

// Priority orders cues against each other. Higher wins.
type Priority int

const (
	// PriorityFlavour is incidental character, dropped whenever anything is pending.
	PriorityFlavour Priority = iota
	// PriorityAmbient is worth saying when nothing else is queued.
	PriorityAmbient
	// PriorityNotice is worth queueing for.
	PriorityNotice
	// PriorityAlert interrupts whatever is speaking.
	PriorityAlert
)

// priorityNames maps the wire spelling of a priority to its value.
var priorityNames = map[string]Priority{
	"flavour": PriorityFlavour,
	"ambient": PriorityAmbient,
	"notice":  PriorityNotice,
	"alert":   PriorityAlert,
}

// ParsePriority resolves the spelling used in the cue table.
func ParsePriority(text string) (Priority, error) {
	value, ok := priorityNames[text]
	if !ok {
		return PriorityFlavour, fmt.Errorf("%w: unknown priority %q", ErrInvalidCue, text)
	}
	return value, nil
}

// Cue is one thing the application can say, plus the condition that says it.
type Cue struct {
	id       ID
	source   event.Source
	name     string
	edge     event.Edge
	match    map[string]string
	priority Priority
	cooldown time.Duration
	purpose  string
}

// Definition is the unvalidated shape a cue arrives in from the cue table.
type Definition struct {
	ID       string
	Source   string
	Event    string
	Flag     string
	Edge     string
	Match    map[string]string
	Priority string
	Cooldown time.Duration
	Purpose  string
}

// endsInDigits reports whether an id's final segment is made of digits alone (FR-219).
//
// A take in the flat form is numbered by a trailing dot and digits, so a file named for a
// cue followed by .2 is that cue's second take. An id ending in digits would give such a
// file two meanings, which is why no table may hold one, whether it shipped with the
// application or was supplied by the user.
func endsInDigits(id string) bool {
	last := id[strings.LastIndexByte(id, groupSeparator)+1:]
	if last == "" {
		return false
	}
	for _, letter := range last {
		if letter < '0' || letter > '9' {
			return false
		}
	}
	return true
}

// strippedEndings are the characters Windows silently removes from the end of a file or
// directory name.
const strippedEndings = ". "

// endsInStrippedCharacter reports whether an id ends in a dot or a space (FR-222).
//
// A voice's recordings are found by a name equal to the id. Windows strips a trailing dot
// or space from a name as it is created, so a folder made for such an id would arrive
// under a different name and never be found.
func endsInStrippedCharacter(id string) bool {
	return strings.ContainsRune(strippedEndings, rune(id[len(id)-1]))
}

// folderSeparator is what a cue folder's name carries in place of each dot (FR-229).
const folderSeparator = '_'

// Folder returns the name of the folder that holds this cue's takes: the id with every dot
// written as an underscore (FR-229). It is the one place that form is worked out. No id
// holds an underscore (FR-230), so each folder name belongs to exactly one id.
func (id ID) Folder() string {
	return strings.ReplaceAll(string(id), string(groupSeparator), string(folderSeparator))
}

// New validates a definition into a Cue.
//
// A journal cue names an event; a status cue names a flag and may name an edge.
func New(definition Definition) (Cue, error) {
	if definition.ID == "" {
		return Cue{}, fmt.Errorf("%w: empty id", ErrInvalidCue)
	}
	if endsInDigits(definition.ID) {
		return Cue{}, fmt.Errorf(
			"%w: %s ends in a segment of digits, which the flat form reads as a take number",
			ErrInvalidCue, definition.ID,
		)
	}
	if endsInStrippedCharacter(definition.ID) {
		return Cue{}, fmt.Errorf(
			"%w: %q ends in a dot or a space, which Windows strips from a file name",
			ErrInvalidCue, definition.ID,
		)
	}
	if strings.ContainsRune(definition.ID, folderSeparator) {
		return Cue{}, fmt.Errorf(
			"%w: %q holds an underscore, which a cue folder's name writes in place of a dot",
			ErrInvalidCue, definition.ID,
		)
	}

	source := event.Source(definition.Source)
	name := definition.Event
	if source == event.SourceStatus {
		name = definition.Flag
	}
	switch source {
	case event.SourceJournal, event.SourceStatus, event.SourceApplication:
	default:
		return Cue{}, fmt.Errorf("%w: %s has unknown source %q", ErrInvalidCue, definition.ID, definition.Source)
	}
	if name == "" {
		return Cue{}, fmt.Errorf("%w: %s names no event or flag", ErrInvalidCue, definition.ID)
	}

	priority := PriorityAmbient
	if definition.Priority != "" {
		parsed, err := ParsePriority(definition.Priority)
		if err != nil {
			return Cue{}, fmt.Errorf("%s: %w", definition.ID, err)
		}
		priority = parsed
	}

	edge := event.Edge(definition.Edge)
	switch edge {
	case event.EdgeNone, event.EdgeRising, event.EdgeFalling:
	default:
		return Cue{}, fmt.Errorf("%w: %s has unknown edge %q", ErrInvalidCue, definition.ID, definition.Edge)
	}
	if definition.Cooldown < 0 {
		return Cue{}, fmt.Errorf("%w: %s has a negative cooldown", ErrInvalidCue, definition.ID)
	}

	copied := make(map[string]string, len(definition.Match))
	for key, value := range definition.Match {
		copied[key] = value
	}

	return Cue{
		id:       ID(definition.ID),
		source:   source,
		name:     name,
		edge:     edge,
		match:    copied,
		priority: priority,
		cooldown: definition.Cooldown,
		purpose:  definition.Purpose,
	}, nil
}

// ID returns the cue's identifier.
func (c Cue) ID() ID { return c.id }

// Title returns the cue in words, for a reader rather than for the code.
func (c Cue) Title() string { return c.id.Title() }

// Purpose returns when the cue is heard, as the cue table writes it (FR-231). It is the one
// piece of reader facing text written by hand rather than generated from the id.
func (c Cue) Purpose() string { return c.purpose }

// Source returns which event source can drive this cue.
func (c Cue) Source() event.Source { return c.source }

// Name returns the journal event or status flag the cue listens for.
func (c Cue) Name() string { return c.name }

// Priority returns how urgent the cue is.
func (c Cue) Priority() Priority { return c.priority }

// Cooldown returns the minimum interval between two firings of this cue.
func (c Cue) Cooldown() time.Duration { return c.cooldown }

// Specificity reports how many payload fields the cue constrains. A cue matching
// more fields describes a narrower situation, so it wins over a broader one.
func (c Cue) Specificity() int { return len(c.match) }

// Matches reports whether an event belongs to this cue.
func (c Cue) Matches(candidate event.Event) bool {
	if candidate.Source() != c.source || candidate.Name() != c.name {
		return false
	}
	if c.edge != event.EdgeNone && candidate.Edge() != c.edge {
		return false
	}
	for key, want := range c.match {
		got, ok := candidate.Field(key)
		if !ok || !equalField(got, want) {
			return false
		}
	}
	return true
}

// equalField compares a payload value against the cue table's textual expectation.
// The table is text, so every comparison is made in the value's rendered form.
func equalField(got any, want string) bool {
	switch typed := got.(type) {
	case string:
		return typed == want
	case bool:
		return fmt.Sprintf("%t", typed) == want
	case float64:
		return fmt.Sprintf("%g", typed) == want || fmt.Sprintf("%.0f", typed) == want
	default:
		return fmt.Sprintf("%v", typed) == want
	}
}
