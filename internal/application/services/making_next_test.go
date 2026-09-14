package services_test

// FR-511 and FR-514: the order a machine voice's lines are made in; a cue's lines made next when
// the cue fires with none made.

import (
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// makingInOrder builds the service over twoCues, making cues in the order given.
func makingInOrder(t *testing.T, order []cue.ID, maker *makingtest.Maker, store *makingtest.Store) *services.MakingService {
	t.Helper()
	files := makingtest.Files{Material: makingtest.Material()}
	return services.NewMakingService(twoCues(t, "di"), order, files, maker, store)
}

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

// FR-511: a cast makes the cues in the order given.
func TestLinesAreMadeInTheOrderGiven(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	emma := voiceNamed(t, "bf_emma")
	service := makingInOrder(t, []cue.ID{"Undocked"}, maker, store)

	if _, err := service.Cast(emma); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()

	if got, want := store.Held(emma.ID()), keys("du", "dɪ", "di", "bə", "bɪ", "bi"); !slices.Equal(got, want) {
		t.Errorf("made %v, want Undocked's lines before Docked's: %v", got, want)
	}
}

// FR-514: a cue's unmade lines are made next, first line first, after the line already being made;
// each line is made once.
func TestMakeNextPutsACuesUnmadeLinesAheadOfTheRest(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	maker.PauseOn = 1
	emma := voiceNamed(t, "bf_emma")
	store.Hold(emma.ID(), makingtest.Key("dɪ"))
	service := makingInOrder(t, []cue.ID{"Docked", "Undocked"}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	<-maker.Started
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

// FR-514: no line is on its way for a cue with none to make, a cue the script lacks, a cue asked for
// once making has ended or a cast that is over.
func TestMakeNextAnswersWhetherALineIsOnItsWay(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	maker.PauseOn = 1
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	store.Hold(emma.ID(), keys("du", "dɪ", "di")...)
	service := makingInOrder(t, nil, maker, store)

	before, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	<-maker.Started
	asked := cueMaker(t, before)
	if asked.MakeNext("Undocked") || asked.MakeNext("Scanned") {
		t.Error("a line was on its way for a cue with every line made or no lines at all")
	}
	if _, err := service.Cast(michael); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	if asked.MakeNext("Docked") {
		t.Error("a line was on its way for a cast that is over")
	}
	service.Wait()

	// am_michael's six lines took calls 2 to 7; bf_emma's lines went with his cast, so her first line
	// is call 8, which fails and stays unmade once making ends.
	maker.FailOn = 8
	after, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()
	if cueMaker(t, after).MakeNext("Docked") {
		t.Error("a line was on its way once making had ended")
	}
}
