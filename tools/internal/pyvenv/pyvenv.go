// Package pyvenv finds the Python a tool runs its script in: the one in the venv inside the tool's
// own directory. It is internal to tools, so only the tools can import it.
package pyvenv

import (
	"path/filepath"
	"runtime"
)

const (
	// venvDir is the folder in a tool's directory that holds its venv.
	venvDir = "venv"
	// windows is the operating system whose venvs keep their Python under Scripts.
	windows = "windows"
)

// Interpreter returns the Python of the venv in the tool directory given, which lives in a different
// place on Windows.
func Interpreter(toolDir string) string {
	return interpreterOn(runtime.GOOS, toolDir)
}

// interpreterOn returns the Python of the venv in toolDir as a venv lays it out on the operating
// system goos names.
func interpreterOn(goos, toolDir string) string {
	if goos == windows {
		return filepath.Join(toolDir, venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(toolDir, venvDir, "bin", "python")
}
