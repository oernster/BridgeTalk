package services_test

// FR-511 to FR-520, FR-527 and FR-530: making a machine voice's lines over fakes of the model, the
// voice's files and the store. Docked stands for the confirmation throughout.

import (
	"errors"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/pause"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
	"github.com/oernster/bridge-talk/internal/domain/speech"
)

// The reasons the tests below refuse a voice's files and a delete with.
var (
	errFiles  = errors.New("am_michael.bin is missing")
	errDelete = errors.New("a made line is in use")
)

// voiceNamed parses a voice the test expects to be offered.
func voiceNamed(t *testing.T, id string) machinevoice.Voice {
	t.Helper()
	voice, err := machinevoice.Parse(id)
	if err != nil {
		t.Fatalf("Parse(%q): %v", id, err)
	}
	return voice
}

// makingWith builds the service over twoCues with Docked as the confirmation.
func makingWith(t *testing.T, lastBritish string, files makingtest.Files, maker *makingtest.Maker, store *makingtest.Store) *services.MakingService {
	t.Helper()
	if files.Material.Files == (making.Files{}) {
		files.Material = makingtest.Material()
	}
	return services.NewMakingService(twoCues(t, lastBritish), pause.Book{}, "Docked", files, maker, store, &makingtest.Log{})
}

// twoCues is a script giving Docked and Undocked three lines each. lastBritish is Undocked's third
// British sounds; "bə" there matches Docked's first.
func twoCues(t *testing.T, lastBritish string) script.Voiced {
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
	return voiced
}

// FR-511, FR-512 and FR-515: a cast makes the confirmation's lines not yet made in the voice's own
// accent, making no other line; with nothing stale, nothing is deleted.
func TestCastingMakesOnlyTheConfirmationsUnmadeLines(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	store.Hold(emma.ID(), makingtest.Key("bə"))
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()

	if got, want := store.Held(emma.ID()), keys("bə", "bɪ", "bi"); maker.Made() != 2 || !slices.Equal(got, want) {
		t.Errorf("made %d leaving %v; want the confirmation's 2 unmade lines: %v", maker.Made(), got, want)
	}
	if takes, ok := source.Lookup("Docked"); !ok || len(takes) != 3 {
		t.Errorf("Docked answered %v, %v; want its three made lines", takes, ok)
	}
	if _, ok := source.Lookup("Undocked"); ok {
		t.Error("a cue nothing asked for was made")
	}
	got := service.Progress()
	if got.Voice != "bf_emma" || got.Making || got.Current != 3 || got.Total != 6 || got.CuesServed != 1 {
		t.Errorf("progress = %+v, want bf_emma with 3 of 6 current over 1 cue", got)
	}

	if _, err := service.Cast(michael); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()
	if !slices.Contains(store.Held(michael.ID()), makingtest.Key("æə")) {
		t.Errorf("am_michael's lines %v were not made in his accent", store.Held(michael.ID()))
	}
	if slices.ContainsFunc(store.Entries(), func(entry string) bool { return entry == "delete bf_emma" || entry == "delete am_michael" }) {
		t.Errorf("store saw %v, want no delete with nothing stale", store.Entries())
	}
}

// FR-544: a cast loads the model at once without making a line and without waiting for the load.
func TestCastingLoadsTheModelAtOnceWithoutWaitingForIt(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	maker.HoldLoad = make(chan struct{})
	emma := voiceNamed(t, "bf_emma")
	store.Hold(emma.ID(), keys("bə", "bɪ", "bi")...)
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	cast := make(chan struct{})
	go func() {
		defer close(cast)
		if _, err := service.Cast(emma); err != nil {
			t.Errorf("Cast: %v", err)
		}
	}()
	makingtest.Await(t, cast, "the cast returning while the model loads")
	makingtest.Await(t, maker.Loaded, "the model loading")
	close(maker.HoldLoad)
	service.Wait()

	if maker.Made() != 0 || maker.Loads() != 1 {
		t.Errorf("made %d lines and loaded %d times; want the model loaded once with no line made",
			maker.Made(), maker.Loads())
	}
}

