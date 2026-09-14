//go:build !windows

package speechmodel

import "errors"

// ErrUnsupported says ONNX Runtime is loaded on Windows alone for now. Linux is in scope once all
// other work is done (REQUIREMENTS.md section 2.3); until then every line made elsewhere fails
// with this reason.
var ErrUnsupported = errors.New("the speech model runs on Windows only")

// openSession loads nothing off Windows.
func openSession(string) (session, error) { return nil, ErrUnsupported }
