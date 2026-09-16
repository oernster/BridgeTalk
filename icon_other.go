//go:build !windows

package main

import _ "embed"

// applicationIcon is the committed icon file, handed to a tray that is given a picture rather than
// reading the icon out of the binary as Windows does (FR-814).
//
//go:embed assets/application-icon.ico
var applicationIcon []byte
