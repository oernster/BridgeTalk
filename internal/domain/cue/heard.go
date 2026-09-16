package cue

// Heard reports whether a moment can be heard: switched on in Chatter. A moment Chatter does not list,
// such as the cue from the application, is heard (FR-745).
type Heard func(ID) bool

// HeardAll hears every moment, for a caller with no switches to ask.
func HeardAll(ID) bool { return true }