// FR-544: casting only recorded voices never loads the model.
func TestCastingARecordedVoiceLoadsNothing(t *testing.T) {
	maker := makingtest.NewMaker()
	service := makingWith(t, "di", makingtest.Files{}, maker, makingtest.NewStore())

	service.CastRecorded()

	if maker.Loads() != 0 {
		t.Errorf("loaded %d times for a recorded voice, want none", maker.Loads())
	}
}

// FR-514: while making is under way, the voice answers a cue with its current made lines alone; a
// cue with none yet is silent.
func TestWhileMakingTheVoiceSpeaksOnlyWhatIsMade(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	maker.BlockOn = 1
	emma := voiceNamed(t, "bf_emma")
	store.Hold(emma.ID(), makingtest.Key("bə"))
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	makingtest.Await(t, maker.Started, "making starting")
	defer service.Stop()

	if takes, ok := source.Lookup("Docked"); !ok || !slices.Equal(takes, []string{store.Path(emma, makingtest.Key("bə"))}) {
		t.Errorf("Docked answered %v, %v; want the one line already made", takes, ok)
	}
	if _, ok := source.Lookup("Undocked"); ok {
		t.Error("a cue with nothing made answered")
	}
	if !service.Progress().Making {
		t.Error("progress does not say making is under way")
	}
}

// FR-516 and FR-527: casting another voice stops making, keeping every line written so far by either
// voice. The line being made when it stopped is no failure; the voice cast before answers nothing.
func TestCastingAnotherVoiceStopsMakingKeepingWhatWasWritten(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	maker.BlockOn = 2
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	before, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	makingtest.Await(t, maker.Started, "making starting")
	if _, err := service.Cast(michael); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()

	want := []string{"write bf_emma", "write am_michael", "write am_michael", "write am_michael"}
	if got := store.Entries(); !slices.Equal(got, want) {
		t.Errorf("store saw %v, want %v", got, want)
	}
	if got := service.Progress(); got.Voice != "am_michael" || len(got.Failed) != 0 || got.Current != 3 {
		t.Errorf("progress = %+v, want am_michael's confirmation made with no failure", got)
	}
	if len(store.Held(emma.ID())) != 1 {
		t.Errorf("bf_emma's lines %v, want the one written kept", store.Held(emma.ID()))
	}
	if _, ok := before.Lookup("Docked"); ok {
		t.Error("the voice cast before still answered")
	}
}

