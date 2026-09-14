package pyvenv

import (
	"path/filepath"
	"runtime"
	"testing"
)

// toolDir is a tool's directory as a tool names it, from the repository root.
const toolDir = "tools/sounds"

// On Windows the venv keeps its Python under Scripts as python.exe.
func TestTheWindowsVenvKeepsItsPythonUnderScripts(t *testing.T) {
	want := filepath.Join(toolDir, "venv", "Scripts", "python.exe")
	if got := interpreterOn("windows", toolDir); got != want {
		t.Errorf("interpreterOn(windows) = %q, want %q", got, want)
	}
}

// Everywhere else the venv keeps its Python under bin as python.
func TestEveryOtherVenvKeepsItsPythonUnderBin(t *testing.T) {
	want := filepath.Join(toolDir, "venv", "bin", "python")
	for _, goos := range []string{"linux", "darwin"} {
		if got := interpreterOn(goos, toolDir); got != want {
			t.Errorf("interpreterOn(%s) = %q, want %q", goos, got, want)
		}
	}
}

// Interpreter answers the layout of the operating system it runs on.
func TestInterpreterAnswersTheLayoutOfThisOperatingSystem(t *testing.T) {
	if got, want := Interpreter(toolDir), interpreterOn(runtime.GOOS, toolDir); got != want {
		t.Errorf("Interpreter = %q, want %q", got, want)
	}
}
