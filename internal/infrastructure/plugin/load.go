package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// Opener loads one library file. The real one is OpenLibrary; a test hands in its own, so
// everything here is exercised without a library file existing.
type Opener func(path string) (Library, error)

// lister reads the entries of one directory. Load passes os.ReadDir.
//
// It is a parameter for the reason library.lister is one: a rule here cannot be exercised
// on a Windows disk at all. A folder that cannot be read has to be told apart from one that
// is not there; on Windows reading a file as a directory answers that it is not there,
// measured on 2026-09-16. So the only way to reach the refusal is to hand the loader a
// reader that refuses.
type lister func(dir string) ([]os.DirEntry, error)

// Refusal is one plugin passed over, named with the reason it was passed over for.
//
// Every one of these reaches the run log (FR-567). A plugin present and silent is the case
// a user cannot diagnose without being told.
type Refusal struct {
	// File is the file's own name, without the folder, which is what the user sees when
	// they open the plugins folder.
	File string
	// Why says what was wrong with it, in words a reader can act on.
	Why string
}

// Set is every plugin loaded from one folder, plus the thread they are all called on.
type Set struct {
	Plugins  []*Plugin
	Refusals []Refusal

	on *runner
}

// Voices lists every voice every loaded plugin offers.
func (s *Set) Voices() []*Voice {
	var out []*Voice
	for _, each := range s.Plugins {
		out = append(out, each.voices...)
	}
	return out
}

// Close ends the plugin thread. The plugins themselves are never unloaded, for the reason
// ONNX Runtime is never unloaded: whether a library can be unloaded safely while its own
// threads may still run has not been measured.
func (s *Set) Close() {
	if s.on != nil {
		s.on.close()
	}
}

// Load loads every plugin in dir, answering those that loaded and those that did not.
//
// An absent folder is not a fault: almost every user has no plugin, so nothing about the
// ordinary case should mention them (FR-562). A folder that cannot be read is a fault worth
// naming, since the user put something there and it is not being seen.
//
// One file failing never stops another (FR-561), so every file is tried and every refusal
// is kept. Files are taken in name order, so what a user sees does not depend on the order
// a directory happens to list.
func Load(dir string, open Opener) *Set { return load(dir, open, os.ReadDir) }

// load is Load with the directory read handed in.
func load(dir string, open Opener, read lister) *Set {
	set := &Set{on: newRunner()}
	entries, err := read(dir)
	if os.IsNotExist(err) {
		set.on.close()
		return set
	}
	if err != nil {
		set.on.close()
		set.Refusals = []Refusal{{File: filepath.Base(dir), Why: refusal.Reason(err).Error()}}
		return set
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		loaded, why := set.one(filepath.Join(dir, name), name, open)
		if why != "" {
			set.Refusals = append(set.Refusals, Refusal{File: name, Why: why})
			continue
		}
		set.Plugins = append(set.Plugins, loaded)
	}
	if len(set.Plugins) == 0 {
		set.on.close()
	}
	return set
}

// one loads a single file, answering either the plugin or why it was passed over.
func (s *Set) one(path, name string, open Opener) (*Plugin, string) {
	library, err := open(path)
	if err != nil {
		return nil, err.Error()
	}
	loaded := &Plugin{File: name, library: library, on: s.on}

	var version int32
	s.on.do(func() { version = library.Version() })
	if version != ABIVersion {
		return nil, fmt.Sprintf(
			"it was built against interface version %d; this is %s %d",
			version, product.Name, ABIVersion)
	}

	answer, err := loaded.ask(library.Describe)
	if err != nil {
		return nil, "it would not say what it is: " + err.Error()
	}
	described, err := DecodeDescription(answer)
	if err != nil {
		return nil, "what it said about itself will not read: " + err.Error()
	}
	if len(described.Voices) == 0 {
		return nil, "it offers no voice"
	}

	loaded.Name = described.Name
	for index, voice := range described.Voices {
		if voice.Name == "" || voice.ID == "" {
			return nil, fmt.Sprintf("its voice %d has no %s", index+1, missing(voice))
		}
		reason := voice.Reason
		if !voice.Ready && reason == "" {
			reason = noReasonGiven
		}
		loaded.voices = append(loaded.voices, &Voice{
			ID: voice.ID, Name: voice.Name, Ready: voice.Ready, Reason: reason,
			plugin: loaded, index: int32(index),
		})
	}
	return loaded, ""
}

// noReasonGiven stands in for the reason a voice that cannot speak did not give. It is filled in
// here, where the voice is read, so the Cast pane, a refused cast and the log all say it rather than
// trailing off after a colon (FR-570). A plugin may leave the reason empty; nothing checks that it
// does not.
const noReasonGiven = "it gave no reason"

// missing says which of the two things a voice has to have it is without, so the refusal
// tells the plugin's author which one to put right.
func missing(voice VoiceInfo) string {
	if voice.Name == "" {
		return "name"
	}
	return "id"
}
