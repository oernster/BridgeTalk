package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/pause"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

var (
	// errAnswerCount means the break finder answered a different number of lines than it was handed,
	// so its answers could no longer be matched to lines.
	errAnswerCount = errors.New("the break finder answered a different number of lines")
	// errCutOutside means the break finder answered a cut outside the samples of its line.
	errCutOutside = errors.New("the break finder answered a cut outside its line")
	// errModelChanged means the model file's digest changed between voices, so no one model matches
	// every pause.
	errModelChanged = errors.New("the model file changed during the run")
)

// wavName names the file a line is written to for the finder, by its place among the voice's lines.
const wavName = "line-%d.wav"

// maker makes a line's samples with the model from its numbers and its style row.
type maker interface {
	Make(ctx context.Context, tokens []int64, style []float32) ([]float32, error)
}

// opener reads what a voice's lines are made from: its style with the digests of its style file and
// the model.
type opener interface {
	Open(voice machinevoice.Voice) (ports.Material, error)
}

// finder finds the break in each of a voice's lines written as WAV files, answering in the same order.
type finder interface {
	Find(asked request) ([]found, error)
}

// finding finds every voice's pauses, writing each voice's lines in a folder under work for the
// finder and printing how many of each voice's lines are doubtful to out.
type finding struct {
	files  opener
	maker  maker
	finder finder
	work   string
	out    io.Writer
}

// book finds the pauses of each voice over the lines the script joins in its accent, printing each
// voice then the total, then builds the book (FR-551, FR-552).
func (f finding) book(ctx context.Context, voiced script.Voiced, voices []machinevoice.Voice) (pause.Book, error) {
	paused := make(map[string]pause.Voice, len(voices))
	model := ""
	lines, doubtful := 0, 0
	for _, voice := range voices {
		found, digest, err := f.voice(ctx, voice, voiced.Joined(voice.Accent()))
		if err != nil {
			return pause.Book{}, err
		}
		if model != "" && digest != model {
			return pause.Book{}, fmt.Errorf("%w: %s was made with %s, the voices before it with %s", errModelChanged, voice.ID(), digest, model)
		}
		model = digest
		paused[voice.ID()] = found
		lines += len(found.Entries)
		doubtful += f.report(voice, found)
	}
	fmt.Fprintf(f.out, "%d voices: %d of %d lines doubtful\n", len(voices), doubtful, lines)
	return pause.NewBook(samplesIn(silence), model, paused)
}

// voice makes each of a voice's joined lines, digests its samples then writes it for the finder,
// building the voice's pauses from what the finder answers. It answers the model's digest beside them.
func (f finding) voice(ctx context.Context, voice machinevoice.Voice, lines []script.JoinedLine) (pause.Voice, string, error) {
	material, err := f.files.Open(voice)
	if err != nil {
		return pause.Voice{}, "", err
	}
	dir, err := os.MkdirTemp(f.work, voice.ID()+"-*")
	if err != nil {
		return pause.Voice{}, "", fmt.Errorf("making a folder for %s's lines: %w", voice.ID(), err)
	}
	defer os.RemoveAll(dir)
	entries := make([]pause.Entry, 0, len(lines))
	lengths := make([]int, 0, len(lines))
	files := make([]string, 0, len(lines))
	for at, line := range lines {
		samples, err := f.samples(ctx, material.Style, line)
		if err != nil {
			return pause.Voice{}, "", fmt.Errorf("making %s: %w", pause.LineName(voice.ID(), line.Cue, line.Index), err)
		}
		path := filepath.Join(dir, fmt.Sprintf(wavName, at))
		if err := writeWAV(path, samples); err != nil {
			return pause.Voice{}, "", err
		}
		entries = append(entries, pause.Entry{Cue: line.Cue, Index: line.Index, Sounds: line.Sounds, Digest: pause.Digest(samples)})
		lengths = append(lengths, len(samples))
		files = append(files, path)
	}
	answers, err := f.finder.Find(requestFor(voice, files))
	if err != nil {
		return pause.Voice{}, "", fmt.Errorf("finding the breaks in %s's lines: %w", voice.ID(), err)
	}
	if len(answers) != len(files) {
		return pause.Voice{}, "", fmt.Errorf("%w: %d for %s's %d lines", errAnswerCount, len(answers), voice.ID(), len(files))
	}
	for at, doubtful := range judged(answers) {
		if cut := answers[at].Cut; cut != nil && (*cut <= 0 || *cut >= lengths[at]) {
			return pause.Voice{}, "", fmt.Errorf(
				"%w: sample %d of the %d in %s", errCutOutside, *cut, lengths[at], pause.LineName(voice.ID(), lines[at].Cue, lines[at].Index),
			)
		}
		entries[at].Doubtful = doubtful
		if !doubtful {
			entries[at].Sample = *answers[at].Cut
		}
	}
	return pause.Voice{Style: material.Files.Style, Entries: entries}, material.Files.Model, nil
}

// samples makes one line's samples from its saved speech sounds with the voice's style.
func (f finding) samples(ctx context.Context, style speech.Style, line script.JoinedLine) ([]float32, error) {
	tokens, err := speech.Tokens(line.Sounds)
	if err != nil {
		return nil, err
	}
	row, err := style.For(tokens)
	if err != nil {
		return nil, err
	}
	return f.maker.Make(ctx, tokens, row)
}

// judged answers which of a voice's answers are doubtful (FR-552): one with no cut or no final, then
// one whose final lies too far from the median over the finals that were found.
func judged(answers []found) []bool {
	doubtful := make([]bool, len(answers))
	var finals []float64
	var places []int
	for at, answer := range answers {
		if answer.Cut == nil || answer.Final == nil {
			doubtful[at] = true
			continue
		}
		finals = append(finals, *answer.Final)
		places = append(places, at)
	}
	for at, far := range pause.Doubtful(finals) {
		doubtful[places[at]] = far
	}
	return doubtful
}

// report prints how many of a voice's lines are doubtful, then lists each by voice, cue and line
// (FR-552). It answers how many there were.
func (f finding) report(voice machinevoice.Voice, paused pause.Voice) int {
	var listed []string
	for _, entry := range paused.Entries {
		if entry.Doubtful {
			listed = append(listed, pause.LineName(voice.ID(), entry.Cue, entry.Index))
		}
	}
	fmt.Fprintf(f.out, "%s: %d of %d lines doubtful\n", voice.ID(), len(listed), len(paused.Entries))
	for _, name := range listed {
		fmt.Fprintf(f.out, "  %s\n", name)
	}
	return len(listed)
}
