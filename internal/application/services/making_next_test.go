package services_test

// FR-514: a cue's lines made next when the cue fires with none made, starting a run where none is
// going.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
)

// cueMaker is the audio source's way of making a cue's lines on call.
func cueMaker(t *testing.T, source ports.AudioSource) ports.CueMaker {
	t.Helper()
	maker, ok := source.(ports.CueMaker)
	if !ok {
		t.Fatalf("the audio source %T makes no line on call", source)
	}
	return maker
}

// keys gives the key of each sounds, in order.
func keys(sounds ...string) []string {
	out := make([]string, 0, len(sounds))
	for _, each := range sounds {
		out = append(out, makingtest.Key(each))
	}
	return out
}

// A cue's unmade lines are made next, first line first, after the line already being made; each line
// is made once.
func TestMakeNextPutsACuesUnmadeLinesAheadOfTheRest(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	maker.PauseOn = 1
	emma := voiceNamed(t, "bf_emma")
	store.Hold(emma.ID(), makingtest.Key("dɪ"))
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	makingtest.Await(t, maker.Started, "making starting")
	if !cueMaker(t, source).MakeNext("Undocked") {
		t.Fatal("MakeNext(Undocked) said no line is on its way")
	}
	close(maker.Resume)
	service.Wait()

	if got, want := store.Held(emma.ID()), keys("dɪ", "bə", "du", "di", "bɪ", "bi"); !slices.Equal(got, want) {
		t.Errorf("made %v, want Undocked's two unmade lines straight after the one under way: %v", got, want)
	}
	if maker.Made() != 5 {
		t.Errorf("made %d lines, want each of the 5 unmade lines once", maker.Made())
	}
}

// With nothing being made, a cue asked for starts making of its own.
func TestMakeNextStartsMakingWhenNothingIsUnderWay(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	emma := voiceNamed(t, "bf_emma")
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()
	if service.Progress().Making {
		t.Fatal("making was still under way once the confirmation was made")
	}

	if !cueMaker(t, source).MakeNext("Undocked") {
		t.Fatal("MakeNext(Undocked) said no line is on its way")
	}
	service.Wait()

	if got := service.Progress(); got.Current != 6 || got.Making {
		t.Errorf("progress = %+v, want every line made and making over", got)
	}
}

// No line is on its way for a cue with none to make, a cue the script lacks, a cast that is over or a
// cast stopped as the application closes.
func TestMakeNextAnswersWhetherALineIsOnItsWay(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	maker.PauseOn = 1
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	store.Hold(emma.ID(), keys("du", "dɪ", "di")...)
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	before, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	makingtest.Await(t, maker.Started, "making starting")
	asked := cueMaker(t, before)
	if asked.MakeNext("Undocked") || asked.MakeNext("Scanned") {
		t.Error("a line was on its way for a cue with every line made or no lines at all")
	}
	after, err := service.Cast(michael)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	if asked.MakeNext("Docked") {
		t.Error("a line was on its way for a cast that is over")
	}

	service.Stop()
	if cueMaker(t, after).MakeNext("Undocked") {
		t.Error("a line was on its way once making was stopped")
	}
}
