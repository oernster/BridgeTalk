//go:build !windows && !linux

package nativelib

import "errors"

// ErrUnsupported says native libraries are loaded on Windows and Linux alone.
var ErrUnsupported = errors.New("native libraries are loaded on Windows and Linux only")

// Library is never made here.
type Library struct{}

// Open loads nothing here.
func Open(string) (*Library, error) { return nil, ErrUnsupported }

// Function finds nothing here.
func (*Library) Function(string) (uintptr, error) { return 0, ErrUnsupported }

// Call calls nothing here.
//
//go:uintptrescapes
func Call(uintptr, ...uintptr) uintptr { return 0 }
