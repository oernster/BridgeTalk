//go:build linux

package nativelib

import (
	"errors"
	"strings"

	"github.com/ebitengine/purego"
)

// Library is one loaded shared object.
type Library struct {
	handle uintptr
	path   string
}

// Open loads the shared object at path, resolving every symbol now so a missing dependency is found
// here rather than at the first call. The reason for a refusal is what the loader said, without the
// path, which the caller names.
func Open(path string) (*Library, error) {
	handle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return nil, withoutName(err, path)
	}
	return &Library{handle: handle, path: path}, nil
}

// Function answers the address of the function the library exports as name.
func (l *Library) Function(name string) (uintptr, error) {
	address, err := purego.Dlsym(l.handle, name)
	if err != nil {
		return 0, withoutName(err, l.path)
	}
	return address, nil
}

// Call calls the function at address with arguments, answering its first result.
//
//go:uintptrescapes
func Call(address uintptr, arguments ...uintptr) uintptr {
	answer, _, _ := purego.SyscallN(address, arguments...)
	return answer
}

// withoutName keeps what the loader said, without the library's path, which the caller gives.
func withoutName(err error, path string) error {
	return errors.New(strings.ReplaceAll(err.Error(), path+": ", ""))
}
