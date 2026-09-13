package library

// What a voice folder still has no take for, plus the way to the folder a take belongs
// in.
//
// Recording happens in a program built for it. The part only this application can do is
// know which moments a voice is missing and exactly which folder each take goes in.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// ErrUnknownCue means a moment was named that the cue vocabulary does not hold.
var ErrUnknownCue = errors.New("no moment by that id")

// VoiceDirs lists every folder directly inside root by name: every voice folder, whether
// or not it holds a take yet (FR-312). Files beside them are not voices. os.ReadDir
// answers sorted by name, which is the order the page shows them in.
func VoiceDirs(root string) ([]string, error) {
	if root == "" {
		return nil, ErrNoRoot
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", root, err)
	}
	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// Missing answers with the cues the named voice folder has no take for, in table order,
// plus how many cues it has at least one take for (FR-311, FR-313).
func Missing(root, name string, table cue.Table) ([]cue.Cue, int, error) {
	dir, err := voiceDir(root, name)
	if err != nil {
		return nil, 0, err
	}
	voice, _ := ScanVoice(dir, table)
	missing := []cue.Cue{}
	recorded := 0
	for _, item := range table.All() {
		if _, ok := voice.Lookup(item.ID()); ok {
			recorded++
			continue
		}
		missing = append(missing, item)
	}
	return missing, recorded, nil
}

// MomentFolder answers with the folder a take for one cue belongs in inside the named
// voice folder, making it where missing (FR-314). It adds and never replaces, as making
// a voice's folders does.
func MomentFolder(root, name, id string, table cue.Table) (string, error) {
	dir, err := voiceDir(root, name)
	if err != nil {
		return "", err
	}
	for _, item := range table.All() {
		if string(item.ID()) != id {
			continue
		}
		target := filepath.Join(dir, id)
		if err := os.MkdirAll(target, folderPerm); err != nil {
			return "", fmt.Errorf("making %s: %w", target, err)
		}
		return target, nil
	}
	return "", fmt.Errorf("%q: %w", id, ErrUnknownCue)
}

// voiceDir checks a voice folder name and answers with its directory, which has to exist
// already: neither the list nor a moment's folder makes a new voice.
func voiceDir(root, name string) (string, error) {
	if err := CheckVoiceName(name); err != nil {
		return "", err
	}
	if root == "" {
		return "", ErrNoRoot
	}
	dir := filepath.Join(root, name)
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", dir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", dir)
	}
	return dir, nil
}
