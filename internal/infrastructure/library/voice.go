// Package library discovers voices on disk and resolves cues into takes.
//
// A voice is one person's recordings, held in a directory the user chose. What makes
// a directory a cue's takes is that its name IS the cue id, compared as whole strings
// and case insensitively, with nothing normalised away. Everything in the package
// rests on that rule: a name either is a cue id or it is not, so the scan report can
// say of every file it found whether it was used and if not why not, with no third
// answer. It also means a directory of audio organised for some other purpose
// contributes nothing by accident, which is what makes it safe to point the
// application at a directory and simply see what happens.
package library

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// audioExtensions are the formats the player can decode.
var audioExtensions = map[string]struct{}{
	".mp3":  {},
	".wav":  {},
	".flac": {},
	".ogg":  {},
}

// lister reads the entries of one directory. Scan and ScanVoice pass os.ReadDir.
//
// It is a parameter rather than a direct call because one rule cannot be exercised on
// a Windows disk at all: two cue directories differing only in case (FR-218). Windows
// refuses to create the second beside the first, measured, so the test that holds the
// merge hands the scan a tree held in memory instead.
type lister func(dir string) ([]os.DirEntry, error)

// Voice is one person's recordings, indexed by the cue each answers.
type Voice struct {
	// Name is the voice as a reader meets it: the directory's own name.
	Name string
	// Root is the directory the recordings sit in.
	Root string
	// Takes counts the playable files resolved to a cue.
	Takes int

	byCue map[cue.ID][]string
}

// Lookup returns the takes recorded for a cue; false when the voice has none.
func (v Voice) Lookup(id cue.ID) ([]string, bool) {
	clips, ok := v.byCue[id]
	if !ok || len(clips) == 0 {
		return nil, false
	}
	return clips, true
}

// Cues counts the cues this voice has at least one take for.
func (v Voice) Cues() int { return len(v.byCue) }

// Reason says why something in a voice's directory was passed over.
type Reason struct {
	// Path is what was found, relative to the directory scanned.
	Path string
	// Why says what was wrong with it, in words a reader can act on.
	Why string
}

// Report is everything a scan passed over, so nothing is skipped in silence.
//
// A user who has just pointed the application at the wrong directory gets the same
// answer without it as one who misspelled a cue in a folder name: an empty list.
// The report is the difference between "nothing here" and "here is what I found and
// could not use".
type Report struct {
	// Unmatched names entries whose names are no cue id.
	Unmatched []Reason
	// Empty names candidate voices that resolved no take at all.
	Empty []Reason
	// Duplicated names cue directories that differ only by case, which one machine
	// permits and another refuses.
	Duplicated []Reason
}

// Any reports whether the scan passed anything over.
func (r Report) Any() bool {
	return len(r.Unmatched) > 0 || len(r.Empty) > 0 || len(r.Duplicated) > 0
}

// index maps a cue id lowered to the id itself, which is how an exact but case
// insensitive comparison is made without normalising anything else away.
func index(table cue.Table) map[string]cue.ID {
	all := table.All()
	out := make(map[string]cue.ID, len(all))
	for _, item := range all {
		out[strings.ToLower(string(item.ID()))] = item.ID()
	}
	return out
}

// folderIndex maps a cue folder's name lowered to the id it belongs to (FR-229). A folder
// is matched on the id with its dots written as underscores, so a folder named with dots
// is no cue's folder. The flat form keeps the dotted id, which index holds.
func folderIndex(table cue.Table) map[string]cue.ID {
	all := table.All()
	out := make(map[string]cue.ID, len(all))
	for _, item := range all {
		out[strings.ToLower(item.ID().Folder())] = item.ID()
	}
	return out
}

// Scan reads a library root and returns the voices in it.
//
// Every immediate subdirectory is a candidate. One becomes a voice when at least one
// take resolves inside it; the rest are reported rather than dropped, because a
// directory holding no recognised name and a directory holding none of the right
// names look identical from the outside.
func Scan(root string, table cue.Table) ([]Voice, Report, error) {
	return scan(root, table, os.ReadDir)
}

