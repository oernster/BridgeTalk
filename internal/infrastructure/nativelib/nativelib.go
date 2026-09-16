// Package nativelib loads a native library by its full path, finds its functions by name and calls
// them, all with cgo disabled: a DLL through golang.org/x/sys/windows on Windows, a shared object
// through purego on Linux. It is the one home for crossing into native code, so ONNX Runtime and a
// plugin are called the same way on both.
//
// An address handed across is converted to uintptr in the argument list of Call itself and nowhere
// earlier. Call is marked //go:uintptrescapes, so Go moves whatever such an address points at to the
// heap, where no stack move can leave the library reading an old copy, then keeps it alive until Call
// returns (TestAddressesAreConvertedOnlyWhereTheCallIsMade in tests/structural).
//
// A library once loaded stays loaded until the process ends: nothing here unloads one, since whether a
// library can be unloaded safely while its own threads may still run has not been measured.
package nativelib

import "unsafe"

// Pointer treats an address a library answered with as the pointer it is. The memory is the
// library's own, never Go's, so the collector has nothing there to move or free.
func Pointer(address uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&address))
}

// String copies the null-terminated bytes a library answered the address of; empty for no address.
func String(address uintptr) string {
	if address == 0 {
		return ""
	}
	start := (*byte)(Pointer(address))
	length := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(start), length)) != 0 {
		length++
	}
	return string(unsafe.Slice(start, length))
}
