//go:build windows || linux

package plugin

// Calling a plugin's three functions, which is the only part of this package that loads native code
// and the only part a test here cannot reach without a library file.
//
// The technique is the one speechmodel uses for ONNX Runtime, through nativelib: the library is loaded
// by its full path, its functions are found by name and each is called with cgo disabled. Every
// address is converted to uintptr inside the argument list of nativelib.Call itself, never earlier,
// which is the rule TestAddressesAreConvertedOnlyWhereTheCallIsMade holds over the whole tree: only
// there does Go keep the buffer where the library was told it is.

import (
	"fmt"
	"unsafe"

	"github.com/oernster/bridge-talk/internal/infrastructure/nativelib"
	"github.com/oernster/bridge-talk/internal/product"
)

// The exported names a plugin has to carry, built from the one place the product is
// named. The slug is the identity for somewhere only a name without spaces will do, which
// is exactly what an exported C name is.
const (
	versionFunction  = product.Slug + "PluginABIVersion"
	describeFunction = product.Slug + "PluginDescribe"
	takesFunction    = product.Slug + "PluginTakes"
)

// nativeLibrary is one loaded plugin file.
type nativeLibrary struct {
	version  uintptr
	describe uintptr
	takes    uintptr
}

// OpenLibrary loads a plugin file and finds the three functions it must export.
//
// A file that is no plugin is refused here rather than later: a library that loads but
// exports none of these is something else the user dropped into the folder; saying so
// by name is more use than a voice quietly never appearing.
func OpenLibrary(path string) (Library, error) {
	loaded, err := nativelib.Open(path)
	if err != nil {
		return nil, fmt.Errorf("it could not be loaded: %w", err)
	}
	library := &nativeLibrary{}
	for _, each := range []struct {
		name string
		at   *uintptr
	}{
		{versionFunction, &library.version},
		{describeFunction, &library.describe},
		{takesFunction, &library.takes},
	} {
		found, err := loaded.Function(each.name)
		if err != nil {
			return nil, fmt.Errorf("it exports no %s, so it is no plugin", each.name)
		}
		*each.at = found
	}
	return library, nil
}

// Version asks which interface version the plugin was built against. It takes nothing, so
// no address crosses.
func (l *nativeLibrary) Version() int32 {
	return int32(nativelib.Call(l.version))
}

// Describe asks the plugin to fill buffer; for the size alone when buffer is empty.
//
// The empty case is written out rather than folded into one call, because the address of
// the first byte of an empty slice cannot be taken and a helper answering zero for it would
// be the very helper the conversion rule forbids.
func (l *nativeLibrary) Describe(buffer []byte) int32 {
	if len(buffer) == 0 {
		return int32(nativelib.Call(l.describe, 0, 0))
	}
	return int32(nativelib.Call(l.describe,
		uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer))))
}

// Takes asks the plugin what one voice may play for one cue, on the same terms.
//
// A cue id is never empty, so an empty one is refused here rather than being sent as a null
// pointer a plugin would have to guard against.
func (l *nativeLibrary) Takes(voiceIndex int32, cueID []byte, buffer []byte) int32 {
	if len(cueID) == 0 {
		return -1
	}
	if len(buffer) == 0 {
		return int32(nativelib.Call(l.takes, uintptr(voiceIndex),
			uintptr(unsafe.Pointer(&cueID[0])), uintptr(len(cueID)), 0, 0))
	}
	return int32(nativelib.Call(l.takes, uintptr(voiceIndex),
		uintptr(unsafe.Pointer(&cueID[0])), uintptr(len(cueID)),
		uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer))))
}
