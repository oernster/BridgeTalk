//go:build linux

package nativelibtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// libcName is the file name of the C library every Linux process loading a library has mapped.
const libcName = "libc.so.6"

// mapsFile lists what the running process has mapped, one mapping a line with the file last.
const mapsFile = "/proc/self/maps"

// systemLibrary answers the C library by the full path the process mapped it from, which is where it
// sits on this distribution rather than where one distribution puts it.
func systemLibrary() (string, error) {
	maps, err := os.ReadFile(mapsFile)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(maps), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && filepath.Base(fields[len(fields)-1]) == libcName {
			return fields[len(fields)-1], nil
		}
	}
	return "", fmt.Errorf("%s maps no %s", mapsFile, libcName)
}
