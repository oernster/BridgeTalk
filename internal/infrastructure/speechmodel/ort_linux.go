//go:build linux

package speechmodel

import "syscall"

// modelPathFor answers the model's path as ONNX Runtime reads one off Windows: bytes, null-terminated.
func modelPathFor(path string) (*byte, error) { return syscall.BytePtrFromString(path) }
