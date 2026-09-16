//go:build windows

package main

// applicationIcon is empty on Windows, where the tray reads the icon out of the binary.
var applicationIcon []byte

// nativeVoicesMissingOn is empty on Windows, where machine voices and plugins load their native
// libraries (FR-817, FR-818).
const nativeVoicesMissingOn = ""
