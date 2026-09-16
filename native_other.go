//go:build !windows

package main

import _ "embed"

// applicationIcon is the committed icon file, handed to a tray that is given a picture rather than
// reading the icon out of the binary as Windows does (FR-814).
//
//go:embed assets/application-icon.ico
var applicationIcon []byte

// nativeVoicesMissingOn names the platform where machine voices and plugins are not offered yet:
// both load a native library, which is loaded on Windows alone so far (FR-817, FR-818). Linux is
// the one other platform in scope (section 2.3), so it is the name a reader is given.
const nativeVoicesMissingOn = "Linux"
