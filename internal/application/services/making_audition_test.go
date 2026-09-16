package services_test

// FR-546 to FR-548: auditioning a machine voice over fakes. The line drawn is made where it is not
// yet made, next after the line under way; it is kept and its path answered. What cannot be made
// answers why, keeping nothing.

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/take"
)

// lineAt draws the line at a fixed place in a group, so a test knows which line an audition makes.
type lineAt int

func (l lineAt) Intn(n int) int { return int(l) % n }

// awaitWaiting waits until an audition waits for the model, failing past makingtest.AwaitLimit.
func awaitWaiting(t *testing.T, service *services.MakingService) {
	t.Helper()
	deadline := time.Now().Add(makingtest.AwaitLimit)
	for service.AuditionsWaiting() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the audition never waited for the model")
		}
		time.Sleep(time.Millisecond)
	}
}

// FR-546: the groups offered are the script's, each counting its lines.
func TestTheGroupsAuditionedAreTheScriptsCountingTheirLines(t *testing.T) {
	service := makingWith(t, "di", makingtest.Files{}, makingtest.NewMaker(), makingtest.NewStore())

	want := []script.Group{{Key: "Docked", Lines: 3}, {Key: "Undocked", Lines: 3}}
	if got := service.AuditionGroups(cue.HeardAll); !slices.Equal(got, want) {
		t.Errorf("groups = %v, want %v", got, want)
	}
}

// FR-546: a voice not cast is auditioned. The line drawn is made in its accent, kept and answered;
// no other line is made.
func TestAnAuditionMakesTheLineDrawnKeepsItAndAnswersWhereItPlays(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	path, err := service.Audition(voiceNamed(t, "am_michael"), "Undocked", cue.HeardAll, lineAt(1))

	if err != nil {
		t.Fatalf("Audition: %v", err)
	}
	key := makingtest.Key("dæɪ")
	if path != makingtest.PathOf("am_michael", key) {
		t.Errorf("path = %q, want the second line's made line", path)
	}
	if got := store.Held("am_michael"); maker.Made() != 1 || !slices.Equal(got, []string{key}) {
		t.Errorf("made %d leaving %v; want the line drawn alone", maker.Made(), got)
	}
}

// FR-546: a line already made is answered without being made again.
func TestAnAuditionOfALineAlreadyMadeMakesNothing(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	key := makingtest.Key("dæ")
	store.Hold("am_michael", key)
	service := makingWith(t, "di", makingtest.Files{}, maker, store)

	path, err := service.Audition(voiceNamed(t, "am_michael"), "Undocked", cue.HeardAll, lineAt(0))

	if err != nil || path != makingtest.PathOf("am_michael", key) || maker.Made() != 0 {
		t.Errorf("answered %q, %v after making %d; want the made line with nothing made", path, err, maker.Made())
	}
}

// FR-546 with FR-527: a line auditioned for the cast voice counts as made, so the cast plays it and
// never makes it again.
func TestALineAuditionedForTheCastVoiceCountsAsMade(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	emma := voiceNamed(t, "bf_emma")
	service := makingWith(t, "di", makingtest.Files{}, maker, store)
	source, err := service.Cast(emma)
	if err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()

	if _, err := service.Audition(emma, "Undocked", cue.HeardAll, lineAt(1)); err != nil {
		t.Fatalf("Audition: %v", err)
	}

	want := []take.Take{take.Of(makingtest.PathOf("bf_emma", makingtest.Key("dɪ")))}
	if takes, ok := source.Lookup("Undocked"); !ok || !reflect.DeepEqual(takes, want) {
		t.Errorf("Undocked answered %v, %v; want the auditioned line", takes, ok)
	}
	if got := service.Progress().Current; got != 4 {
		t.Errorf("%d lines current, want the confirmation's 3 and the auditioned one", got)
	}
}

