package making_test

// FR-511 to FR-513, FR-515 and FR-522: what a machine voice still has to make.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/script"
)

// files are the style file and model digests the plans below are made against.
var files = making.Files{Style: "style-1", Model: "model-1"}

// voiced builds a script giving Docked and Undocked three lines each with saved speech sounds,
// against a table that also holds Touchdown. firstBritish is Docked's first British sounds.
func voiced(t *testing.T, firstBritish string) script.Voiced {
	t.Helper()
	var cues []cue.Cue
	for _, id := range []string{"Docked", "Undocked", "Touchdown"} {
		item, err := cue.New(cue.Definition{ID: id, Source: "journal", Event: id, Purpose: "When it happens."})
		if err != nil {
			t.Fatalf("building cue %q: %v", id, err)
		}
		cues = append(cues, item)
	}
	lines := map[string][]string{
		"Docked":   {"Docking complete.", "Down safely.", "Docked."},
		"Undocked": {"Undocked.", "Clear of the pad.", "Leaving."},
	}
	built, err := script.New(lines, cue.NewTable(cues))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	saved := map[string]script.Saved{
		"Docked":   {Lines: lines["Docked"], British: []string{firstBritish, "bɪ", "bi"}, American: []string{"æə", "æɪ", "æi"}},
		"Undocked": {Lines: lines["Undocked"], British: []string{"du", "dɪ", "di"}, American: []string{"dæ", "dæɪ", "dæi"}},
	}
	voicedScript, err := script.Voice(built, saved)
	if err != nil {
		t.Fatalf("Voice: %v", err)
	}
	return voicedScript
}

// keysOf returns the keys of the lines given.
func keysOf(lines []making.Line) []string {
	keys := make([]string, 0, len(lines))
	for _, line := range lines {
		keys = append(keys, line.Key)
	}
	return keys
}

// FR-513: a key changes with the line's speech sounds, the style file or the model. The parts
// are kept apart, so moving a symbol from the sounds to the style file changes it too.
func TestAKeyChangesWithTheSoundsTheStyleFileOrTheModel(t *testing.T) {
	base := making.Key("bə", files)
	for name, other := range map[string]string{
		"sounds":   making.Key("bɪ", files),
		"style":    making.Key("bə", making.Files{Style: "style-2", Model: files.Model}),
		"model":    making.Key("bə", making.Files{Style: files.Style, Model: "model-2"}),
		"boundary": making.Key("b", making.Files{Style: "ə" + files.Style, Model: files.Model}),
	} {
		if other == base {
			t.Errorf("a changed %s left the key unchanged", name)
		}
	}
	if making.Key("bə", files) != base {
		t.Error("the same sounds and files gave two keys")
	}
}

// FR-511 and FR-515: with nothing made, every line is to make in the voice's own accent, cue by
// cue in order; none is current.
func TestWithNothingMadeEveryLineIsToMakeInTheVoicesAccent(t *testing.T) {
	plan := making.New(voiced(t, "bə"), machinevoice.British, files, nil)
	toMake := plan.ToMake()
	if len(toMake) != 6 || plan.Current() != 0 || plan.Total() != 6 {
		t.Fatalf("to make %d, current %d of %d; want 6, 0 of 6", len(toMake), plan.Current(), plan.Total())
	}
	if want := (making.Line{Cue: "Docked", Index: 0, Sounds: "bə", Key: making.Key("bə", files)}); toMake[0] != want {
		t.Errorf("first line = %+v, want %+v", toMake[0], want)
	}
	if last := toMake[5]; last.Cue != "Undocked" || last.Index != 2 {
		t.Errorf("last line = %+v, want Undocked's third", last)
	}
	if american := making.New(voiced(t, "bə"), machinevoice.American, files, nil).ToMake()[0]; american.Sounds != "æə" {
		t.Errorf("an American voice's first sounds = %q, want æə", american.Sounds)
	}
}

// FR-511, FR-515 and FR-522: a line whose key is on disk is current and the rest are to make, in
// order. A key on disk for no line counts for nothing.
func TestLinesWithAKeyOnDiskAreCurrentAndTheRestAreToMake(t *testing.T) {
	all := making.New(voiced(t, "bə"), machinevoice.British, files, nil).ToMake()
	plan := making.New(voiced(t, "bə"), machinevoice.British, files, []string{all[0].Key, all[2].Key, "no line's key"})
	if got, want := plan.ToMake(), []making.Line{all[1], all[3], all[4], all[5]}; !slices.Equal(got, want) {
		t.Errorf("to make = %+v, want %+v", got, want)
	}
	if plan.Current() != 2 || plan.Total() != 6 {
		t.Errorf("current %d of %d, want 2 of 6", plan.Current(), plan.Total())
	}
	if plan.CuesServed() != 1 {
		t.Errorf("cues served = %d, want 1", plan.CuesServed())
	}
}

// FR-513's acceptance: a line whose speech sounds changed is made again and no other line is.
func TestALineWhoseSoundsChangedIsTheOnlyOneMadeAgain(t *testing.T) {
	made := keysOf(making.New(voiced(t, "bə"), machinevoice.British, files, nil).ToMake())
	again := making.New(voiced(t, "bəz"), machinevoice.British, files, made).ToMake()
	if len(again) != 1 || again[0].Cue != "Docked" || again[0].Index != 0 {
		t.Errorf("to make again = %+v, want Docked's first line alone", again)
	}
}

// FR-513: a new style file makes every line again.
func TestANewStyleFileMakesEveryLineAgain(t *testing.T) {
	made := keysOf(making.New(voiced(t, "bə"), machinevoice.British, files, nil).ToMake())
	plan := making.New(voiced(t, "bə"), machinevoice.British, making.Files{Style: "style-2", Model: files.Model}, made)
	if len(plan.ToMake()) != 6 || plan.Current() != 0 {
		t.Errorf("to make %d, current %d; want every line to make again", len(plan.ToMake()), plan.Current())
	}
}
