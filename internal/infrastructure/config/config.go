// Package config loads the shipped cue table.
//
// It is embedded in the binary so the application runs from a single executable. It
// can be overridden from disk so a cue can be retuned without a rebuild.
package config

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/refusal"
)

//go:embed cues.toml
var embeddedCues []byte

// cueFile is the shape of cues.toml.
//
// There is no title. A title is generated from the id, so a table that writes one is
// refused along with any other key this shape does not hold, rather than the key being
// dropped in silence and the table appearing to say something it does not. The purpose is
// the exception: it says when the cue is heard, which no id can, so it is written by hand
// and every cue must carry one (FR-231).
type cueFile struct {
	Cue []struct {
		ID       string            `toml:"id"`
		Source   string            `toml:"source"`
		Event    string            `toml:"event"`
		Flag     string            `toml:"flag"`
		Edge     string            `toml:"edge"`
		Match    map[string]string `toml:"match"`
		Priority string            `toml:"priority"`
		Cooldown int               `toml:"cooldown"`
		Purpose  string            `toml:"purpose"`
	} `toml:"cue"`
}

// LoadCueTable parses the shipped cue table; or a replacement from disk.
func LoadCueTable(override string) (cue.Table, error) {
	raw, err := contents(override, embeddedCues)
	if err != nil {
		return cue.Table{}, err
	}
	var parsed cueFile
	meta, err := toml.Decode(string(raw), &parsed)
	if err != nil {
		return cue.Table{}, fmt.Errorf("parsing cue table: %w", err)
	}
	if unknown := meta.Undecoded(); len(unknown) > 0 {
		return cue.Table{}, fmt.Errorf("%w: unknown key %s", cue.ErrInvalidCue, unknown[0])
	}

	seen := make(map[string]struct{}, len(parsed.Cue))
	cues := make([]cue.Cue, 0, len(parsed.Cue))
	for _, entry := range parsed.Cue {
		if _, duplicate := seen[entry.ID]; duplicate {
			return cue.Table{}, fmt.Errorf("%w: duplicate id %q", cue.ErrInvalidCue, entry.ID)
		}
		seen[entry.ID] = struct{}{}
		built, err := cue.New(cue.Definition{
			ID:       entry.ID,
			Source:   entry.Source,
			Event:    entry.Event,
			Flag:     entry.Flag,
			Edge:     entry.Edge,
			Match:    entry.Match,
			Priority: entry.Priority,
			Cooldown: time.Duration(entry.Cooldown) * time.Second,
			Purpose:  entry.Purpose,
		})
		if err != nil {
			return cue.Table{}, err
		}
		if strings.TrimSpace(entry.Purpose) == "" {
			return cue.Table{}, fmt.Errorf(
				"%w: %s has no purpose, the sentence saying when the cue is heard",
				cue.ErrInvalidCue, entry.ID,
			)
		}
		cues = append(cues, built)
	}
	return cue.NewTable(cues), nil
}

// contents returns an override file's bytes; or the embedded default.
func contents(override string, embedded []byte) ([]byte, error) {
	if override == "" {
		return embedded, nil
	}
	raw, err := os.ReadFile(override)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", override, refusal.Reason(err))
	}
	return raw, nil
}
