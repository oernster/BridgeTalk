// Package ports declares the interfaces the use cases depend on. Every one of them
// is implemented in infrastructure and injected at the composition root, so nothing
// in the application layer knows about files, audio devices or the shell.
package ports

import (
	"time"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
)

// EventSource yields the events that have happened since it was last asked.
//
// Poll returns only what is new. A source that has nothing new returns an empty
// slice and no error, which is the common case and must stay cheap.
type EventSource interface {
	Poll() ([]event.Event, error)
	Name() string
}

// AudioPlayer plays clips and reports when it has finished.
//
// Play is asynchronous: it starts the sequence and returns. Done yields exactly one
// value per completed or stopped sequence, so a scheduler can wait on it rather
// than polling. Implementations run their own output thread, so Play, Stop and
// Playing are safe to call from another goroutine.
type AudioPlayer interface {
	Play(clips []string, gap time.Duration) error
	Stop()
	Playing() bool
	Done() <-chan struct{}
	Close() error
}

// Performance is one resolved answer to a cue: every take the cast voice holds for it.
// The takes are alternatives; the reaction service picks one, so exactly one is spoken.
type Performance struct {
	Clips []string
}

// VoiceCatalogue resolves a cue into the takes the chosen voice can play for it.
//
// The boolean reports whether the voice can serve the cue at all. One that cannot is
// not an error: an unrecorded cue is silence; silence is the correct answer when
// the alternative is saying the wrong line.
type VoiceCatalogue interface {
	Clips(id cue.ID) (Performance, bool)
	ActiveVoice() string
	Coverage() (served int, total int)
}

// Clock supplies the current time, injected so cooldown and dedupe behaviour is
// reproducible under test.
type Clock interface {
	Now() time.Time
}

// Settings is what the user has chosen that outlives a run.
//
// Every field is empty until something is chosen: the application detects the two
// directories from the profile and picks the first voice it finds, recording a value
// only where the reader has said otherwise. Storing what was detected would freeze a
// guess, so a profile that moves would keep pointing at where it used to be and a
// voice that was never chosen would be honoured as though it had been.
type Settings struct {
	LibraryRoot string
	JournalDir  string

	// Voice is the voice the reader cast, by name. A name is stored rather than a
	// path because a voice is identified by its name everywhere else here, so a root
	// that moves carries the choice with it; a name that is no longer installed
	// falls back at startup the same way an unknown one on the command line does.
	Voice string
}

// SettingsStore keeps those choices between runs.
//
// Load answers with what is stored and nothing else. A first run has no file, an
// unreadable one is indistinguishable from that to a reader; neither is a reason to
// refuse to start, so there is no error to handle: an empty Settings means detect
// as usual.
type SettingsStore interface {
	Load() Settings
	Save(Settings) error
}

// Reaction is one decision the engine made, for the reaction log.
//
// It carries what the log shows and no more. The cue's source and its priority were
// filled in by both reporters and read only on the way to a wire field the page never
// rendered, so the whole chain was write-only. Both are still on the cue itself, which
// is where a reader that wants them should ask. The name of the event that raised the cue
// went the same way: every cue id begins with it, so the log only ever repeated the id
// (FR-234).
type Reaction struct {
	At      time.Time
	Cue     cue.ID
	Clip    string
	Outcome string
}

// Reporter receives every decision, including the ones that produced no sound, so
// a cue that never fires can be diagnosed by looking rather than guessing.
type Reporter interface {
	Report(reaction Reaction)
}

// Outcome values recorded on a Reaction.
const (
	// OutcomePlayed means a clip was started.
	OutcomePlayed = "played"
	// OutcomeQueued means the request is waiting behind something louder.
	OutcomeQueued = "queued"
	// OutcomeCooldown means the cue fired too recently.
	OutcomeCooldown = "cooldown"
	// OutcomeDuplicate means the same cue arrived twice inside the dedupe window.
	OutcomeDuplicate = "duplicate"
	// OutcomeUnbound means the active voice has nothing for this cue.
	OutcomeUnbound = "unbound"
	// OutcomeDropped means a low-priority request was discarded while busy.
	OutcomeDropped = "dropped"
	// OutcomeNoCue means no cue in the table claimed the event.
	OutcomeNoCue = "no cue"
)
