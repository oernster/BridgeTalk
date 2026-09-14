package main

import (
	"time"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
)

// Every setting of the pause before commander has its one home here. pauses.py holds none: it reads
// each from the request (FR-551, FR-553).
const (
	// silence is how long the silence a pause inserts lasts, chosen by ear (FR-553).
	silence = 40 * time.Millisecond
	// timeStep is the length of each frame Praat reads voicing in (FR-551).
	timeStep = 10 * time.Millisecond
	// bridgedGap is the longest unvoiced gap counted as voiced (FR-551).
	bridgedGap = 20 * time.Millisecond
	// shortestFinal is the shortest the final voiced stretch after a break may be (FR-551).
	shortestFinal = 200 * time.Millisecond
	// quietWindow is the window whose quietest place within the break the pause goes in the middle
	// of (FR-551).
	quietWindow = 10 * time.Millisecond
)

// The bands Praat reads voicing in, by the voice's sex: the ranges of the measurements in section 6.1
// (FR-551).
const (
	femaleFloorHz   = 120
	femaleCeilingHz = 350
	maleFloorHz     = 65
	maleCeilingHz   = 200
)

// pitchRange is the band Praat reads voicing in, in hertz.
type pitchRange struct {
	floor, ceiling float64
}

// pitchRanges gives each sex its band.
var pitchRanges = map[machinevoice.Sex]pitchRange{
	machinevoice.Female: {floor: femaleFloorHz, ceiling: femaleCeilingHz},
	machinevoice.Male:   {floor: maleFloorHz, ceiling: maleCeilingHz},
}

// request is what pauses.py is handed for one voice: every setting, then the voice's lines written as
// WAV files in order. Frames are counted in the request's time step; windows in samples.
type request struct {
	// TimeStep is the length of a frame in seconds, as Praat reads it.
	TimeStep float64 `json:"time_step"`
	// Frame is the length of a frame in samples.
	Frame int `json:"frame"`
	// PitchFloor is the lowest pitch Praat reads as voiced, in hertz.
	PitchFloor float64 `json:"pitch_floor"`
	// PitchCeiling is the highest pitch Praat reads as voiced, in hertz.
	PitchCeiling float64 `json:"pitch_ceiling"`
	// Bridged is the most unvoiced frames between voiced ones counted as voiced.
	Bridged int `json:"bridged"`
	// Final is the fewest frames the final voiced stretch may hold.
	Final int `json:"final"`
	// Quiet is the length of the quiet window in samples.
	Quiet int `json:"quiet"`
	// Files is each line's WAV file, in the order the script joins the lines.
	Files []string `json:"files"`
}

// found is pauses.py's answer for one file: the sample the pause goes at and the length in seconds of
// the final voiced stretch, each nil where no break was found.
type found struct {
	// Cut is the sample the pause goes at.
	Cut *int `json:"cut"`
	// Final is the length of the final voiced stretch in seconds.
	Final *float64 `json:"final"`
}

// samplesIn answers how many samples at the model's rate a duration lasts.
func samplesIn(length time.Duration) int { return int(length * madelines.SampleRate / time.Second) }

// framesIn answers how many whole frames a duration lasts.
func framesIn(length time.Duration) int { return int(length / timeStep) }

// requestFor asks for the breaks in a voice's lines, with Praat's band for the voice's sex.
func requestFor(voice machinevoice.Voice, files []string) request {
	pitch := pitchRanges[voice.Sex()]
	return request{
		TimeStep: timeStep.Seconds(), Frame: samplesIn(timeStep),
		PitchFloor: pitch.floor, PitchCeiling: pitch.ceiling,
		Bridged: framesIn(bridgedGap), Final: framesIn(shortestFinal), Quiet: samplesIn(quietWindow),
		Files: files,
	}
}
