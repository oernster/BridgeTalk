package plugin

// The Windows half: loading a library file, finding its functions and calling one with a
// buffer the caller owns.
//
// No plugin exists to load, nor can one be built here, so these use libraries Windows
// itself ships. That proves what the plugin path actually depends on: that a library loads
// by full path, that a missing export is found to be missing rather than called, then that a
// call through syscall.SyscallN fills a Go buffer and answers a size.

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// systemLibrary is a library present on every Windows machine, used as a library that is
// certainly real and certainly not a plugin.
const systemLibrary = "kernel32.dll"

func TestARealLibraryThatIsNoPluginIsRefusedByTheFunctionItLacks(t *testing.T) {
	t.Parallel()

	library, err := OpenLibrary(systemLibrary)

	if library != nil {
		t.Fatalf("kernel32 loaded as a plugin")
	}
	if err == nil || !strings.Contains(err.Error(), versionFunction) {
		t.Errorf("refusal = %v, want the missing function named", err)
	}
}

func TestSomethingThatIsNoLibraryAtAllIsRefused(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "not-a-library.dll")
	if err := os.WriteFile(path, []byte("this is text"), 0o600); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	library, err := OpenLibrary(path)

	if library != nil || err == nil {
		t.Fatalf("library = %v, err = %v; want a refusal", library, err)
	}
	if !strings.Contains(err.Error(), "could not be loaded") {
		t.Errorf("refusal = %v, want it to say the file would not load", err)
	}
}

func TestAFileThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := OpenLibrary(filepath.Join(t.TempDir(), "absent.dll")); err == nil {
		t.Error("a library that is not there loaded")
	}
}

// The plumbing a plugin call rests on, measured rather than assumed: a library loaded by
// name, a function found in it and called through syscall.SyscallN with a buffer this
// process owns, answering a size the way the plugin protocol does.
//
// GetSystemDirectoryA is used because it takes a caller-owned buffer and a length, then
// answers the bytes needed when the buffer is too small, which is the same shape as
// BridgeTalkPluginDescribe. Every address is converted inside the call's own argument list,
// as TestAddressesAreConvertedOnlyWhereTheCallIsMade requires.
func TestACallThroughSyscallFillsABufferThisProcessOwns(t *testing.T) {
	t.Parallel()

	loaded, err := windows.LoadDLL(systemLibrary)
	if err != nil {
		t.Fatalf("loading %s: %v", systemLibrary, err)
	}
	found, err := loaded.FindProc("GetSystemDirectoryA")
	if err != nil {
		t.Fatalf("finding GetSystemDirectoryA: %v", err)
	}

	needed, _, _ := syscall.SyscallN(found.Addr(), 0, 0)
	if needed == 0 {
		t.Fatal("asking for the size answered nothing")
	}

	buffer := make([]byte, needed)
	written, _, _ := syscall.SyscallN(found.Addr(),
		uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))

	if written == 0 || written >= needed {
		t.Fatalf("wrote %d bytes into a buffer of %d, want fewer than it asked for", written, needed)
	}
	if got := string(buffer[:written]); !strings.Contains(strings.ToLower(got), "system32") {
		t.Errorf("the buffer holds %q, which is no system directory", got)
	}
}

// Every call is made from one thread that belongs to nothing else, which is what lets a
// plugin keep state belonging to a thread, a COM apartment being the usual one.
func TestEveryCallArrivesOnOneThreadOfItsOwn(t *testing.T) {
	t.Parallel()

	on := newRunner()
	defer on.close()

	var seen []uint32
	for range 8 {
		on.do(func() { seen = append(seen, windows.GetCurrentThreadId()) })
	}

	if len(seen) != 8 {
		t.Fatalf("saw %d thread ids, want 8", len(seen))
	}
	for _, id := range seen {
		if id != seen[0] {
			t.Fatalf("calls arrived on %v, want one thread throughout", seen)
		}
	}
	if seen[0] == windows.GetCurrentThreadId() {
		t.Error("calls arrived on the calling thread, so the thread is not the plugin's own")
	}
}
