// Package plugintest is a plugin that answers in the layouts a real one answers in,
// without being a library file.
//
// It exists because a real plugin cannot be built here. A library file exporting C
// functions needs cgo and a C toolchain, which this repository deliberately does without:
// ONNX Runtime is called through its C API for that very reason. So the contract is stood
// up in Go instead, obeying every rule PLUGINS-GUIDE.md states: the caller owns the buffer,
// a size is asked for before an answer is filled, a negative return is a refusal and never
// a size, then every string carries its own length.
//
// What that proves and what it does not: it proves the layouts, the protocol and everything
// read back from them. It does not prove the call into a library file, which is the Windows
// half's own business and is measured there.
package plugintest

import "encoding/binary"

// ABIVersion is the interface version this package writes, which is the one the application
// implements. A test wanting a mismatch sets its own on the Plugin.
const ABIVersion = 1

// Refused is what every call answers when it cannot, which a reader must never mistake for
// a size (PLUGINS-GUIDE.md, calling rule 3).
const Refused = -1

// intSize is the width of every count and length, as on the real wire.
const intSize = 4

// Writer builds an answer byte by byte in the layout a plugin writes.
//
// It is the other half of the reader in wire.go, written out rather than shared, so a fault
// in one cannot hide itself by being present in both.
type Writer struct{ data []byte }

// Count appends one number: a length, a flag or a total.
func (w *Writer) Count(value int32) {
	var room [intSize]byte
	binary.LittleEndian.PutUint32(room[:], uint32(value))
	w.data = append(w.data, room[:]...)
}

// Text appends one length-prefixed string.
func (w *Writer) Text(value string) {
	w.Count(int32(len(value)))
	w.data = append(w.data, value...)
}

// Bytes answers what has been written.
func (w *Writer) Bytes() []byte { return w.data }

// Voice is one voice a test plugin offers.
type Voice struct {
	// ID identifies the voice within its plugin.
	ID string
	// Name is what the user would see.
	Name string
	// Ready says whether the audio it needs is present.
	Ready bool
	// Reason says why it is not; empty while Ready.
	Reason string
	// Answers maps a cue id to the takes this voice offers for it, each take its parts in
	// the order they are heard.
	Answers map[string][][]string
}

// Plugin is a whole plugin: what it says it is, what it offers and what it refuses.
type Plugin struct {
	// ABI is the interface version it claims. Zero means ABIVersion, so a test that does
	// not care about the handshake says nothing about it.
	ABI int32
	// Name is the plugin's own name.
	Name string
	// Voices are the voices it offers, in the order it offers them.
	Voices []Voice

	// RefuseVersion, RefuseDescribe and RefuseTakes make the matching call answer Refused,
	// which is how a test reaches the paths a misbehaving plugin would.
	RefuseVersion  bool
	RefuseDescribe bool
	RefuseTakes    bool
	// Corrupt replaces a described answer with bytes that are no layout at all.
	Corrupt bool
}

// Version answers the interface version this plugin was built against.
func (p *Plugin) Version() int32 {
	if p.RefuseVersion {
		return Refused
	}
	if p.ABI == 0 {
		return ABIVersion
	}
	return p.ABI
}

// Describe writes this plugin's account of itself, obeying the size-first protocol.
func (p *Plugin) Describe(buffer []byte) int32 {
	if p.RefuseDescribe {
		return Refused
	}
	return answer(p.description(), buffer)
}

// Takes writes what one voice may play for one cue, obeying the same protocol.
//
// A voice index outside the ones offered is a refusal rather than an empty answer: the
// application asked about a voice this plugin never mentioned, which is a fault in the
// asking rather than a cue the voice has nothing for.
func (p *Plugin) Takes(voiceIndex int32, cueID []byte, buffer []byte) int32 {
	if p.RefuseTakes || voiceIndex < 0 || int(voiceIndex) >= len(p.Voices) {
		return Refused
	}
	return answer(takesOf(p.Voices[voiceIndex].Answers[string(cueID)]), buffer)
}

// description writes the description layout.
func (p *Plugin) description() []byte {
	if p.Corrupt {
		return corrupted()
	}
	w := &Writer{}
	w.Count(int32(len(p.Voices)))
	w.Text(p.Name)
	for _, voice := range p.Voices {
		w.Text(voice.ID)
		w.Text(voice.Name)
		w.Count(flag(voice.Ready))
		w.Text(voice.Reason)
	}
	return w.Bytes()
}

// Description writes a description without a whole plugin around it, for a test that reads
// one directly.
func Description(name string, voices ...Voice) []byte {
	return (&Plugin{Name: name, Voices: voices}).description()
}

// Takes writes a takes answer without a plugin around it, each take given as its parts.
func Takes(takes ...[]string) []byte { return takesOf(takes) }

// takesOf writes the takes layout.
func takesOf(takes [][]string) []byte {
	w := &Writer{}
	w.Count(int32(len(takes)))
	for _, one := range takes {
		w.Count(int32(len(one)))
		for _, part := range one {
			w.Text(part)
		}
	}
	return w.Bytes()
}

// corrupted is an answer that is no layout: a count promising more voices than there are
// bytes to hold them.
func corrupted() []byte {
	w := &Writer{}
	w.Count(1)
	return w.Bytes()
}

// flag writes a boolean the way the wire carries one.
func flag(set bool) int32 {
	if set {
		return 1
	}
	return 0
}

// answer implements the size-first protocol over a written answer: an empty buffer asks how
// many bytes are needed, a buffer large enough is filled and one too small is refused.
//
// A buffer too small is a refusal rather than a partial write, because the caller asked for
// the size first and was told; arriving with less than that is the caller's fault and a
// partial answer would read as a whole one.
func answer(written []byte, buffer []byte) int32 {
	if len(buffer) == 0 {
		return int32(len(written))
	}
	if len(buffer) < len(written) {
		return Refused
	}
	copy(buffer, written)
	return int32(len(written))
}
