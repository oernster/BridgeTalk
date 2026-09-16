//go:build windows

package speechmodel

import "golang.org/x/sys/windows"

// modelPathFor answers the model's path as ONNX Runtime reads one on Windows: UTF-16, null-terminated.
func modelPathFor(path string) (*uint16, error) { return windows.UTF16PtrFromString(path) }
