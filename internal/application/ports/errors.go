package ports

import "errors"

// ErrPlaybackFailed is returned by a player that could not start a sequence.
var ErrPlaybackFailed = errors.New("playback failed")
