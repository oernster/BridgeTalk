//go:build !windows && !linux

package speechmodel

import "errors"

// ErrUnsupported says ONNX Runtime is loaded on Windows and Linux alone; every line made elsewhere
// fails with this reason.
var ErrUnsupported = errors.New("the speech model runs on Windows and Linux only")

// openSession loads nothing here.
func openSession(string) (session, error) { return nil, ErrUnsupported }
