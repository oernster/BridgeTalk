// Package plugin reads what a plugin says, in the layouts PLUGINS-GUIDE.md states.
//
// Everything here treats a plugin's answer as foreign input. A plugin is somebody else's
// code, built at a time of their choosing against a contract they may have misread, so no
// length is trusted, no count is believed and nothing is read past the end of what arrived.
// A malformed answer is a refusal naming what was wrong with it, never a panic.
package plugin

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/oernster/bridge-talk/internal/domain/take"
)

// ErrMalformed is returned for an answer that does not read as the layout it should be.
var ErrMalformed = errors.New("the plugin's answer does not read as it should")

// intSize is the width of every count and length crossing the boundary. One width for all
// of them is what lets a reader step through an answer without a table of field sizes.
const intSize = 4

// positionSize is the width of a span's offset and length, the two numbers that measure a
// place in a file rather than in an answer: a file can be larger than a 32 bit number reaches
// (FR-588).
const positionSize = 8

// A part's kind, the number ahead of each part saying what follows it.
const (
	// partFile is a whole file: its path alone follows.
	partFile = 0
	// partSpan is a span of a file: its path, its format, its offset and its length follow.
	partSpan = 1
)

// VoiceInfo is one voice a plugin offers, as the plugin describes it.
type VoiceInfo struct {
	// ID identifies the voice within its plugin and is kept in the settings beside the
	// plugin's name, so the same voice is found again next run.
	ID string
	// Name is what the user sees.
	Name string
	// Group is the group the voice is shown in; empty for none (FR-582).
	Group string
	// Ready says whether the audio this voice needs is present on this machine.
	Ready bool
	// Reason says why it is not, in the plugin's own words; empty while Ready.
	Reason string
}

// Description is what a plugin says about itself and the voices it offers.
type Description struct {
	Name   string
	Voices []VoiceInfo
}

// reader steps through an answer, refusing anything that would read past its end.
//
// It keeps the first refusal rather than returning one from every method: a caller reads a
// whole layout and asks once whether it held together, which keeps the layout readable as
// a layout instead of as a column of error checks.
type reader struct {
	data []byte
	at   int
	err  error
}

// fail records the first thing that went wrong and nothing after it.
func (r *reader) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf("%w: "+format, append([]any{ErrMalformed}, args...)...)
	}
}

// count reads one number, which is a length, an index or a total.
func (r *reader) count(what string) int32 {
	if r.err != nil {
		return 0
	}
	if r.at+intSize > len(r.data) {
		r.fail("it ends before its %s", what)
		return 0
	}
	value := int32(binary.LittleEndian.Uint32(r.data[r.at : r.at+intSize]))
	r.at += intSize
	return value
}

// position reads one of a span's two 64 bit numbers. It is not checked here: a span that does
// not lie inside its file is a part passed over when it is played (FR-590), not an answer that
// will not read, so one bad span costs one part rather than every take of a cue.
func (r *reader) position(what string) int64 {
	if r.err != nil {
		return 0
	}
	if r.at+positionSize > len(r.data) {
		r.fail("it ends before its %s", what)
		return 0
	}
	value := int64(binary.LittleEndian.Uint64(r.data[r.at : r.at+positionSize]))
	r.at += positionSize
	return value
}

// part reads one part of a take, whose kind says whether it is a whole file or a span.
func (r *reader) part() take.Part {
	kind := r.count("part kind")
	path := r.text("part path")
	switch {
	case r.err != nil:
		return take.Part{}
	case kind == partFile:
		return take.File(path)
	case kind == partSpan:
		format := r.text("span format")
		offset := r.position("span offset")
		length := r.position("span length")
		return take.SpanOf(path, format, offset, length)
	}
	r.fail("one of its parts is of kind %d, which is neither a file nor a span", kind)
	return take.Part{}
}

// size reads a number that may not be negative and may not exceed what is left, which is
// every length and every count in these layouts.
//
// Both guards matter against foreign input: a negative would index backwards, while a count
// larger than the bytes remaining would have a reader allocate for entries that cannot be
// there. Nothing in a layout is smaller than one byte, so the bytes left are a true ceiling
// on how many entries can follow.
func (r *reader) size(what string) int32 {
	value := r.count(what)
	if r.err != nil {
		return 0
	}
	if value < 0 {
		r.fail("its %s is %d, which is not a number of anything", what, value)
		return 0
	}
	if int(value) > len(r.data)-r.at {
		r.fail("its %s is %d with %d bytes left", what, value, len(r.data)-r.at)
		return 0
	}
	return value
}

// text reads one length-prefixed string.
//
// The bytes are required to be UTF-8 because everything downstream treats them as text: a
// name is shown in the window and a path is opened. Bytes that are not UTF-8 would reach a
// user as a name nobody can read or a path nobody can find, so they are refused here where
// the plugin that sent them can still be named.
func (r *reader) text(what string) string {
	length := r.size(what + " length")
	if r.err != nil {
		return ""
	}
	value := string(r.data[r.at : r.at+int(length)])
	r.at += int(length)
	if !utf8.ValidString(value) {
		r.fail("its %s is not UTF-8", what)
		return ""
	}
	return value
}

// done reports the first refusal; failing that, it reports the whole answer as read.
//
// Bytes left over are a refusal too. They mean the answer was written to a different layout
// from the one being read; a reader that shrugged at them would accept a plugin built
// against a version it did not declare.
func (r *reader) done(what string) error {
	if r.err != nil {
		return r.err
	}
	if r.at != len(r.data) {
		return fmt.Errorf("%w: %d bytes follow the %s", ErrMalformed, len(r.data)-r.at, what)
	}
	return nil
}

// DecodeDescription reads a plugin's account of itself.
func DecodeDescription(data []byte) (Description, error) {
	r := &reader{data: data}
	voices := r.size("voice count")
	described := Description{Name: r.text("plugin name")}
	for index := int32(0); index < voices; index++ {
		voice := VoiceInfo{ID: r.text("voice id"), Name: r.text("voice name"), Group: r.text("group")}
		voice.Ready = r.count("ready flag") != 0
		voice.Reason = r.text("reason")
		if r.err != nil {
			return Description{}, r.err
		}
		described.Voices = append(described.Voices, voice)
	}
	if err := r.done("description"); err != nil {
		return Description{}, err
	}
	return described, nil
}

// DecodeTakes reads what a voice may play for one cue.
//
// A take with no parts is refused rather than dropped. A plugin that answers one is not
// answering nothing, it is answering wrongly; the difference is worth telling the user
// who installed it.
func DecodeTakes(data []byte) ([]take.Take, error) {
	r := &reader{data: data}
	count := r.size("take count")
	var takes []take.Take
	for index := int32(0); index < count; index++ {
		parts := r.size("part count")
		if r.err == nil && parts == 0 {
			r.fail("one of its takes has no parts")
		}
		if r.err != nil {
			return nil, r.err
		}
		one := make(take.Take, 0, parts)
		for part := int32(0); part < parts; part++ {
			one = append(one, r.part())
		}
		if r.err != nil {
			return nil, r.err
		}
		takes = append(takes, one)
	}
	if err := r.done("takes"); err != nil {
		return nil, err
	}
	return takes, nil
}
