package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/oernster/bridge-talk/tools/internal/pyvenv"
)

const (
	// toolDir holds pauses.py and the venv it runs in, from the repository root.
	toolDir = "tools/pauses"
	// finderScript is the Python half of the tool.
	finderScript = "pauses.py"
)

// python finds breaks by running pauses.py in the tool's own venv under the repository at root: a JSON
// request in, a JSON list of answers out.
type python struct {
	root string
}

// Find runs pauses.py once over every file of one voice.
func (p python) Find(asked request) ([]found, error) {
	body, err := json.Marshal(asked)
	if err != nil {
		return nil, err
	}
	interpreter := pyvenv.Interpreter(filepath.Join(p.root, toolDir))
	command := exec.Command(interpreter, filepath.Join(p.root, toolDir, finderScript))
	command.Stdin = bytes.NewReader(body)
	command.Stderr = os.Stderr
	answer, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("running %s in %s: %w", finderScript, interpreter, err)
	}
	var answers []found
	if err := json.Unmarshal(answer, &answers); err != nil {
		return nil, fmt.Errorf("reading what %s answered: %w", finderScript, err)
	}
	return answers, nil
}
