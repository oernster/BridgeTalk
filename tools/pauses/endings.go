package main

import (
	"context"
	"time"

	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/measured"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// Every setting of the fade after a final nasal has its one home here. endings.py holds none: it reads
// each from the request (FR-555, FR-556).
const (
	// fadeLength is how long a fade lasts, ending where the hiss starts, chosen by ear (FR-556).
	fadeLength = 30 * time.Millisecond
	// burstFrame is the length of each frame a burst is read in (FR-555).
	burstFrame = 10 * time.Millisecond
	// hissFrame is the length of each frame the start of the hiss is read in, over the burst frame
	// before a burst (FR-555).
	hissFrame = 500 * time.Microsecond
	// burstWithin is the longest a burst's last frame may end before the line's last loud frame ends
	// (FR-555).
	burstWithin = 50 * time.Millisecond
	// burstHighHz is the frequency above which a burst frame holds its share of energy (FR-555).
	burstHighHz = 3000
	// burstLoudDB is the loudness a frame must exceed to be read at all, in decibels (FR-555).
	burstLoudDB = -50
	// burstShare is the least share of a frame's energy above burstHighHz that makes it a burst frame
	// (FR-555).
	burstShare = 0.4
)

// fadedWord is what the tool prints of the lines given a fade (FR-555).
const fadedWord = "faded"

// endingRequest is what endings.py is handed for one voice: every setting, then the voice's lines
// ending on a nasal written as WAV files in order. Frames and spans are counted in samples.
type endingRequest struct {
	// Frame is the length of a frame in samples.
	Frame int `json:"frame"`
	// HissFrame is the length in samples of each frame the start of the hiss is read in.
	HissFrame int `json:"hiss_frame"`
	// HighHz is the frequency above which a burst frame holds its share of energy.
	HighHz float64 `json:"high_hz"`
	// LoudDB is the loudness in decibels a frame must exceed to be read.
	LoudDB float64 `json:"loud_db"`
	// Share is the least share of a frame's energy above HighHz that makes it a burst frame.
	Share float64 `json:"share"`
	// Within is the most samples a burst may end before the line's last loud frame ends.
	Within int `json:"within"`
	// Files is each line's WAV file, in the order the script lists the lines.
	Files []string `json:"files"`
}

// ended is endings.py's answer for one file: the sample the hiss starts at, nil where the line ends on
// no burst.
type ended struct {
	// Hiss is the sample the hiss starts at.
	Hiss *int `json:"hiss"`
}

// endingFinder finds where each of a voice's lines written as WAV files ends on a burst, answering in
// the same order.
type endingFinder interface {
	FindEndings(asked endingRequest) ([]ended, error)
}

// endingRequestFor asks where a voice's lines end on a burst.
func endingRequestFor(files []string) endingRequest {
	return endingRequest{
		Frame: samplesIn(burstFrame), HissFrame: samplesIn(hissFrame), HighHz: burstHighHz, LoudDB: burstLoudDB, Share: burstShare,
		Within: samplesIn(burstWithin), Files: files,
	}
}

// endings finds the endings of each voice over the lines ending on a nasal in its accent, printing
// each voice then the total, then builds the book (FR-555).
func (f finding) endings(ctx context.Context, voiced script.Voiced, voices []machinevoice.Voice) (ending.Book, error) {
	faded, model, err := eachVoice(f, voices, fadedWord, func(voice machinevoice.Voice) (measuredVoice[ending.Voice], error) {
		return f.endingsOf(ctx, voice, voiced.EndingOnNasal(voice.Accent()))
	})
	if err != nil {
		return ending.Book{}, err
	}
	return ending.NewBook(samplesIn(fadeLength), model, faded)
}

// endingsOf makes each of a voice's lines ending on a nasal for the ending finder, building the voice's
// endings from what it answers and printing the voice's faded lines. Each fade starts its length before
// the sample the hiss starts at, so it ends where the hiss starts (FR-556).
func (f finding) endingsOf(ctx context.Context, voice machinevoice.Voice, lines []script.SavedLine) (measuredVoice[ending.Voice], error) {
	var answers []ended
	made, err := f.makeFor(ctx, voice, lines, func(files []string) (int, error) {
		var err error
		answers, err = f.ends.FindEndings(endingRequestFor(files))
		return len(answers), err
	})
	if err != nil {
		return measuredVoice[ending.Voice]{}, err
	}
	entries := make([]ending.Entry, 0, len(lines))
	var listed []string
	for at, line := range lines {
		entry := ending.Entry{Cue: line.Cue, Index: line.Index, Sounds: line.Sounds, Digest: made.digests[at]}
		if hiss := answers[at].Hiss; hiss != nil {
			start := *hiss - samplesIn(fadeLength)
			if err := within(start, made.lengths[at], voice, line); err != nil {
				return measuredVoice[ending.Voice]{}, err
			}
			entry.Sample = start
			listed = append(listed, measured.LineName(voice.ID(), line.Cue, line.Index))
		}
		entries = append(entries, entry)
	}
	f.list(voice, fadedWord, listed, len(entries))
	return measuredVoice[ending.Voice]{
		entries: ending.Voice{Style: made.from.Style, Entries: entries}, model: made.from.Model, lines: len(entries), marked: len(listed),
	}, nil
}
