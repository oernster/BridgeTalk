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
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/measured"
	"github.com/oernster/bridge-talk/internal/domain/pause"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

var (
	// errAnswerCount means a finder answered a different number of lines than it was handed, so its
	// answers could no longer be matched to lines.
	errAnswerCount = errors.New("a finder answered a different number of lines")
	// errCutOutside means a finder answered a sample outside the samples of its line.
	errCutOutside = errors.New("a finder answered a sample outside its line")
	// errModelChanged means the model file's digest changed between voices, so no one model matches
	// every entry.
	errModelChanged = errors.New("the model file changed during the run")
)

// wavName names the file a line is written to for a finder, by its place among the voice's lines.
const wavName = "line-%d.wav"

// doubtfulWord is what the tool prints of the lines whose break was doubtful (FR-552).
const doubtfulWord = "doubtful"

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

// finding finds every voice's pauses and endings, writing each voice's lines in a folder under work
// for the finders and printing what each voice's lines are marked as to out.
type finding struct {
	files  opener
	maker  maker
	finder finder
	ends   endingFinder
	work   string
	out    io.Writer
}

// madeLines is one voice's lines as a finder was handed them: the digest and the length of each line's
// samples in order, with the digests of the files they were made from.
type madeLines struct {
	digests []string
	lengths []int
	from    making.Files
}

// measuredVoice is what measuring one voice answers: its entries, the digest of the model its lines
// were made with, how many lines it has and how many of them were marked.
type measuredVoice[V any] struct {
	entries V
	model   string
	lines   int
	marked  int
}

// eachVoice measures each voice in turn, holding every voice to the model the first was made with,
// then prints how many lines over every voice were marked as said. It answers each voice's entries by
// id with the model's digest.
func eachVoice[V any](f finding, voices []machinevoice.Voice, said string, measure func(machinevoice.Voice) (measuredVoice[V], error)) (map[string]V, string, error) {
	entries := make(map[string]V, len(voices))
	model := ""
	lines, marked := 0, 0
	for _, voice := range voices {
		one, err := measure(voice)
		if err != nil {
			return nil, "", err
		}
		if model != "" && one.model != model {
			return nil, "", fmt.Errorf("%w: %s was made with %s, the voices before it with %s", errModelChanged, voice.ID(), one.model, model)
		}
		model = one.model
		entries[voice.ID()] = one.entries
		lines += one.lines
		marked += one.marked
	}
	fmt.Fprintf(f.out, "%d voices: %d of %d lines %s\n", len(voices), marked, lines, said)
	return entries, model, nil
}

// book finds the pauses of each voice over the lines the script joins in its accent, printing each
// voice then the total, then builds the book (FR-551, FR-552).
func (f finding) book(ctx context.Context, voiced script.Voiced, voices []machinevoice.Voice) (pause.Book, error) {
	paused, model, err := eachVoice(f, voices, doubtfulWord, func(voice machinevoice.Voice) (measuredVoice[pause.Voice], error) {
		return f.voice(ctx, voice, voiced.Joined(voice.Accent()))
	})
	if err != nil {
		return pause.Book{}, err
	}
	return pause.NewBook(samplesIn(silence), model, paused)
}

// voice makes each of a voice's joined lines for the break finder, building the voice's pauses from
// what it answers and printing the voice's doubtful lines.
func (f finding) voice(ctx context.Context, voice machinevoice.Voice, lines []script.SavedLine) (measuredVoice[pause.Voice], error) {
	var answers []found
	made, err := f.makeFor(ctx, voice, lines, func(files []string) (int, error) {
		var err error
		answers, err = f.finder.Find(requestFor(voice, files))
		return len(answers), err
	})
	if err != nil {
		return measuredVoice[pause.Voice]{}, err
	}
	entries := make([]pause.Entry, 0, len(lines))
	var listed []string
	for at, doubtful := range judged(answers) {
		line := lines[at]
		if cut := answers[at].Cut; cut != nil {
			if err := within(*cut, made.lengths[at], voice, line); err != nil {
				return measuredVoice[pause.Voice]{}, err
			}
		}
		entry := pause.Entry{Cue: line.Cue, Index: line.Index, Sounds: line.Sounds, Digest: made.digests[at], Doubtful: doubtful}
		if doubtful {
			listed = append(listed, measured.LineName(voice.ID(), line.Cue, line.Index))
		} else {
			entry.Sample = *answers[at].Cut
		}
		entries = append(entries, entry)
	}
	f.list(voice, doubtfulWord, listed, len(entries))
	return measuredVoice[pause.Voice]{
		entries: pause.Voice{Style: made.from.Style, Entries: entries}, model: made.from.Model, lines: len(entries), marked: len(listed),
	}, nil
}

// makeFor makes each of a voice's lines, digests its samples then writes it in a folder of its own
// under work, handing every file to ask, which answers how many lines it answered for; the files are
// removed once it has answered (FR-551, FR-555).
func (f finding) makeFor(ctx context.Context, voice machinevoice.Voice, lines []script.SavedLine, ask func(files []string) (int, error)) (madeLines, error) {
	material, err := f.files.Open(voice)
	if err != nil {
		return madeLines{}, err
	}
	dir, err := os.MkdirTemp(f.work, voice.ID()+"-*")
	if err != nil {
		return madeLines{}, fmt.Errorf("making a folder for %s's lines: %w", voice.ID(), err)
	}
	defer os.RemoveAll(dir)
	made := madeLines{from: material.Files}
	files := make([]string, 0, len(lines))
	for at, line := range lines {
		samples, err := f.samples(ctx, material.Style, line)
		if err != nil {
			return madeLines{}, fmt.Errorf("making %s: %w", measured.LineName(voice.ID(), line.Cue, line.Index), err)
		}
		path := filepath.Join(dir, fmt.Sprintf(wavName, at))
		if err := writeWAV(path, samples); err != nil {
			return madeLines{}, err
		}
		made.digests = append(made.digests, pause.Digest(samples))
		made.lengths = append(made.lengths, len(samples))
		files = append(files, path)
	}
	answered, err := ask(files)
	if err != nil {
		return madeLines{}, fmt.Errorf("finding in %s's lines: %w", voice.ID(), err)
	}
	if answered != len(files) {
		return madeLines{}, fmt.Errorf("%w: %d for %s's %d lines", errAnswerCount, answered, voice.ID(), len(files))
	}
	return made, nil
}

// within refuses a sample a finder answered for a line where it is not inside the line: at its first
// sample or at or beyond its end.
func within(sample, length int, voice machinevoice.Voice, line script.SavedLine) error {
	if sample <= 0 || sample >= length {
		return fmt.Errorf("%w: sample %d of the %d in %s", errCutOutside, sample, length, measured.LineName(voice.ID(), line.Cue, line.Index))
	}
	return nil
}

// samples makes one line's samples from its saved speech sounds with the voice's style.
func (f finding) samples(ctx context.Context, style speech.Style, line script.SavedLine) ([]float32, error) {
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

// list prints how many of a voice's lines are marked as said, then names each by voice, cue and line
// (FR-552, FR-555).
func (f finding) list(voice machinevoice.Voice, said string, names []string, lines int) {
	fmt.Fprintf(f.out, "%s: %d of %d lines %s\n", voice.ID(), len(names), lines, said)
	for _, name := range names {
		fmt.Fprintf(f.out, "  %s\n", name)
	}
}
