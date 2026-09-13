package library

// The optional manifest: voice.toml in a voice's directory (FR-210, FR-211).
//
// A voice is found by its names alone and most voices never have a manifest. Where one is
// present it gives the voice the name it is shown by, a credit line and takes the convention
// cannot find. It only ever adds. A manifest that cannot be used costs the voice nothing its
// names found (FR-211); one entry that cannot be used costs nothing but itself.

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/infrastructure/audio"
	"github.com/oernster/bridge-talk/internal/infrastructure/tomlfile"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// ManifestFile is the manifest's name inside a voice directory.
const ManifestFile = "voice.toml"

// manifestShape is the shape of voice.toml, REQUIREMENTS.md section 3.3. A key it does not
// hold makes the file malformed rather than being dropped in silence.
type manifestShape struct {
	Name   string              `toml:"name"`
	Credit string              `toml:"credit"`
	Takes  map[string][]string `toml:"takes"`
}

// manifest reads a voice's manifest. A voice with none answers with an empty shape and no
// reason; so does one whose manifest cannot be used, except that it names why (FR-211). Either
// way the scan goes on by the voice's names.
func (from disk) manifest(dir string) (manifestShape, []Reason) {
	raw, err := from.readFile(filepath.Join(dir, ManifestFile))
	if errors.Is(err, fs.ErrNotExist) {
		return manifestShape{}, nil
	}
	if err != nil {
		return manifestShape{}, []Reason{unusable(fmt.Sprintf("this cannot be read (%v)", refusal.Reason(err)))}
	}
	var shape manifestShape
	if err := tomlfile.Decode(raw, &shape); err != nil {
		return manifestShape{}, []Reason{unusable(fmt.Sprintf("this cannot be used (%v)", err))}
	}
	return shape, nil
}

// Display is the name the voice is shown by: the name its manifest gives; its directory's
// where there is none (FR-210). A voice built without a scan is shown by its directory too.
func (v Voice) Display() string {
	if v.display == "" {
		return v.Name
	}
	return v.display
}

// unusable names a manifest the scan could not use at all.
func unusable(why string) Reason {
	return Reason{Path: ManifestFile, Why: why + "; the voice is found by its names alone"}
}

// declare applies a manifest to the voice its directory holds: the name it is shown by, its
// credit and the takes it declares (FR-210). It answers with where each take it added sits
// inside the voice's directory.
//
// Keys are read in sorted order, so the report names what was passed over in the same order on
// every run. A declared take is decoded as every other take is, so one that will not play is
// reported where FR-204 reports it; everything else wrong with an entry belongs to the manifest.
func (from disk) declare(dir string, shape manifestShape, lookup map[string]cue.ID, voice *Voice, report *Report) []string {
	if name := strings.TrimSpace(shape.Name); name != "" {
		voice.display = name
	}
	voice.Credit = strings.TrimSpace(shape.Credit)

	keys := make([]string, 0, len(shape.Takes))
	for key := range shape.Takes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var reached []string
	for _, key := range keys {
		id, ok := lookup[strings.ToLower(key)]
		if !ok {
			report.Manifest = append(report.Manifest, Reason{
				Path: ManifestFile,
				Why:  fmt.Sprintf("%q is no cue's id, so the takes listed under it are never played", key),
			})
			continue
		}
		for _, written := range shape.Takes[key] {
			found := filepath.Clean(filepath.FromSlash(written))
			if why, refused := outside(written, found); refused {
				report.Manifest = append(report.Manifest, Reason{Path: ManifestFile, Why: why})
				continue
			}
			if reason, bad := from.refuse(dir, found); bad {
				report.Undecodable = append(report.Undecodable, reason)
				continue
			}
			voice.byCue[id] = append(voice.byCue[id], filepath.Join(dir, found))
			reached = append(reached, found)
		}
	}
	return reached
}

// outside says why a declared path is not read, before anything is opened: it is not inside
// the voice's directory; it is not a recording the player decodes (FR-203).
func outside(written, found string) (string, bool) {
	if !filepath.IsLocal(found) {
		return fmt.Sprintf("%q is not inside the voice's directory, so it is not read", written), true
	}
	if !audio.Recognised(found) {
		return fmt.Sprintf("%q is not a recording the player decodes, so it is not read", written), true
	}
	return "", false
}

// unclaimed drops from what the names could not match everything the manifest reached: a file
// it declares and a folder holding one. Reporting either as never played would be false
// (FR-208, FR-210). Names are compared ignoring case, since a declared path in another case
// reaches the file only where the platform ignores case too.
func unclaimed(unmatched []Reason, reached []string) []Reason {
	var out []Reason
	for _, reason := range unmatched {
		if !claimed(reason.Path, reached) {
			out = append(out, reason)
		}
	}
	return out
}

// claimed reports whether a path is one the manifest reached or a folder holding one.
func claimed(path string, reached []string) bool {
	lowered := strings.ToLower(path)
	for _, found := range reached {
		found = strings.ToLower(found)
		if found == lowered || strings.HasPrefix(found, lowered+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
