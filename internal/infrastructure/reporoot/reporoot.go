// Package reporoot finds the repository root: the nearest folder at or above a starting folder that
// holds go.mod. The tools that work on the tree and the structural tests share this one walk.
package reporoot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// moduleFile marks the root.
const moduleFile = "go.mod"

// ErrNoModule means the walk reached the top of the drive without meeting go.mod.
var ErrNoModule = errors.New("no folder above holds go.mod")

// Find answers with the nearest folder at or above from that holds go.mod.
func Find(from string) (string, error) { return find(from, isFile) }

// isFile reports whether path names a file rather than a folder or nothing.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// find walks up from from, asking exists about go.mod in each folder until the top of the drive.
func find(from string, exists func(string) bool) (string, error) {
	dir := filepath.Clean(from)
	for {
		if exists(filepath.Join(dir, moduleFile)) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("walking up from %s: %w", from, ErrNoModule)
		}
		dir = parent
	}
}
