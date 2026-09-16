package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/infrastructure/wholefile"
	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// settingsFile is what the store writes inside the application's own directory.
const settingsFile = "settings.json"

// writingSuffix names the working file a save writes beside the target before renaming
// it into place.
const writingSuffix = ".writing"

// dirPerm and filePerm are the ordinary permissions for a per-user config file.
const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// stored is the file's shape, kept apart from ports.Settings on purpose.
//
// The file is a format and the port is a type; letting one be the other means a field
// rename in the application silently changes what is on disk and a file written by an
// older build stops being read. The json names are the contract and they are here.
type stored struct {
	LibraryRoot string `json:"libraryRoot"`
	JournalDir  string `json:"journalDir"`
	Voice       string `json:"voice"`
	// MachineVoice is absent from a file an older build wrote, which reads as no machine voice kept.
	MachineVoice string `json:"machineVoice"`
	// Plugin and PluginVoice are absent from a file an older build wrote, which reads as no plugin
	// voice kept.
	Plugin      string `json:"plugin"`
	PluginVoice string `json:"pluginVoice"`
	// SwitchedOff is absent from a file an older build wrote, which reads as every moment on (FR-628).
	SwitchedOff []string `json:"switchedOff"`
}

// Settings reads and writes the choices that outlive a run.
type Settings struct{ path string }

// NewSettings builds a store under the user's own configuration directory.
//
// A machine with no such directory is not a reason to refuse to start, so the store is
// still returned with an empty path: it then loads nothing and saving says why. The
// alternative, failing at the composition root, would stop an application that works
// perfectly well on detected paths.
func NewSettings() *Settings {
	base, err := os.UserConfigDir()
	if err != nil {
		return &Settings{}
	}
	return &Settings{path: filepath.Join(base, product.Slug, settingsFile)}
}

// Path is where the settings are kept, so a pane can say where it looked.
func (s *Settings) Path() string { return s.path }

// Load answers with what is stored. It answers with nothing at all where there is no
// file, where it cannot be read or where it does not parse.
//
// None of those three is different to a reader: in each case nothing has been chosen
// that can be honoured, so the application detects as usual. Returning an error would
// make the caller decide between "start anyway" and "refuse", where only one answer
// is sensible.
func (s *Settings) Load() ports.Settings {
	if s.path == "" {
		return ports.Settings{}
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return ports.Settings{}
	}
	var held stored
	if err := json.Unmarshal(raw, &held); err != nil {
		return ports.Settings{}
	}
	return ports.Settings{
		LibraryRoot:  held.LibraryRoot,
		JournalDir:   held.JournalDir,
		Voice:        held.Voice,
		MachineVoice: held.MachineVoice,
		Plugin:       held.Plugin,
		PluginVoice:  held.PluginVoice,
		SwitchedOff:  held.SwitchedOff,
	}
}

// Save writes the choices, creating the directory on the way.
//
// It writes through a temporary file in the same directory and renames over the
// target, so an interrupted write leaves the previous settings rather than a truncated
// file that loads as nothing.
func (s *Settings) Save(chosen ports.Settings) error {
	if s.path == "" {
		return fmt.Errorf("no configuration directory on this machine")
	}
	// The encode cannot fail: stored holds strings only; the encoder replaces
	// invalid UTF-8 rather than refusing it. The error it returns is discarded here
	// rather than checked, because a branch nothing can reach is a branch nothing can
	// test.
	raw, _ := json.MarshalIndent(stored{
		LibraryRoot:  chosen.LibraryRoot,
		JournalDir:   chosen.JournalDir,
		Voice:        chosen.Voice,
		MachineVoice: chosen.MachineVoice,
		Plugin:       chosen.Plugin,
		PluginVoice:  chosen.PluginVoice,
		SwitchedOff:  chosen.SwitchedOff,
	}, "", "  ")

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("creating %s: %w", dir, refusal.Reason(err))
	}
	return wholefile.Write(s.path, writingSuffix, filePerm, func(file io.Writer) error {
		_, err := file.Write(raw)
		return err
	})
}

// Forget removes the stored choices, for an uninstall that has been asked to forget them.
//
// Nothing stored is not a failure: there is simply nothing to forget. The working file a
// save writes goes too, in case one was interrupted. The directory is removed only once
// it is empty, since anything else found in it is not this store's to delete.
func (s *Settings) Forget() error {
	if s.path == "" {
		return nil
	}
	for _, target := range []string{s.path, s.path + writingSuffix} {
		if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("removing %s: %w", target, refusal.Reason(err))
		}
	}
	// Removing a directory that still holds something fails, which is the intent: that
	// failure is the refusal to delete what is not ours rather than a fault to report.
	_ = os.Remove(filepath.Dir(s.path))
	return nil
}
