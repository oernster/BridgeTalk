// Package take holds what a voice answers a cue with. It is pure: no file is opened here
// and no path is interpreted, so the same take means the same thing to a recorded voice,
// a machine voice and a plugin.
package take

import "strconv"

// Part is one piece of a take: a whole file or a span of bytes inside one (FR-588).
//
// A span is how audio held many recordings to one file is played where it stands, without
// each recording first being copied out to a file of its own. Nothing here checks that a span
// lies inside its file or that its format is one the player decodes: that needs the disk and
// the decoders, so the player asks when the part is about to be heard and passes over a part
// that fails (FR-590).
type Part struct {
	// Path is the file the part is read from.
	Path string
	// Span is the stretch of that file the part is; nil for the whole file.
	Span *Span
}

// Span is a stretch of a file holding one recording.
type Span struct {
	// Format names how the bytes are decoded, since the file's own extension says nothing
	// about what is inside it (FR-589).
	Format string
	// Offset is the position of the span's first byte.
	Offset int64
	// Length is the number of bytes in the span.
	Length int64
}

// File is a part that is a whole file.
func File(path string) Part { return Part{Path: path} }

// SpanOf is a part that is a span of a file.
func SpanOf(path, format string, offset, length int64) Part {
	return Part{Path: path, Span: &Span{Format: format, Offset: offset, Length: length}}
}

// spanMark separates a span's path from its position in a key. No path on Windows or Linux
// can hold a zero byte, so a whole file's key never reads as a span's. A key is an identity
// and is never shown.
const spanMark = "\x00"

// Key identifies a part: its path for a whole file, its path with its offset and length for a
// span, so two spans of one file are two parts (FR-591).
func (p Part) Key() string {
	if p.Span == nil {
		return p.Path
	}
	return p.Path + spanMark + strconv.FormatInt(p.Span.Offset, 10) + spanMark +
		strconv.FormatInt(p.Span.Length, 10)
}

// Take is one answer to a cue: the parts it is made of, in the order they are heard.
//
// Most takes have one part. A take of several parts is a line recorded in pieces, played
// one after another with no added gap (FR-573). The parts are alternatives to nothing:
// they are one utterance, whereas two takes of the same cue are alternatives to each other
// and only one of them is ever spoken.
type Take []Part

// Key identifies a take among the takes of one cue, so the picker can avoid the one it
// chose last (FR-610).
//
// The first part is the identity. Two takes of a cue never begin with the same part, since
// a voice answers each part once. An empty take has no identity and answers with the empty
// string, which no part's key equals.
func (t Take) Key() string {
	if len(t) == 0 {
		return ""
	}
	return t[0].Key()
}

// Source is the file the take begins in, which is what a reader is shown of it; empty for an
// empty take. It is not the key: a key tells two spans of one file apart and is never shown.
func (t Take) Source() string {
	if len(t) == 0 {
		return ""
	}
	return t[0].Path
}

// Empty reports whether a take has no parts, which is not a take at all. A source that
// answers one is answering nothing, so it is passed over rather than played.
func (t Take) Empty() bool { return len(t) == 0 }

// Of builds a take of whole files, which reads better than a conversion at every call site
// that has files to offer.
func Of(paths ...string) Take {
	parts := make(Take, 0, len(paths))
	for _, path := range paths {
		parts = append(parts, File(path))
	}
	return parts
}
