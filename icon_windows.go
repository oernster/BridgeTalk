//go:build windows

package main

// applicationIcon is empty on Windows, where the tray reads the icon out of the binary.
var applicationIcon []byte
