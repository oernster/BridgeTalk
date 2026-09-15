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
	// toolDir holds the finders and the venv they run in, from the repository root.
	toolDir = "tools/pauses"
	// finderScript is the Python half of the tool that finds breaks.
	finderScript = "pauses.py"
	// endingsScript is the Python half of the tool that finds bursts (FR-555).
	endingsScript = "endings.py"
)

// python finds breaks and bursts by running the finders in the tool's own venv under the repository
// at root: a JSON request in, a JSON list of answers out.
type python struct {
	root string
}

// Find runs pauses.py once over every file of one voice.
func (p python) Find(asked request) ([]found, error) {
	var answers []found
	err := p.run(finderScript, asked, &answers)
	return answers, err
}

// FindEndings runs endings.py once over every file of one voice (FR-555).
func (p python) FindEndings(asked endingRequest) ([]ended, error) {
	var answers []ended
	err := p.run(endingsScript, asked, &answers)
	return answers, err
}

// run runs one finder over a request, reading its answers into answers.
func (p python) run(script string, asked, answers any) error {
	body, err := json.Marshal(asked)
	if err != nil {
		return err
	}
	interpreter := pyvenv.Interpreter(filepath.Join(p.root, toolDir))
	command := exec.Command(interpreter, filepath.Join(p.root, toolDir, script))
	command.Stdin = bytes.NewReader(body)
	command.Stderr = os.Stderr
	answer, err := command.Output()
	if err != nil {
		return fmt.Errorf("running %s in %s: %w", script, interpreter, err)
	}
	if err := json.Unmarshal(answer, answers); err != nil {
		return fmt.Errorf("reading what %s answered: %w", script, err)
	}
	return nil
}
