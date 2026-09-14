package services_test

// FR-511 to FR-520, FR-527 and FR-530: making a machine voice's lines over fakes of the model, the
// voice's files and the store.

import (
	"errors"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// madeFrom are the digests every made line below is keyed by.
var madeFrom = making.Files{Style: "style-1", Model: "model-1"}

// keyOf is the key a line with these sounds is made under.
func keyOf(sounds string) string { return making.Key(sounds, madeFrom) }

// voiceNamed parses a voice the test expects to be offered.
func voiceNamed(t *testing.T, id string) machinevoice.Voice {
	t.Helper()
	voice, err := machinevoice.Parse(id)
	if err != nil {
		t.Fatalf("Parse(%q): %v", id, err)
	}
	return voice
}

// material is a style of zeros under the digests above.
func material(t *testing.T) ports.Material {
	t.Helper()
	style, err := speech.NewStyle(make([]float32, speech.MaxSymbols*speech.StyleWidth))
	if err != nil {
		t.Fatalf("NewStyle: %v", err)
	}
	return ports.Material{Style: style, Files: madeFrom}
}

// makingWith builds the service over a script giving Docked and Undocked three lines each.
// lastBritish is Undocked's third British sounds; "bə" there matches Docked's first.
func makingWith(t *testing.T, lastBritish string, files fakeFiles, maker *fakeMaker, store *fakeStore) *services.MakingService {
	t.Helper()
	voiced, err := scripttest.Build(map[string]script.Saved{
		"Docked": {
			Lines:   []string{"Docking complete.", "Down safely.", "Docked."},
			British: []string{"bə", "bɪ", "bi"}, American: []string{"æə", "æɪ", "æi"},
		},
		"Undocked": {
			Lines:   []string{"Undocked.", "Clear of the pad.", "Leaving."},
			British: []string{"du", "dɪ", lastBritish}, American: []string{"dæ", "dæɪ", "dæi"},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if files.material.Files == (making.Files{}) {
		files.material = material(t)
	}
	return services.NewMakingService(voiced, files, maker, store)
}

// FR-511, FR-512, FR-514 and FR-515: casting makes every line with no current made line, in the
// voice's own accent, then answers every cue with its made lines.
func TestCastingMakesEveryLineNotYetMadeInTheVoicesAccent(t *testing.T) {
	store, maker := newFakeStore(), newFakeMaker()
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	store.hold(emma, keyOf("bə"))
	service := makingWith(t, "di", fakeFiles{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()

	if maker.made() != 5 || len(store.held(emma)) != 6 {
		t.Errorf("made %d lines leaving %d on disk; want 5 made and all 6 on disk", maker.made(), len(store.held(emma)))
	}
	if takes, ok := source.Lookup("Undocked"); !ok || len(takes) != 3 || takes[0] != store.Path(emma, keyOf("du")) {
		t.Errorf("Undocked answered %v, %v; want its three made lines", takes, ok)
	}
	got := service.Progress()
	if got.Voice != "bf_emma" || got.Making || got.Current != 6 || got.Total != 6 || got.CuesServed != 2 {
		t.Errorf("progress = %+v, want bf_emma done with 6 of 6 current over 2 cues", got)
	}

	if _, err := service.Cast(michael); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()
	if !slices.Contains(store.held(michael), keyOf("æə")) {
		t.Errorf("am_michael's lines %v were not made in his accent", store.held(michael))
	}
}

// FR-514: while making is under way, the voice answers a cue with its current made lines alone; a
// cue with none yet is silent.
func TestWhileMakingTheVoiceSpeaksOnlyWhatIsMade(t *testing.T) {
	store, maker := newFakeStore(), newFakeMaker()
	maker.blockOn = 1
	emma := voiceNamed(t, "bf_emma")
	store.hold(emma, keyOf("bə"))
	service := makingWith(t, "di", fakeFiles{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	<-maker.started
	defer service.Stop()

	if takes, ok := source.Lookup("Docked"); !ok || !slices.Equal(takes, []string{store.Path(emma, keyOf("bə"))}) {
		t.Errorf("Docked answered %v, %v; want the one line already made", takes, ok)
	}
	if _, ok := source.Lookup("Undocked"); ok {
		t.Error("a cue with nothing made answered")
	}
	if !service.Progress().Making {
		t.Error("progress does not say making is under way")
	}
}

// FR-516 and FR-527: casting another voice stops making, keeping every line written so far, before
// the other voices' lines are deleted. The line being made when it stopped is no failure; the voice
// cast before answers nothing from then on.
func TestCastingAnotherVoiceStopsMakingKeepingWhatWasWritten(t *testing.T) {
	store, maker := newFakeStore(), newFakeMaker()
	maker.blockOn = 3
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	service := makingWith(t, "di", fakeFiles{}, maker, store)

	before, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	<-maker.started
	if _, err := service.Cast(michael); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()

	if got, want := store.entries()[:4], []string{"keep bf_emma", "write bf_emma", "write bf_emma", "keep am_michael"}; !slices.Equal(got, want) {
		t.Errorf("store saw %v, want %v", got, want)
	}
	if got := service.Progress(); got.Voice != "am_michael" || len(got.Failed) != 0 || got.Current != 6 {
		t.Errorf("progress = %+v, want am_michael done with no failure", got)
	}
	if _, ok := before.Lookup("Docked"); ok {
		t.Error("the voice cast before still answered")
	}
}

// FR-518: a line that cannot be made is reported with its cue and the reason; making goes on. A
// line of no sounds is one; the model failing is another.
func TestALineThatCannotBeMadeIsReportedAndMakingGoesOn(t *testing.T) {
	store, maker := newFakeStore(), newFakeMaker()
	maker.failOn = 1
	emma := voiceNamed(t, "bf_emma")
	service := makingWith(t, "", fakeFiles{}, maker, store)

	if _, err := service.Cast(emma); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()

	got := service.Progress()
	if len(got.Failed) != 2 {
		t.Fatalf("failed = %+v, want two lines", got.Failed)
	}
	first, second := got.Failed[0], got.Failed[1]
	if first.Cue != "Docked" || first.Index != 0 || !errors.Is(first.Reason, errModel) {
		t.Errorf("first failure = %+v, want Docked's first line refused by the model", first)
	}
	if second.Cue != "Undocked" || second.Index != 2 || !errors.Is(second.Reason, speech.ErrNoSounds) {
		t.Errorf("second failure = %+v, want Undocked's third line of no sounds", second)
	}
	if got.Current != 4 || got.Making || got.Stopped != nil || len(store.held(emma)) != 4 {
		t.Errorf("progress = %+v with %d on disk; want the other 4 made", got, len(store.held(emma)))
	}
}

// FR-520: a made line that cannot be written stops making, saying why.
func TestAWriteFailureStopsMakingAndSaysWhy(t *testing.T) {
	store, maker := newFakeStore(), newFakeMaker()
	store.failWrite = 2
	emma := voiceNamed(t, "bf_emma")
	service := makingWith(t, "di", fakeFiles{}, maker, store)

	if _, err := service.Cast(emma); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()

	got := service.Progress()
	if !errors.Is(got.Stopped, errDisk) || got.Making || got.Current != 1 || maker.made() != 2 {
		t.Errorf("progress = %+v after %d made; want stopped by the disk after the second", got, maker.made())
	}
}

// FR-519: a voice whose files cannot be read is refused with the reason, changing nothing: the
// voice already cast keeps its lines and goes on answering.
func TestAVoiceWhoseFilesCannotBeReadIsRefusedChangingNothing(t *testing.T) {
	store, maker := newFakeStore(), newFakeMaker()
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	files := fakeFiles{refused: map[string]error{"am_michael": errFiles}}
	service := makingWith(t, "di", files, maker, store)
	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()
	before := store.entries()

	refused, err := service.Cast(michael)

	if !errors.Is(err, errFiles) || refused != nil {
		t.Errorf("Cast(am_michael) = %v, %v; want refused naming the file", refused, err)
	}
	if !slices.Equal(store.entries(), before) || service.Progress().Voice != "bf_emma" {
		t.Errorf("the refusal changed something: store %v, progress %+v", store.entries(), service.Progress())
	}
	if _, ok := source.Lookup("Docked"); !ok {
		t.Error("the voice already cast stopped answering")
	}
}

// FR-527 and FR-530: casting deletes every other voice's made lines; where one cannot be deleted
// the cast says so and completes.
func TestCastingDeletesOtherVoicesLinesSayingWhereItCannot(t *testing.T) {
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	for _, deleteErr := range []error{nil, errDelete} {
		store := newFakeStore()
		store.hold(michael, "an old line")
		store.deleteErr = deleteErr
		service := makingWith(t, "di", fakeFiles{}, newFakeMaker(), store)

		if _, err := service.Cast(emma); err != nil {
			t.Fatalf("Cast: %v", err)
		}
		service.Wait()

		got := service.Progress()
		if !errors.Is(got.NotDeleted, deleteErr) || got.Current != 6 {
			t.Errorf("progress = %+v, want the cast complete with NotDeleted %v", got, deleteErr)
		}
		if deleteErr == nil && len(store.held(michael)) != 0 {
			t.Errorf("am_michael's lines %v survived the cast", store.held(michael))
		}
	}
}

// FR-516, FR-527 and FR-530: casting a recorded voice stops making and deletes every made line,
// saying where it cannot; the machine voice cast before answers nothing from then on.
func TestCastingARecordedVoiceStopsMakingAndDeletesEveryLine(t *testing.T) {
	store, maker := newFakeStore(), newFakeMaker()
	maker.blockOn = 1
	emma := voiceNamed(t, "bf_emma")
	service := makingWith(t, "di", fakeFiles{}, maker, store)
	service.CastRecorded()

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	<-maker.started
	store.deleteErr = errDelete
	service.CastRecorded()

	if got, want := store.entries(), []string{"delete all", "keep bf_emma", "delete all"}; !slices.Equal(got, want) {
		t.Errorf("store saw %v, want %v", got, want)
	}
	if got := service.Progress(); got.Voice != "" || got.Making || !errors.Is(got.NotDeleted, errDelete) {
		t.Errorf("progress = %+v, want no machine voice, not making, the delete failure", got)
	}
	if _, ok := source.Lookup("Docked"); ok {
		t.Error("the machine voice cast before still answered")
	}
}

// Two lines with the same sounds share one made line: it is made once and both cues answer with it.
func TestALineSharingSoundsWithOneMadeIsMadeOnce(t *testing.T) {
	store, maker := newFakeStore(), newFakeMaker()
	emma := voiceNamed(t, "bf_emma")
	service := makingWith(t, "bə", fakeFiles{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()

	shared := store.Path(emma, keyOf("bə"))
	docked, _ := source.Lookup("Docked")
	undocked, _ := source.Lookup("Undocked")
	if maker.made() != 5 || service.Progress().Current != 6 || !slices.Contains(docked, shared) || !slices.Contains(undocked, shared) {
		t.Errorf("made %d, current %d, Docked %v, Undocked %v; want one made line shared by both",
			maker.made(), service.Progress().Current, docked, undocked)
	}
}