// FR-546: an audition waiting for the model is made next after the line under way, ahead of the rest
// of the run.
func TestAnAuditionIsMadeNextAfterTheLineUnderWay(t *testing.T) {
	store, maker := makingtest.NewStore(), makingtest.NewMaker()
	maker.PauseOn = 1
	service := makingWith(t, "di", makingtest.Files{}, maker, store)
	if _, err := service.Cast(voiceNamed(t, "bf_emma")); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	makingtest.Await(t, maker.Started, "the cast's first line starting")

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = service.Audition(voiceNamed(t, "am_michael"), "Undocked", cue.HeardAll, lineAt(0))
	}()
	awaitWaiting(t, service)
	close(maker.Resume)
	makingtest.Await(t, done, "the audition finishing")
	service.Wait()

	want := []string{"write bf_emma", "write am_michael", "write bf_emma", "write bf_emma"}
	if got := store.Entries(); !slices.Equal(got, want) {
		t.Errorf("written %v, want the audition's line straight after the line under way: %v", got, want)
	}
}

// FR-548: files that cannot be read, a line the model refuses and a line that cannot be written each
// answer why, keeping nothing.
func TestAnAuditionThatCannotBeMadeAnswersWhyKeepingNothing(t *testing.T) {
	for name, each := range map[string]struct {
		files             makingtest.Files
		failOn, failWrite int
		want              error
	}{
		"files refused": {files: makingtest.Files{Refused: map[string]error{"am_michael": errFiles}}, want: errFiles},
		"model refuses": {failOn: 1, want: makingtest.ErrModel},
		"write fails":   {failWrite: 1, want: makingtest.ErrDisk},
	} {
		t.Run(name, func(t *testing.T) {
			store, maker := makingtest.NewStore(), makingtest.NewMaker()
			maker.FailOn, store.FailWrite = each.failOn, each.failWrite
			service := makingWith(t, "di", each.files, maker, store)

			_, err := service.Audition(voiceNamed(t, "am_michael"), "Undocked", cue.HeardAll, lineAt(0))

			if !errors.Is(err, each.want) {
				t.Errorf("Audition = %v, want %v", err, each.want)
			}
			if held := store.Held("am_michael"); len(held) != 0 {
				t.Errorf("kept %v, want nothing", held)
			}
		})
	}
}

// FR-546: a group the script holds no lines for is refused before anything is made.
func TestAnAuditionOfAGroupTheScriptLacksIsRefused(t *testing.T) {
	maker := makingtest.NewMaker()
	service := makingWith(t, "di", makingtest.Files{}, maker, makingtest.NewStore())

	_, err := service.Audition(voiceNamed(t, "am_michael"), "ShieldState", cue.HeardAll, lineAt(0))

	if !errors.Is(err, services.ErrNothingToAudition) || maker.Made() != 0 {
		t.Errorf("Audition = %v after making %d, want ErrNothingToAudition with nothing made", err, maker.Made())
	}
}

// FR-745 and FR-747: a machine voice is auditioned on the lines of its moments switched on alone. A
// group whose moments are all off is counted as nothing, marked switched off and refused before
// anything is made; the other groups are as they were.
func TestAMachineAuditionDrawsOnlyOnMomentsSwitchedOn(t *testing.T) {
	maker := makingtest.NewMaker()
	service := makingWith(t, "di", makingtest.Files{}, maker, makingtest.NewStore())
	heard := func(id cue.ID) bool { return id != "Undocked" }

	want := []script.Group{{Key: "Docked", Lines: 3}, {Key: "Undocked", SwitchedOff: true}}
	if got := service.AuditionGroups(heard); !slices.Equal(got, want) {
		t.Errorf("groups = %v, want %v", got, want)
	}
	_, err := service.Audition(voiceNamed(t, "am_michael"), "Undocked", heard, lineAt(0))
	if !errors.Is(err, services.ErrNothingToAudition) || maker.Made() != 0 {
		t.Errorf("Audition = %v after making %d, want ErrNothingToAudition with nothing made", err, maker.Made())
	}
	if _, err := service.Audition(voiceNamed(t, "am_michael"), "Docked", heard, lineAt(0)); err != nil {
		t.Errorf("a group switched on was refused: %v", err)
	}
}
