package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
)

// toolDir holds sounds.py and the venv it runs in, from the repository root.
const toolDir = "tools/sounds"

// britishFlag asks sounds.py for British English; without it the sounds are American.
const britishFlag = "--british"

// python makes speech sounds by running sounds.py in the tool's own venv: a JSON list of lines
// in, a JSON list of their speech sounds out.
type python struct{}

// Make runs sounds.py once for one accent over every line given.
func (python) Make(accent machinevoice.Accent, lines []string) ([]string, error) {
	request, err := json.Marshal(lines)
	if err != nil {
		return nil, err
	}
	args := []string{filepath.Join(toolDir, "sounds.py")}
	if accent == machinevoice.British {
		args = append(args, britishFlag)
	}
	command := exec.Command(interpreter(), args...)
	command.Stdin = bytes.NewReader(request)
	command.Stderr = os.Stderr
	answer, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("running sounds.py in %s: %w", interpreter(), err)
	}
	var sounds []string
	if err := json.Unmarshal(answer, &sounds); err != nil {
		return nil, fmt.Errorf("reading what sounds.py answered: %w", err)
	}
	return sounds, nil
}

// interpreter is the venv's own Python, which lives in a different place on Windows.
func interpreter() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(toolDir, "venv", "Scripts", "python.exe")
	}
	return filepath.Join(toolDir, "venv", "bin", "python")
}
