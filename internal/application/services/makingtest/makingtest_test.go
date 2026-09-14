package makingtest_test

// The fakes answer as their documents say, since every suite that makes lines over them relies on it.

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
)

// errRefused is the reason a test refuses a voice with.
var errRefused = errors.New("bf_emma.bin is missing")

// emma is the voice every fake below is asked about.
func emma(t *testing.T) machinevoice.Voice {
	t.Helper()
	voice, err := machinevoice.Parse("bf_emma")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return voice
}

func TestMaterialIsKeyedByMadeFrom(t *testing.T) {
	t.Parallel()
	if got := makingtest.Material(); got.Files != makingtest.MadeFrom {
		t.Errorf("material files = %+v, want %+v", got.Files, makingtest.MadeFrom)
	}
	if got, want := makingtest.Key("bə"), making.Key("bə", makingtest.MadeFrom); got != want {
		t.Errorf("Key = %q, want %q", got, want)
	}
}

func TestFilesAnswerTheMaterialUnlessTheVoiceIsRefused(t *testing.T) {
	t.Parallel()
	files := makingtest.Files{Material: makingtest.Material()}
	if got, err := files.Open(emma(t)); err != nil || got.Files != makingtest.MadeFrom {
		t.Errorf("Open = %+v, %v; want the material", got, err)
	}
	files.Refused = map[string]error{"bf_emma": errRefused}
	if _, err := files.Open(emma(t)); !errors.Is(err, errRefused) {
		t.Errorf("Open = %v, want the refusal", err)
	}
}

func TestTheMakerCountsFailsBlocksAndCloses(t *testing.T) {
	t.Parallel()
	maker := makingtest.NewMaker()
	maker.FailOn, maker.BlockOn = 2, 3
	ctx, cancel := context.WithCancel(context.Background())

	if samples, err := maker.Make(ctx, []int64{1, 2}, nil); err != nil || !slices.Equal(samples, []float32{2}) {
		t.Errorf("first Make = %v, %v; want one sample counting two numbers", samples, err)
	}
	if _, err := maker.Make(ctx, nil, nil); !errors.Is(err, makingtest.ErrModel) {
		t.Errorf("second Make = %v, want the model failing", err)
	}
	go func() {
		<-maker.Started
		cancel()
	}()
	if _, err := maker.Make(ctx, nil, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("third Make = %v, want it to wait for its context to end", err)
	}
	maker.Close()
	if maker.Made() != 3 || maker.Closed() != 1 {
		t.Errorf("made %d, closed %d; want 3 and 1", maker.Made(), maker.Closed())
	}
}

func TestTheMakerPausesUntilResumedOrStopped(t *testing.T) {
	t.Parallel()
	resumed := makingtest.NewMaker()
	resumed.PauseOn = 1
	go func() {
		<-resumed.Started
		close(resumed.Resume)
	}()
	if samples, err := resumed.Make(context.Background(), []int64{1}, nil); err != nil || !slices.Equal(samples, []float32{1}) {
		t.Errorf("resumed Make = %v, %v; want its line", samples, err)
	}

	stopped := makingtest.NewMaker()
	stopped.PauseOn = 1
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stopped.Started
		cancel()
	}()
	if _, err := stopped.Make(ctx, nil, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("stopped Make = %v, want it to end with its context", err)
	}
}

func TestTheStoreKeepsLogsAndFailsWhenTold(t *testing.T) {
	t.Parallel()
	voice := emma(t)
	store := makingtest.NewStore()
	store.FailWrite = 2
	store.Hold("am_michael", "old")

	if err := store.Write(voice, "a", nil); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	if err := store.Write(voice, "b", nil); !errors.Is(err, makingtest.ErrDisk) {
		t.Errorf("second Write = %v, want the disk failing", err)
	}
	if got := store.Keys(voice); !slices.Equal(got, []string{"a"}) {
		t.Errorf("Keys = %v, want the one line written", got)
	}
	if got := store.Path(voice, "a"); got != "bf_emma/a.flac" || got != makingtest.PathOf("bf_emma", "a") {
		t.Errorf("Path = %q, want bf_emma/a.flac as PathOf gives it", got)
	}
	if err := store.DeleteAllBut(voice); err != nil || len(store.Held("am_michael")) != 0 {
		t.Errorf("DeleteAllBut = %v leaving %v; want am_michael's line gone", err, store.Held("am_michael"))
	}
	if err := store.DeleteAll(); err != nil || len(store.Held("bf_emma")) != 0 {
		t.Errorf("DeleteAll = %v leaving %v; want nothing left", err, store.Held("bf_emma"))
	}

	store.DeleteErr = errRefused
	if err := store.DeleteAllBut(voice); !errors.Is(err, errRefused) {
		t.Errorf("DeleteAllBut = %v, want the delete failing", err)
	}
	if err := store.DeleteAll(); !errors.Is(err, errRefused) {
		t.Errorf("DeleteAll = %v, want the delete failing", err)
	}
	want := []string{"write bf_emma", "keep bf_emma", "delete all", "keep bf_emma", "delete all"}
	if got := store.Entries(); !slices.Equal(got, want) {
		t.Errorf("Entries = %v, want %v", got, want)
	}
}
