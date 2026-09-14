package making_test

// FR-511 to FR-515 and FR-522: what a machine voice still has to make.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/pause"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
)

// files are the style file and model digests the plans below are made against.
var files = making.Files{Style: "style-1", Model: "model-1"}

// emma and michael are the British and the American voice the plans below are made for; noPauses is
// a book giving neither a pause.
var (
	emma     = parsed("bf_emma")
	michael  = parsed("am_michael")
	noPauses = pause.Book{}
)

// parsed parses a voice every test here relies on being offered.
func parsed(id string) machinevoice.Voice {
	voice, err := machinevoice.Parse(id)
	if err != nil {
		panic(err)
	}
	return voice
}

// voiced builds a script giving Docked and Undocked three lines each with saved speech sounds.
// firstBritish is Docked's first British sounds.
func voiced(t *testing.T, firstBritish string) script.Voiced {
	t.Helper()
	built, err := scripttest.Build(map[string]script.Saved{
		"Docked": {
			Lines:   []string{"Docking complete.", "Down safely.", "Docked."},
			British: []string{firstBritish, "bɪ", "bi"}, American: []string{"æə", "æɪ", "æi"},
		},
		"Undocked": {
			Lines:   []string{"Undocked.", "Clear of the pad.", "Leaving."},
			British: []string{"du", "dɪ", "di"}, American: []string{"dæ", "dæɪ", "dæi"},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return built
}

// keysOf returns the keys of the lines given.
func keysOf(lines []making.Line) []string {
	keys := make([]string, 0, len(lines))
	for _, line := range lines {
		keys = append(keys, line.Key)
	}
	return keys
}

// FR-514: a cue's lines still to make come in line order; a line already made is not among them.
func TestUnmadeGivesACuesLinesStillToMakeInLineOrder(t *testing.T) {
	all := making.New(voiced(t, "bə"), emma, files, noPauses, nil).ToMake()
	plan := making.New(voiced(t, "bə"), emma, files, noPauses, []string{all[1].Key})

	got := plan.Unmade("Docked")

	if len(got) != 2 || got[0] != all[0] || got[1] != all[2] {
		t.Errorf("Unmade(Docked) = %+v, want its first and third lines", got)
	}
	if none := plan.Unmade("Scanned"); len(none) != 0 {
		t.Errorf("Unmade of a cue with no lines = %+v, want none", none)
	}
}

// FR-527: the keys on disk that no line holds are stale, in the order they were given; a key a line
// holds is not, whether or not it is in the order given.
func TestStaleGivesTheKeysOnDiskNoLineHolds(t *testing.T) {
	all := making.New(voiced(t, "bə"), emma, files, noPauses, nil).ToMake()
	onDisk := []string{all[0].Key, "an old key", all[4].Key, "another old key"}

	got := making.New(voiced(t, "bə"), emma, files, noPauses, onDisk).Stale()

	if want := []string{"an old key", "another old key"}; !slices.Equal(got, want) {
		t.Errorf("Stale = %v, want %v", got, want)
	}
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
	plan := making.New(voiced(t, "bə"), emma, files, noPauses, nil)
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
	if american := making.New(voiced(t, "bə"), michael, files, noPauses, nil).ToMake()[0]; american.Sounds != "æə" {
		t.Errorf("an American voice's first sounds = %q, want æə", american.Sounds)
	}
}

// FR-511, FR-515 and FR-522: a line whose key is on disk is current and the rest are to make, in
// order. A key on disk for no line counts for nothing.
func TestLinesWithAKeyOnDiskAreCurrentAndTheRestAreToMake(t *testing.T) {
	all := making.New(voiced(t, "bə"), emma, files, noPauses, nil).ToMake()
	plan := making.New(voiced(t, "bə"), emma, files, noPauses, []string{all[0].Key, all[2].Key, "no line's key"})
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
	made := keysOf(making.New(voiced(t, "bə"), emma, files, noPauses, nil).ToMake())
	again := making.New(voiced(t, "bəz"), emma, files, noPauses, made).ToMake()
	if len(again) != 1 || again[0].Cue != "Docked" || again[0].Index != 0 {
		t.Errorf("to make again = %+v, want Docked's first line alone", again)
	}
}

// FR-513: a new style file makes every line again.
func TestANewStyleFileMakesEveryLineAgain(t *testing.T) {
	made := keysOf(making.New(voiced(t, "bə"), emma, files, noPauses, nil).ToMake())
	plan := making.New(voiced(t, "bə"), emma, making.Files{Style: "style-2", Model: files.Model}, noPauses, made)
	if len(plan.ToMake()) != 6 || plan.Current() != 0 {
		t.Errorf("to make %d, current %d; want every line to make again", len(plan.ToMake()), plan.Current())
	}
}

// FR-514 and FR-515: a line made while making is under way is current in the plan WithMade
// answers, which counts it and hands its key out for its cue. The plan it came from is unchanged.
func TestAMadeLineIsCurrentInThePlanThatHoldsIt(t *testing.T) {
	plan := making.New(voiced(t, "bə"), emma, files, noPauses, nil)
	first := plan.ToMake()[0]

	made := plan.WithMade(first.Key)

	if !made.Made(first.Key) || made.Current() != 1 || !slices.Equal(made.Takes("Docked"), []string{first.Key}) {
		t.Errorf("the plan holding the made line: made %v, current %d, takes %v",
			made.Made(first.Key), made.Current(), made.Takes("Docked"))
	}
	if plan.Made(first.Key) || plan.Current() != 0 || len(plan.Takes("Docked")) != 0 {
		t.Error("the plan WithMade was asked of changed")
	}
}

// FR-514: a cue's takes are the distinct keys of its current lines in line order. Two lines with
// the same sounds share one made line, so it is one take; a cue with nothing current has none.
func TestACuesTakesAreTheDistinctKeysOfItsCurrentLines(t *testing.T) {
	all := making.New(voiced(t, "bi"), emma, files, noPauses, nil).ToMake()
	plan := making.New(voiced(t, "bi"), emma, files, noPauses, []string{all[2].Key, all[1].Key})

	if got, want := plan.Takes("Docked"), []string{all[0].Key, all[1].Key}; !slices.Equal(got, want) {
		t.Errorf("Docked's takes = %v, want %v", got, want)
	}
	for _, id := range []cue.ID{"Undocked", "Touchdown"} {
		if got := plan.Takes(id); len(got) != 0 {
			t.Errorf("%s's takes = %v, want none", id, got)
		}
	}
}
