//go:build windows

package nativelib

// The Windows half, over a library Windows itself ships, since no library of this project's own can be
// built here: a missing library or function refused without its name, a call filling a buffer this
// process owns and a string answered by address.

import (
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"github.com/oernster/bridge-talk/internal/infrastructure/nativelib/nativelibtest"
)

func TestALibraryOrFunctionNotThereIsRefusedWithoutItsName(t *testing.T) {
	t.Parallel()

	absent := filepath.Join(t.TempDir(), "absent.dll")
	if loaded, err := Open(absent); loaded != nil || err == nil || strings.Contains(err.Error(), absent) {
		t.Errorf("Open(absent) = %v, %v; want a refusal without the path", loaded, err)
	}
	const missing = "NoSuchFunction"
	if address, err := kernel32(t).Function(missing); address != 0 || err == nil || strings.Contains(err.Error(), missing) {
		t.Errorf("Function(%s) = %d, %v; want a refusal without the name", missing, address, err)
	}
}

// A name no file or function can have is refused before Windows is asked, with the reason Go gives.
func TestANameNoLibraryOrFunctionCanHaveIsRefused(t *testing.T) {
	t.Parallel()

	const impossible = "absent\x00"
	if loaded, err := Open(impossible); loaded != nil || err == nil {
		t.Errorf("Open(%q) = %v, %v; want a refusal", impossible, loaded, err)
	}
	if address, err := kernel32(t).Function(impossible); address != 0 || err == nil {
		t.Errorf("Function(%q) = %d, %v; want a refusal", impossible, address, err)
	}
}

// GetSystemDirectoryA takes a caller-owned buffer and its length, answering the bytes needed when the
// buffer is too small: the shape a plugin's Describe has.
func TestACallFillsABufferThisProcessOwns(t *testing.T) {
	t.Parallel()

	function := kernel32Function(t, "GetSystemDirectoryA")

	needed := Call(function, 0, 0)
	if needed == 0 {
		t.Fatal("asking for the size answered nothing")
	}
	buffer := make([]byte, needed)
	written := Call(function, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))

	if written == 0 || written >= needed {
		t.Fatalf("wrote %d bytes into a buffer of %d, want fewer than it asked for", written, needed)
	}
	if got := string(buffer[:written]); !strings.Contains(strings.ToLower(got), "system32") {
		t.Errorf("the buffer holds %q, which is no system directory", got)
	}
}

func TestAStringAnsweredByAddressIsCopied(t *testing.T) {
	t.Parallel()

	if got := String(Call(kernel32Function(t, "GetCommandLineA"))); !strings.Contains(got, "nativelib") {
		t.Errorf("the command line read %q, want this test binary's", got)
	}
}

// kernel32 answers the system library loaded.
func kernel32(t *testing.T) *Library {
	t.Helper()
	library, err := Open(nativelibtest.SystemLibrary(t))
	if err != nil {
		t.Fatalf("opening the system library: %v", err)
	}
	return library
}

// kernel32Function answers the address of the system library's function name.
func kernel32Function(t *testing.T, name string) uintptr {
	t.Helper()
	function, err := kernel32(t).Function(name)
	if err != nil {
		t.Fatalf("finding %s: %v", name, err)
	}
	return function
}