// FR-518: a line that cannot be made is reported with its cue and the reason; making goes on. A line
// of no sounds is one; the model failing is another.
func TestALineThatCannotBeMadeIsReportedAndMakingGoesOn(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	maker.FailOn = 1
	emma := voiceNamed(t, "bf_emma")
	service := makingWith(t, "", makingtest.Files{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()
	cueMaker(t, source).MakeNext("Undocked")
	service.Wait()

	got := service.Progress()
	if len(got.Failed) != 2 {
		t.Fatalf("failed = %+v, want two lines", got.Failed)
	}
	first, second := got.Failed[0], got.Failed[1]
	if first.Cue != "Docked" || first.Index != 0 || !errors.Is(first.Reason, makingtest.ErrModel) {
		t.Errorf("first failure = %+v, want Docked's first line refused by the model", first)
	}
	if second.Cue != "Undocked" || second.Index != 2 || !errors.Is(second.Reason, speech.ErrNoSounds) {
		t.Errorf("second failure = %+v, want Undocked's third line of no sounds", second)
	}
	if got.Current != 4 || got.Making || got.Stopped != nil || len(store.Held(emma.ID())) != 4 {
		t.Errorf("progress = %+v with %d on disk; want the other 4 made", got, len(store.Held(emma.ID())))
	}
}

// FR-520: a made line that cannot be written stops making, saying why; no line is on its way after.
func TestAWriteFailureStopsMakingAndSaysWhy(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	store.FailWrite = 2
	emma := voiceNamed(t, "bf_emma")
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()

	got := service.Progress()
	if !errors.Is(got.Stopped, makingtest.ErrDisk) || got.Making || got.Current != 1 || maker.Made() != 2 {
		t.Errorf("progress = %+v after %d made; want stopped by the disk after the second", got, maker.Made())
	}
	if cueMaker(t, source).MakeNext("Undocked") {
		t.Error("a line was on its way after making stopped")
	}
}

// FR-519: a voice whose files cannot be read is refused with the reason, changing nothing: the
// voice already cast keeps its lines and goes on answering.
func TestAVoiceWhoseFilesCannotBeReadIsRefusedChangingNothing(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	files := makingtest.Files{Refused: map[string]error{"am_michael": errFiles}}
	service := makingWith(t, "di", files, maker, store)
	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()
	before := store.Entries()

	refused, err := service.Cast(michael)

	if !errors.Is(err, errFiles) || refused != nil {
		t.Errorf("Cast(am_michael) = %v, %v; want refused naming the file", refused, err)
	}
	if !slices.Equal(store.Entries(), before) || service.Progress().Voice != "bf_emma" {
		t.Errorf("the refusal changed something: store %v, progress %+v", store.Entries(), service.Progress())
	}
	if _, ok := source.Lookup("Docked"); !ok {
		t.Error("the voice already cast stopped answering")
	}
}

// FR-527 and FR-530: a cast deletes that voice's lines no longer current and no other voice's; where
// one cannot be deleted the cast says so and completes.
func TestCastingDeletesOnlyTheVoicesStaleLinesSayingWhereItCannot(t *testing.T) {
	emma, michael := voiceNamed(t, "bf_emma"), voiceNamed(t, "am_michael")
	for _, deleteErr := range []error{nil, errDelete} {
		store := makingtest.NewStore()
		store.Hold(michael.ID(), "an old line")
		store.Hold(emma.ID(), "an old line", makingtest.Key("bə"))
		store.DeleteErr = deleteErr
		service := makingWith(t, "di", makingtest.Files{}, makingtest.NewMaker(), store)

		if _, err := service.Cast(emma); err != nil {
			t.Fatalf("Cast: %v", err)
		}
		service.Wait()

		got := service.Progress()
		if !errors.Is(got.NotDeleted, deleteErr) || got.Current != 3 {
			t.Errorf("progress = %+v, want the confirmation made with NotDeleted %v", got, deleteErr)
		}
		if !slices.Equal(store.Held(michael.ID()), []string{"an old line"}) {
			t.Errorf("am_michael's lines %v, want them untouched", store.Held(michael.ID()))
		}
		if old := slices.Contains(store.Held(emma.ID()), "an old line"); old != (deleteErr != nil) {
			t.Errorf("bf_emma's lines %v with delete failing %v", store.Held(emma.ID()), deleteErr)
		}
	}
}

// FR-516 and FR-527: casting a recorded voice stops making and keeps every made line; the machine
// voice cast before answers nothing from then on.
func TestCastingARecordedVoiceStopsMakingKeepingEveryLine(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	maker.BlockOn = 2
	emma := voiceNamed(t, "bf_emma")
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	makingtest.Await(t, maker.Started, "making starting")
	service.CastRecorded()

	if got, want := store.Entries(), []string{"write bf_emma"}; !slices.Equal(got, want) {
		t.Errorf("store saw %v, want %v", got, want)
	}
	if got := service.Progress(); got.Voice != "" || got.Making || got.NotDeleted != nil {
		t.Errorf("progress = %+v, want no machine voice, not making, nothing to delete", got)
	}
	if _, ok := source.Lookup("Docked"); ok {
		t.Error("the machine voice cast before still answered")
	}
}

// Two lines with the same sounds share one made line: it is made once and both cues answer with it.
func TestALineSharingSoundsWithOneMadeIsMadeOnce(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	emma := voiceNamed(t, "bf_emma")
	service := makingWith(t, "bə", makingtest.Files{}, maker, store)

	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()
	cueMaker(t, source).MakeNext("Undocked")
	service.Wait()

	shared := store.Path(emma, makingtest.Key("bə"))
	docked, _ := source.Lookup("Docked")
	undocked, _ := source.Lookup("Undocked")
	if maker.Made() != 5 || service.Progress().Current != 6 || !slices.Contains(docked, shared) || !slices.Contains(undocked, shared) {
		t.Errorf("made %d, current %d, Docked %v, Undocked %v; want one made line shared by both",
			maker.Made(), service.Progress().Current, docked, undocked)
	}
}
