// Package nativelibtest answers a library the operating system itself ships, for the tests of
// everything that loads a native library: a file certainly real, certainly loadable and certainly not
// ONNX Runtime or a plugin.
package nativelibtest

import "testing"

// SystemLibrary answers the full path of a library every machine of this platform has, failing the
// test where none can be found.
func SystemLibrary(t *testing.T) string {
	t.Helper()
	path, err := systemLibrary()
	if err != nil {
		t.Fatalf("finding a system library: %v", err)
	}
	return path
}
