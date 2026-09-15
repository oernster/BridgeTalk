package config_test

// FR-529: a line that gives a word's speech sounds is spoken with them. Oliver heard the
// confirmation's last line end as "on't" in British voices and chose a fully stressed "on" by ear on
// 2026-09-15.

import (
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
)

const (
	// confirmation is the cue played on a cast.
	confirmation cue.ID = "Cast.Confirmed"
	// hereOnLine is the place of "You'll be hearing from me from here on." among its lines.
	hereOnLine = 2
	// stressedOn is how that line's British sounds end.
	stressedOn = "hˈɪə ˈɒn."
)

func TestTheConfirmationEndsOnAStressedOnForBritishVoices(t *testing.T) {
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("LoadCueTable: %v", err)
	}
	voiced, err := config.LoadVoicedScript(table)
	if err != nil {
		t.Fatalf("LoadVoicedScript: %v", err)
	}
	sounds, ok := voiced.Sounds(confirmation, machinevoice.British)
	if !ok || len(sounds) <= hereOnLine {
		t.Fatalf("%s has %d British sounds, want a line at %d", confirmation, len(sounds), hereOnLine)
	}
	if !strings.HasSuffix(sounds[hereOnLine], stressedOn) {
		t.Errorf("British sounds = %q, want them to end %q", sounds[hereOnLine], stressedOn)
	}
}
