package plugin

import (
	"fmt"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/take"
)

// ABIVersion is the interface version this application implements. A plugin stating any
// other version is passed over by name (FR-564).
const ABIVersion = 1

// Library is a loaded plugin's three exported functions, as PLUGINS-GUIDE.md states them.
//
// It is an interface so that everything above it can be exercised without a library file,
// which cannot be built here: see the package comment on plugintest.
type Library interface {
	// Version answers the interface version the plugin was built against.
	Version() int32
	// Describe fills buffer with the plugin's account of itself, answering the bytes needed
	// when buffer is empty and a negative number when it refuses.
	Describe(buffer []byte) int32
	// Takes fills buffer with what one voice may play for one cue, on the same terms.
	Takes(voiceIndex int32, cueID []byte, buffer []byte) int32
}

// Plugin is one loaded plugin: what it calls itself, the file it came from and the voices
// it offers.
type Plugin struct {
	// Name is the plugin's own name, which it gave rather than its file carrying.
	Name string
	// File is the name of the file it was loaded from, used when it has to be named in a
	// refusal or beside a voice sharing another plugin's name (FR-568).
	File string

	voices  []*Voice
	library Library
	on      *runner
}

// Voices lists the voices this plugin offers, in the order it offered them.
func (p *Plugin) Voices() []*Voice { return p.voices }

// Voice is one voice a plugin offers. It is an audio source like any other (FR-501).
type Voice struct {
	// ID identifies the voice within its plugin and is kept in the settings beside the
	// plugin's name, so the same voice is found again next run (FR-569).
	ID string
	// Name is what the user sees.
	Name string
	// Ready says whether the audio this voice needs is present on this machine (FR-570).
	Ready bool
	// Reason says why it is not; empty while Ready.
	Reason string

	plugin *Plugin
	index  int32
}

// Plugin answers which plugin offered this voice.
func (v *Voice) Plugin() *Plugin { return v.plugin }

// Lookup returns the takes this voice holds for a cue; false when it holds none.
//
// A voice whose audio is not present answers nothing rather than asking the plugin: it
// could not be cast (FR-570), so a question reaching here is one nobody should have asked.
// A refusal or an answer that will not read is silence too, recorded by the caller that
// wired the plugin up; a cue no voice serves is silence, which is always preferred to a
// wrong line.
func (v *Voice) Lookup(id cue.ID) ([]take.Take, bool) {
	if !v.Ready {
		return nil, false
	}
	answer, err := v.plugin.ask(func(buffer []byte) int32 {
		return v.plugin.library.Takes(v.index, []byte(id), buffer)
	})
	if err != nil {
		return nil, false
	}
	takes, err := DecodeTakes(answer)
	if err != nil || len(takes) == 0 {
		return nil, false
	}
	return takes, true
}

// ErrRefused is returned for a call the plugin answered with a negative number, which is
// always a refusal and never a size (PLUGINS-GUIDE.md, calling rule 3).
var ErrRefused = fmt.Errorf("the plugin refused the call")

// ask runs the size-first protocol over one of a plugin's buffer-filling functions.
//
// The size is asked for, the buffer is made to fit and the answer is asked for again. A
// plugin whose second answer differs from its first is refused rather than read: the bytes
// in hand are then of an unknown length; reading a prefix of an answer is how a
// truncated list becomes a shorter one that looks complete.
//
// Every call goes through the plugin thread, which is what makes them one at a time.
func (p *Plugin) ask(call func(buffer []byte) int32) ([]byte, error) {
	var size, written int32
	p.on.do(func() { size = call(nil) })
	if size < 0 {
		return nil, ErrRefused
	}
	if size == 0 {
		return nil, nil
	}
	buffer := make([]byte, size)
	p.on.do(func() { written = call(buffer) })
	if written < 0 {
		return nil, ErrRefused
	}
	if written != size {
		return nil, fmt.Errorf("%w: it asked for %d bytes then wrote %d", ErrMalformed, size, written)
	}
	return buffer, nil
}
