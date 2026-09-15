// Package machinevoice holds the voices the application makes its own lines with: which are
// offered, the accent each speaks with and the name each is shown by.
package machinevoice

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrUnknownVoice is returned for an id that is not one of the voices offered.
var ErrUnknownVoice = errors.New("unknown machine voice")

// Accent is the English a voice speaks: its lines are made with that accent's pronunciation
// (FR-510).
type Accent int

const (
	// British is British English.
	British Accent = iota
	// American is American English.
	American
)

// accentWords is how each accent is written in a voice's name (FR-528).
var accentWords = [...]string{British: "British", American: "American"}

// String returns the accent as a voice's name writes it.
func (a Accent) String() string { return accentWords[a] }

// Accents returns every accent a voice may speak, British first. The slice is the caller's own.
func Accents() []Accent { return []Accent{British, American} }

// Sex is whether a voice is female or male.
type Sex int

const (
	// Female is a female voice.
	Female Sex = iota
	// Male is a male voice.
	Male
)

// sexWords is how each sex is written in a voice's name (FR-528).
var sexWords = [...]string{Female: "female", Male: "male"}

// String returns the sex as a voice's name writes it.
func (s Sex) String() string { return sexWords[s] }

// An id is written as an accent letter, a sex letter, an underscore then the name: bf_emma.
const (
	accentLetter = 0
	sexLetter    = 1
	nameStart    = len("bf_")
)

// accentOfLetter and sexOfLetter read the first two letters of an id.
var (
	accentOfLetter = map[byte]Accent{'b': British, 'a': American}
	sexOfLetter    = map[byte]Sex{'f': Female, 'm': Male}
)

// offered is every voice the Cast pane offers, in the order FR-508 lists them. The ids are
// the model's own, so each names its voice's style file.
var offered = []string{
	"bf_alice", "bf_emma", "bf_isabella", "bf_lily",
	"bm_daniel", "bm_fable", "bm_george", "bm_lewis",
	"af_alloy", "af_aoede", "af_bella", "af_heart", "af_jessica", "af_kore",
	"af_nicole", "af_nova", "af_river", "af_sarah", "af_sky",
	"am_adam", "am_echo", "am_eric", "am_fenrir", "am_liam", "am_michael",
	"am_onyx", "am_puck", "am_santa",
}

// Voice is one machine voice offered. Only All and Parse make one, so every Voice holds an
// id from the list offered.
type Voice struct {
	id string
}

// All returns every voice offered, in the order FR-508 lists them. The slice is the caller's
// own.
func All() []Voice {
	voices := make([]Voice, 0, len(offered))
	for _, id := range offered {
		voices = append(voices, Voice{id: id})
	}
	return voices
}

// Parse finds the voice offered under an id, refusing one that is not offered.
func Parse(id string) (Voice, error) {
	if !slices.Contains(offered, id) {
		return Voice{}, fmt.Errorf("%w: %q", ErrUnknownVoice, id)
	}
	return Voice{id: id}, nil
}

// ID returns the model's id for the voice, such as bf_emma.
func (v Voice) ID() string { return v.id }

// Accent returns the English the voice speaks, read from its id (FR-510).
func (v Voice) Accent() Accent { return accentOfLetter[v.id[accentLetter]] }

// Sex returns whether the voice is female or male, read from its id.
func (v Voice) Sex() Sex { return sexOfLetter[v.id[sexLetter]] }

// Name returns the voice as the screen shows it: the name in its id capitalised, then its
// accent and sex in brackets, such as "Emma (British, female)" (FR-528).
func (v Voice) Name() string {
	return fmt.Sprintf("%s (%s)", v.Given(), v.Group())
}

// Given returns the name in the voice's id alone, capitalised, such as "Emma": what its pill shows
// beneath its group's heading (FR-720).
func (v Voice) Given() string {
	name := v.id[nameStart:]
	return strings.ToUpper(name[:1]) + name[1:]
}

// Group returns the accent and sex the voice is offered under, such as "British, female": the
// heading of the panel it sits in (FR-720).
func (v Voice) Group() string {
	return fmt.Sprintf("%s, %s", v.Accent(), v.Sex())
}