func scan(root string, table cue.Table, read lister) ([]Voice, Report, error) {
	entries, err := read(root)
	if err != nil {
		return nil, Report{}, err
	}

	lookup := index(table)
	folders := folderIndex(table)
	var voices []Voice
	var report Report

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		voice, found := scanVoice(filepath.Join(root, entry.Name()), lookup, folders, read)
		report.Unmatched = append(report.Unmatched, prefix(entry.Name(), found.Unmatched)...)
		report.Duplicated = append(report.Duplicated, prefix(entry.Name(), found.Duplicated)...)
		if voice.Takes == 0 {
			report.Empty = append(report.Empty, Reason{
				Path: entry.Name(),
				Why:  "no recording here is named for a cue or sits in a folder that is, so there is nothing to play",
			})
			continue
		}
		voices = append(voices, voice)
	}

	sort.Slice(voices, func(a, b int) bool {
		return strings.ToLower(voices[a].Name) < strings.ToLower(voices[b].Name)
	})
	return voices, report, nil
}

// prefix re-roots a voice's own reasons under the library root, so a path in the
// report is one the reader can find.
func prefix(dir string, reasons []Reason) []Reason {
	out := make([]Reason, 0, len(reasons))
	for _, reason := range reasons {
		out = append(out, Reason{Path: filepath.Join(dir, reason.Path), Why: reason.Why})
	}
	return out
}

// ScanVoice reads one voice directory. It is exported so a caller holding a directory
// rather than a library root can index it without inventing a parent.
func ScanVoice(dir string, table cue.Table) (Voice, Report) {
	return scanVoice(dir, index(table), folderIndex(table), os.ReadDir)
}

func scanVoice(dir string, lookup, folders map[string]cue.ID, read lister) (Voice, Report) {
	voice := Voice{Name: filepath.Base(dir), Root: dir, byCue: map[cue.ID][]string{}}
	var report Report

	entries, err := read(dir)
	if err != nil {
		return voice, report
	}

	seen := map[cue.ID]string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			id, ok := folders[strings.ToLower(name)]
			if !ok {
				report.Unmatched = append(report.Unmatched, Reason{
					Path: name, Why: "this is no cue's name, so nothing inside it is reachable",
				})
				continue
			}
			// Linux permits two directories differing only in case; Windows and macOS
			// refuse. Merging is deterministic and loses nothing, where choosing one
			// would make identical files behave differently on two machines.
			if first, clash := seen[id]; clash && first != name {
				report.Duplicated = append(report.Duplicated, Reason{
					Path: name,
					Why:  "differs from " + first + " only in case; their takes are merged",
				})
			}
			seen[id] = name
			voice.byCue[id] = append(voice.byCue[id], takesIn(filepath.Join(dir, name), read)...)
			continue
		}
		if id, ok := flatTake(name, lookup); ok {
			voice.byCue[id] = append(voice.byCue[id], filepath.Join(dir, name))
			continue
		}
		if _, audio := audioExtensions[strings.ToLower(filepath.Ext(name))]; audio {
			report.Unmatched = append(report.Unmatched, Reason{
				Path: name, Why: "this is no cue's name, so it is never played",
			})
		}
	}

	for id, clips := range voice.byCue {
		if len(clips) == 0 {
			delete(voice.byCue, id)
			continue
		}
		sort.Strings(clips)
		voice.byCue[id] = clips
		voice.Takes += len(clips)
	}
	return voice, report
}

// flatTake reads a file name as a take for a cue.
//
// The base name is the cue id, optionally followed by a dot and digits to tell one
// take from another, so docking.granted.wav and docking.granted.2.wav are two takes of
// the same line. That is unambiguous only because no cue id ends in a segment of
// digits, which is a property of the vocabulary rather than a law, so a structural
// test holds it.
func flatTake(name string, lookup map[string]cue.ID) (cue.ID, bool) {
	extension := strings.ToLower(filepath.Ext(name))
	if _, ok := audioExtensions[extension]; !ok {
		return "", false
	}
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if id, ok := lookup[strings.ToLower(base)]; ok {
		return id, true
	}
	if cut := strings.LastIndexByte(base, '.'); cut > 0 && allDigits(base[cut+1:]) {
		if id, ok := lookup[strings.ToLower(base[:cut])]; ok {
			return id, true
		}
	}
	return "", false
}

func allDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, letter := range text {
		if letter < '0' || letter > '9' {
			return false
		}
	}
	return true
}

// takesIn returns the playable files directly inside a directory. Nothing deeper is
// read: a cue's takes are its own files, so a folder inside one is not a take.
func takesIn(dir string, read lister) []string {
	entries, err := read(dir)
	if err != nil {
		return nil
	}
	var clips []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, ok := audioExtensions[strings.ToLower(filepath.Ext(entry.Name()))]; ok {
			clips = append(clips, filepath.Join(dir, entry.Name()))
		}
	}
	return clips
}
