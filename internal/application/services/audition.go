package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/selection"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// ErrNothingToAudition is returned for a group the script holds no lines for.
var ErrNothingToAudition = errors.New("the script holds no lines for that group")

// AuditionGroups returns the groups a machine voice is auditioned on: the script's, each counting its
// lines (FR-546). Every machine voice speaks the one script, so the answer holds for each of them.
func (m *MakingService) AuditionGroups() []script.Group { return m.voiced.Groups() }

// Audition answers where to play one line of a group, drawn at random, for a machine voice cast or not
// (FR-546). A line with no current made line is made first, next after the line under way, then kept
// (FR-527). Files that cannot be read, a line the model refuses and a line that cannot be written each
// answer why, keeping nothing (FR-548).
//
// The voice's files are read on every audition as a cast reads them, so a line made from files that
// have changed since is never taken for current (FR-513).
func (m *MakingService) Audition(voice machinevoice.Voice, group string, chooser selection.Chooser) (string, error) {
	material, err := m.files.Open(voice)
	if err != nil {
		return "", err
	}
	plan := making.New(m.voiced, voice, material.Files, m.pauses, m.store.Keys(voice))
	lines := plan.Group(group)
	if len(lines) == 0 {
		return "", fmt.Errorf("%w: %s", ErrNothingToAudition, group)
	}
	line := lines[chooser.Intn(len(lines))]
	if !plan.Made(line.Key) {
		if err := m.makeForAudition(voice, material.Style, line); err != nil {
			return "", err
		}
	}
	return m.store.Path(voice, line.Key), nil
}

// makeForAudition makes and writes one line on the model's next turn, then counts it as made where its
// voice is the one cast, so the cast plays it and never makes it again.
func (m *MakingService) makeForAudition(voice machinevoice.Voice, style speech.Style, line making.Line) error {
	m.turn.takeAhead()
	defer m.turn.give()
	samples, err := m.makeLine(context.Background(), voice, style, line)
	if err != nil {
		return err
	}
	if err := m.store.Write(voice, line.Key, samples); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cast && m.voice.ID() == voice.ID() {
		m.plan = m.plan.WithMade(line.Key)
	}
	return nil
}
