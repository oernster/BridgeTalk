//go:build windows

package nativelib

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

// Library is one loaded DLL.
type Library struct {
	dll *windows.DLL
}

// Open loads the DLL at path. Loading by the full path means an older copy Windows keeps in System32
// is never picked up in its place. The reason for a refusal is what Windows said, without the path,
// which the caller names.
func Open(path string) (*Library, error) {
	dll, err := windows.LoadDLL(path)
	if err != nil {
		return nil, withoutName(err)
	}
	return &Library{dll: dll}, nil
}

// Function answers the address of the function the library exports as name.
func (l *Library) Function(name string) (uintptr, error) {
	found, err := l.dll.FindProc(name)
	if err != nil {
		return 0, withoutName(err)
	}
	return found.Addr(), nil
}

// Call calls the function at address with arguments, answering its first result.
//
//go:uintptrescapes
func Call(address uintptr, arguments ...uintptr) uintptr {
	answer, _, _ := syscall.SyscallN(address, arguments...)
	return answer
}

// withoutName keeps what Windows said about a library or function it could not find, without the
// name, which the caller gives.
func withoutName(err error) error {
	var failed *windows.DLLError
	if errors.As(err, &failed) {
		return failed.Err
	}
	return err
}
