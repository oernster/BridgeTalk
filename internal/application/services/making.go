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
	// NotDeleted is why the last cast could not delete the made lines it had to (FR-530).
	NotDeleted error
}

// MakingService makes a machine voice's lines ahead of play and answers its cues with the lines
// made so far.
//
// Making one line takes hundreds of milliseconds, so a cast starts making on a goroutine of its own
// and returns while the Cast pane reads Progress. One making runs at a time: a cast stops the one
// under way and waits for it to end before anything else changes, so no line is written after the
// lines it belongs with are deleted.
type MakingService struct {
	voiced script.Voiced
	files  ports.VoiceFiles
	maker  ports.SpeechMaker
	store  ports.MadeLines

	// casting holds one cast at a time, so a cast from the window and one from the tray cannot
	// interleave.
	casting sync.Mutex

	mu         sync.Mutex
	generation int
	cast       bool
	voice      machinevoice.Voice
	plan       making.Plan
	running    *run
	making     bool
	failed     []LineFailure
	stopped    error
	notDeleted error
}

// run is one making under way: how to stop it and when it has ended.
type run struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// NewMakingService makes lines from the voiced script through the ports given.
func NewMakingService(
	voiced script.Voiced, files ports.VoiceFiles, maker ports.SpeechMaker, store ports.MadeLines,
) *MakingService {
	return &MakingService{voiced: voiced, files: files, maker: maker, store: store}
}

// Cast casts a machine voice and starts making every line it has no current made line for, which
// casting at start does as well (FR-511, FR-512). It answers the audio source that plays the
// voice's current made lines (FR-514).
//
// A voice whose files cannot be read is refused before anything changes (FR-519). Otherwise the
// making under way stops, keeping what it wrote (FR-516). Every other voice's made lines are then
// deleted; a failure to delete is reported rather than stopping the cast (FR-527, FR-530).
func (m *MakingService) Cast(voice machinevoice.Voice) (ports.AudioSource, error) {
	m.casting.Lock()
	defer m.casting.Unlock()

	material, err := m.files.Open(voice)
	if err != nil {
		return nil, err
	}
	m.endRun(true)
	notDeleted := m.store.DeleteAllBut(voice)
	plan := making.New(m.voiced, voice.Accent(), material.Files, m.store.Keys(voice))
	ctx, cancel := context.WithCancel(context.Background())
	started := &run{cancel: cancel, done: make(chan struct{})}

	m.mu.Lock()
	m.generation++
	source := madeVoice{service: m, generation: m.generation}
	m.cast, m.voice, m.plan, m.running = true, voice, plan, started
	m.making, m.failed, m.stopped, m.notDeleted = true, nil, nil, notDeleted
	m.mu.Unlock()

	go m.makeLines(ctx, started, voice, material.Style, plan.ToMake())
	return source, nil
}

// CastRecorded is told a recorded voice is cast: making stops (FR-516) and every made line is
// deleted; a failure to delete is reported (FR-527, FR-530).
func (m *MakingService) CastRecorded() {
	m.casting.Lock()
	defer m.casting.Unlock()

	m.endRun(true)
	notDeleted := m.store.DeleteAll()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.generation++
	m.cast, m.plan = false, making.Plan{}
	m.failed, m.stopped, m.notDeleted = nil, nil, notDeleted
}

// Stop stops the making under way, keeping every line it wrote, as closing the application does.
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

// endRun waits for the making under way to end, stopping it first where stopping is asked.
func (m *MakingService) endRun(stopping bool) {
	m.mu.Lock()
	running := m.running
	m.mu.Unlock()
	if running == nil {
		return
	}
	if stopping {
		running.cancel()
	}
	<-running.done
}

// makeLines makes each line in turn, recording what happens for Progress. A line that shares its
// sounds with one already made is made once.
func (m *MakingService) makeLines(
	ctx context.Context, running *run, voice machinevoice.Voice, style speech.Style, lines []making.Line,
) {
	defer m.finish(running)
	for _, line := range lines {
		if m.made(line.Key) {
			continue
		}
		samples, err := m.makeLine(ctx, style, line)
		if ctx.Err() != nil {
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
			m.stopped = err
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

// made reports whether a line's made line is current.
func (m *MakingService) made(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.plan.Made(key)
}

// finish records that a making has ended, then lets whoever waits for it go on.
func (m *MakingService) finish(running *run) {
	m.mu.Lock()
	m.making = false
	m.mu.Unlock()
	close(running.done)
}

// madeVoice is the audio source for one cast of a machine voice. It answers from the lines made so
// far; once another voice is cast it answers nothing (FR-514).
type madeVoice struct {
	service    *MakingService
	generation int
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
