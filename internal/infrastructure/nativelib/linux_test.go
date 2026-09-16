//go:build linux

package nativelib

// The Linux half, over the C library the process already has mapped, loaded by its full path so a
// refusal can be checked for naming it: a missing library or function refused without its name, a
// call filling a buffer this process owns and a string answered by address.

import (
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"github.com/oernster/bridge-talk/internal/infrastructure/nativelib/nativelibtest"
)

// confstrPath is _CS_PATH, which asks confstr for the default search path.
const confstrPath = 0

// noSuchFile is ENOENT, whose message strerror answers.
const noSuchFile = 2

func TestALibraryOrFunctionNotThereIsRefusedWithoutItsName(t *testing.T) {
	t.Parallel()

	absent := filepath.Join(t.TempDir(), "absent.so")
	if loaded, err := Open(absent); loaded != nil || err == nil || strings.Contains(err.Error(), absent) {
		t.Errorf("Open(absent) = %v, %v; want a refusal without the path", loaded, err)
	}
	path := nativelibtest.SystemLibrary(t)
	library, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%s): %v", path, err)
	}
	if address, err := library.Function("NoSuchFunction"); address != 0 || err == nil || strings.Contains(err.Error(), path) {
		t.Errorf("Function = %d, %v; want a refusal without the library's path", address, err)
	}
}

// confstr takes a caller-owned buffer and its length, answering the bytes the whole value needs: the
// shape a plugin's Describe has.
func TestACallFillsABufferThisProcessOwns(t *testing.T) {
	t.Parallel()

	function := libcFunction(t, "confstr")

	needed := Call(function, confstrPath, 0, 0)
	if needed == 0 {
		t.Fatal("asking for the size answered nothing")
	}
	buffer := make([]byte, needed)
	Call(function, confstrPath, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))

	if got := string(buffer[:needed-1]); !strings.Contains(got, "/bin") {
		t.Errorf("the buffer holds %q, which is no search path", got)
	}
}

func TestAStringAnsweredByAddressIsCopied(t *testing.T) {
	t.Parallel()

	if got := String(Call(libcFunction(t, "strerror"), noSuchFile)); got != "No such file or directory" {
		t.Errorf("strerror(ENOENT) = %q", got)
	}
}

// libcFunction answers the address of the C library's function name.
func libcFunction(t *testing.T, name string) uintptr {
	t.Helper()
	library, err := Open(nativelibtest.SystemLibrary(t))
	if err != nil {
		t.Fatalf("opening the C library: %v", err)
	}
	function, err := library.Function(name)
	if err != nil {
		t.Fatalf("finding %s: %v", name, err)
	}
	return function
}
