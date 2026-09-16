// Package take holds what a voice answers a cue with. It is pure: no file is opened here
// and no path is interpreted, so the same take means the same thing to a recorded voice,
// a machine voice and a plugin.
package take

// Take is one answer to a cue: the parts it is made of, in the order they are heard.
//
// Most takes have one part. A take of several parts is a line recorded in pieces, played
// one after another with no added gap (FR-573). The parts are alternatives to nothing:
// they are one utterance, whereas two takes of the same cue are alternatives to each other
// and only one of them is ever spoken.
type Take []string

// Key identifies a take among the takes of one cue, so the picker can avoid the one it
// chose last (FR-610).
//
// The first part is the identity. Two takes of a cue never begin with the same file, since
// a voice answers each part with a path and a path answers one part. An empty take has no
// identity and answers with the empty string, which no path equals.
func (t Take) Key() string {
	if len(t) == 0 {
		return ""
	}
	return t[0]
}

// Empty reports whether a take has no parts, which is not a take at all. A source that
// answers one is answering nothing, so it is passed over rather than played.
func (t Take) Empty() bool { return len(t) == 0 }

// Of builds a take from its parts, which reads better than a conversion at every call site
// that has a single file to offer.
func Of(parts ...string) Take { return Take(parts) }
