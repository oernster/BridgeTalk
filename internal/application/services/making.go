package services

import (
	"context"
	"slices"
	"sync"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// LineFailure is a line that could not be made: its cue, its place among that cue's lines and why
// (FR-518).
type LineFailure struct {
	Cue    cue.ID
	Index  int
	Reason error
}

// Progress is how far making a machine voice's lines has got, as the Cast pane shows it.
type Progress struct {
	// Voice is the cast machine voice's id; empty while no machine voice is cast.
	Voice string
	// Making reports whether lines are being made now.
	Making bool
	// Current counts the voice's lines with a current made line; Total counts all of them (FR-515).
	Current, Total int
	// CuesServed counts the cues with at least one current made line (FR-522).
	CuesServed int
	// Failed lists the lines that could not be made, in the order making met them (FR-518).
	Failed []LineFailure
	// Stopped is why making stopped short: a made line that could not be written (FR-520).
	Stopped error
	// NotDeleted is why the last cast could not delete the voice's lines no longer current (FR-530).
	NotDeleted error
}

// MakingService makes a machine voice's lines when they are first needed and answers its cues with
// the lines made so far.
//
// A cast makes its confirmation's lines (FR-511); every other line is made when its cue asks for it
// through MakeNext (FR-514). Making one line takes hundreds of milliseconds, so lines are made on a
// goroutine of their own while the Cast pane reads Progress. One run makes at a time; a cast stops the
// run under way and waits for it to end before anything else changes.
type MakingService struct {
	voiced       script.Voiced
	confirmation cue.ID
	files        ports.VoiceFiles
	maker        ports.SpeechMaker
	store        ports.MadeLines

	// casting holds one cast at a time, so a cast from the window and one from the tray cannot
	// interleave.
	casting sync.Mutex

	mu         sync.Mutex
	generation int
	cast       bool
	voice      machinevoice.Voice
	style      speech.Style
	plan       making.Plan
	// queue is the lines still to make, in the order they are made; MakeNext reorders it.
	queue []making.Line
	// ctx ends when the cast is stopped; cancel ends it, always under mu.
	ctx        context.Context
	cancel     context.CancelFunc
	running    *run
	making     bool
	failed     []LineFailure
	stopped    error
	notDeleted error
}

// run is one making under way and when it has ended.
type run struct {
	done chan struct{}
}

// NewMakingService makes lines from the voiced script through the ports given; confirmation is the id
// of the cue played on a cast (cue.Table.Confirmation).
func NewMakingService(
	voiced script.Voiced, confirmation cue.ID, files ports.VoiceFiles, maker ports.SpeechMaker, store ports.MadeLines,
) *MakingService {
	return &MakingService{voiced: voiced, confirmation: confirmation, files: files, maker: maker, store: store}
}

// Cast casts a machine voice, which casting at start does as well (FR-511, FR-512), making the
// confirmation's lines that are not current and loading the model (FR-544). It answers the audio source that plays the voice's
// current made lines and makes a cue's lines on call (FR-514).
//
// A voice whose files cannot be read is refused before anything changes (FR-519). Otherwise the run
// under way stops, keeping what it wrote (FR-516). Every voice keeps its made lines; this voice's lines
// no longer current are deleted, a failure to delete being reported rather than stopping the cast
// (FR-527, FR-530).
func (m *MakingService) Cast(voice machinevoice.Voice) (ports.AudioSource, error) {
	m.casting.Lock()
	defer m.casting.Unlock()

	material, err := m.files.Open(voice)
	if err != nil {
		return nil, err
	}
	m.endRun(true)
	plan := making.New(m.voiced, voice.Accent(), material.Files, m.store.Keys(voice))
	var notDeleted error
	if stale := plan.Stale(); len(stale) > 0 {
		notDeleted = m.store.Delete(voice, stale)
	}
	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	defer m.mu.Unlock()
	m.generation++
	m.cast, m.voice, m.style, m.plan, m.queue = true, voice, material.Style, plan, nil
	m.ctx, m.cancel, m.running, m.making = ctx, cancel, nil, false
	m.failed, m.stopped, m.notDeleted = nil, nil, notDeleted
	m.makeNext(m.confirmation)
	// The cast does not wait for the model; a load that fails is tried again by the first line, which
	// reports why (FR-518).
	go func() { _ = m.maker.Load(ctx) }()
	return madeVoice{service: m, generation: m.generation}, nil
}

// CastRecorded is told a recorded voice is cast: making stops (FR-516); every made line is kept
// (FR-527).
func (m *MakingService) CastRecorded() {
	m.casting.Lock()
	defer m.casting.Unlock()

	m.endRun(true)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.generation++
	m.cast, m.plan, m.queue = false, making.Plan{}, nil
	m.failed, m.stopped, m.notDeleted = nil, nil, nil
}

// Stop stops the run under way, keeping every line it wrote, as closing the application does. No line
// is made for the cast after it.
func (m *MakingService) Stop() {
	m.casting.Lock()
	defer m.casting.Unlock()
	m.endRun(true)
}

// Progress reports how far making has got.
func (m *MakingService) Progress() Progress {
	m.mu.Lock()
	defer m.mu.Unlock()
	progress := Progress{
		Making: m.making, Failed: slices.Clone(m.failed), Stopped: m.stopped, NotDeleted: m.notDeleted,
	}
	if m.cast {
		progress.Voice = m.voice.ID()
		progress.Current, progress.Total, progress.CuesServed = m.plan.Current(), m.plan.Total(), m.plan.CuesServed()
	}
	return progress
}

// endRun waits for the run under way to end, stopping the cast first where stopping is asked. The
// cast is stopped under the lock, so MakeNext never starts a run for it afterwards.
func (m *MakingService) endRun(stopping bool) {
	m.mu.Lock()
	running := m.running
	if stopping && m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()
	if running != nil {
		<-running.done
	}
}

// makeNext puts a cue's unmade lines at the front of the queue, first line first, so they are made
// after the line under way, then starts a run where none is going (FR-511, FR-514). It answers whether
// a line is on its way: false where the cast is stopped, where a write has failed (FR-520) or where
// the cue has none to make. The caller holds mu.
func (m *MakingService) makeNext(id cue.ID) bool {
	if m.ctx.Err() != nil || m.stopped != nil {
		return false
	}
	unmade := m.plan.Unmade(id)
	if len(unmade) == 0 {
		return false
	}
	rest := slices.DeleteFunc(slices.Clone(m.queue), func(line making.Line) bool { return line.Cue == id })
	m.queue = append(unmade, rest...)
	if !m.making {
		m.making = true
		m.running = &run{done: make(chan struct{})}
		go m.makeLines(m.ctx, m.running, m.voice, m.style)
	}
	return true
}

// makeLines makes each line the queue gives in turn, recording what happens for Progress. Wherever it
// ends, it clears making under mu at the moment it sees why, so a run MakeNext starts after that is
// never taken for this one ending.
func (m *MakingService) makeLines(ctx context.Context, running *run, voice machinevoice.Voice, style speech.Style) {
	defer close(running.done)
	for {
		line, ok := m.next()
		if !ok {
			return
		}
		samples, err := m.makeLine(ctx, style, line)
		if ctx.Err() != nil {
			m.mu.Lock()
			m.making = false
			m.mu.Unlock()
			return
		}
		if err != nil {
			m.mu.Lock()
			m.failed = append(m.failed, LineFailure{Cue: line.Cue, Index: line.Index, Reason: err})
			m.mu.Unlock()
			continue
		}
		if err := m.store.Write(voice, line.Key, samples); err != nil {
			m.mu.Lock()
			m.stopped, m.making = err, false
			m.mu.Unlock()
			return
		}
		m.mu.Lock()
		m.plan = m.plan.WithMade(line.Key)
		m.mu.Unlock()
	}
}

// makeLine makes one line's samples: its numbers, its style row, then the model.
func (m *MakingService) makeLine(ctx context.Context, style speech.Style, line making.Line) ([]float32, error) {
	// The script was voiced, which refused any sounds the model cannot read (FR-506), so the
	// numbers are always had.
	tokens, _ := speech.Tokens(line.Sounds)
	row, err := style.For(tokens)
	if err != nil {
		return nil, err
	}
	return m.maker.Make(ctx, tokens, row)
}

// next takes the next line to make off the queue. A line whose made line is already current is passed
// over: one sharing its sounds with a line made before it; one MakeNext put ahead twice. With nothing
// left, the run is over; that is recorded under the same lock, so a cue asked for afterwards starts a
// run of its own.
func (m *MakingService) next() (making.Line, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for len(m.queue) > 0 {
		line := m.queue[0]
		m.queue = m.queue[1:]
		if !m.plan.Made(line.Key) {
			return line, true
		}
	}
	m.making = false
	return making.Line{}, false
}

// madeVoice is the audio source for one cast of a machine voice. It answers from the lines made so
// far; once another voice is cast it answers nothing (FR-514).
type madeVoice struct {
	service    *MakingService
	generation int
}

// MakeNext has a cue's unmade lines made next (FR-514). It answers whether a line is on its way: false
// where the cue has none to make, where making was stopped or where this cast is over.
func (v madeVoice) MakeNext(id cue.ID) bool {
	m := v.service
	m.mu.Lock()
	defer m.mu.Unlock()
	if v.generation != m.generation {
		return false
	}
	return m.makeNext(id)
}

// Lookup returns where a cue's current made lines are played from; false where there are none or
// the cast it belongs to is over.
func (v madeVoice) Lookup(id cue.ID) ([]string, bool) {
	m := v.service
	m.mu.Lock()
	defer m.mu.Unlock()
	if v.generation != m.generation {
		return nil, false
	}
	keys := m.plan.Takes(id)
	if len(keys) == 0 {
		return nil, false
	}
	paths := make([]string, 0, len(keys))
	for _, key := range keys {
		paths = append(paths, m.store.Path(m.voice, key))
	}
	return paths, true
}
