package services_test

// FR-553 with FR-518 and FR-548: a run and an audition write a line with its pause where the samples
// made are those the pause was found in; where they differ, the samples as made with a log line naming
// the voice, the cue and the line. A pause that cannot be inserted is a line that cannot be made.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/pause"
)

// pauseSilence is the silence the books below insert.
const pauseSilence = 3

// otherDigest is a digest no samples below are made with.
const otherDigest = "0123456789abcdef"

// modelAnswer is the samples the maker answers every line with.
var modelAnswer = []float32{1, 2, 3, 4}

// dockedPause is bf_emma's pause for Docked's first line, "Docking complete.", found in modelAnswer
// before its third sample.
func dockedPause() pause.Entry {
	return pause.Entry{Cue: "Docked", Index: 0, Sounds: "bə", Digest: pause.Digest(modelAnswer), Sample: 2}
}

// pausedMaking builds the service over twoCues with a book giving bf_emma the pause given, a maker
// answering modelAnswer and the log given.
func pausedMaking(t *testing.T, entry pause.Entry, store *makingtest.Store, log *makingtest.Log) *services.MakingService {
	t.Helper()
	book, err := pause.NewBook(pauseSilence, makingtest.MadeFrom.Model, map[string]pause.Voice{
		"bf_emma": {Style: makingtest.MadeFrom.Style, Entries: []pause.Entry{entry}},
	})
	if err != nil {
		t.Fatalf("NewBook: %v", err)
	}
	maker := makingtest.NewMaker()
	maker.Answer = modelAnswer
	files := makingtest.Files{Material: makingtest.Material()}
	return services.NewMakingService(twoCues(t, "di"), book, ending.Book{}, "Docked", files, maker, store, log)
}

// castAndWait casts bf_emma, waiting for the confirmation's lines to be made.
func castAndWait(t *testing.T, service *services.MakingService) {
	t.Helper()
	if _, err := service.Cast(voiceNamed(t, "bf_emma")); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	service.Wait()
}

// wrote asserts the samples written under a line's key for bf_emma.
func wrote(t *testing.T, store *makingtest.Store, key string, want []float32) {
	t.Helper()
	if got, ok := store.Written("bf_emma", key); !ok || !slices.Equal(got, want) {
		t.Errorf("wrote %v, %v; want %v", got, ok, want)
	}
}

// loggedTheLine asserts one log line naming the voice, the cue, the line and its text with both digests.
func loggedTheLine(t *testing.T, log *makingtest.Log) {
	t.Helper()
	lines := log.Lines()
	if len(lines) != 1 {
		t.Fatalf("logged %q, want one line", lines)
	}
	for _, fragment := range []string{`bf_emma "Docked" line 1 "Docking complete."`, otherDigest, pause.Digest(modelAnswer)} {
		if !strings.Contains(lines[0], fragment) {
			t.Errorf("logged %q, which does not name %q", lines[0], fragment)
		}
	}
}

// FR-553's acceptance in a run: the line's samples are those measured, so it is written with the
// silence inserted before the pause's sample and nothing is logged; a line with no pause is written
// as made.
func TestARunWritesALineWithItsPauseWhereItsSamplesAreThoseMeasured(t *testing.T) {
	store, log := makingtest.NewStore(), &makingtest.Log{}
	castAndWait(t, pausedMaking(t, dockedPause(), store, log))

	wrote(t, store, making.PausedKey("bə", makingtest.MadeFrom, dockedPause(), pauseSilence), []float32{1, 2, 0, 0, 0, 3, 4})
	wrote(t, store, makingtest.Key("bɪ"), modelAnswer)
	if lines := log.Lines(); len(lines) != 0 {
		t.Errorf("logged %q, want nothing", lines)
	}
}

// FR-553's acceptance in a run: samples other than those measured are written as made, logging the
// voice, the cue and the line.
func TestARunWritesSamplesOtherThanThoseMeasuredAsMadeLoggingTheLine(t *testing.T) {
	store, log := makingtest.NewStore(), &makingtest.Log{}
	entry := dockedPause()
	entry.Digest = otherDigest
	castAndWait(t, pausedMaking(t, entry, store, log))

	wrote(t, store, making.PausedKey("bə", makingtest.MadeFrom, entry, pauseSilence), modelAnswer)
	loggedTheLine(t, log)
}

// FR-552 with FR-553: a line whose break is doubtful is written as made, logging nothing, even where
// its samples are those measured.
func TestADoubtfulLineIsWrittenAsMadeLoggingNothing(t *testing.T) {
	store, log := makingtest.NewStore(), &makingtest.Log{}
	entry := dockedPause()
	entry.Doubtful = true
	castAndWait(t, pausedMaking(t, entry, store, log))

	wrote(t, store, makingtest.Key("bə"), modelAnswer)
	if lines := log.Lines(); len(lines) != 0 {
		t.Errorf("logged %q, want nothing", lines)
	}
}

// FR-518 with FR-553: a pause beyond the samples made cannot be inserted, so its line cannot be made
// and is reported with why; making goes on with the other lines.
func TestAPauseThatCannotBeInsertedIsALineThatCannotBeMade(t *testing.T) {
	store, log := makingtest.NewStore(), &makingtest.Log{}
	entry := dockedPause()
	entry.Sample = len(modelAnswer) + 1
	service := pausedMaking(t, entry, store, log)
	castAndWait(t, service)

	got := service.Progress()
	if len(got.Failed) != 1 || got.Failed[0].Cue != "Docked" || got.Failed[0].Index != 0 || !errors.Is(got.Failed[0].Reason, pause.ErrOutOfRange) {
		t.Errorf("failed = %+v, want Docked's first line out of range", got.Failed)
	}
	if got.Current != 2 || len(store.Held("bf_emma")) != 2 {
		t.Errorf("current %d with %v on disk; want the other 2 lines made", got.Current, store.Held("bf_emma"))
	}
}

// FR-553 in an audition: the line drawn is written with its pause where its samples are those measured;
// where they differ it is written as made, logging the line.
func TestAnAuditionWritesTheLineWithItsPauseOrAsMadeLoggingWhereItsSamplesDiffer(t *testing.T) {
	for name, each := range map[string]struct {
		digest string
		want   []float32
		logged bool
	}{
		"measured":      {digest: pause.Digest(modelAnswer), want: []float32{1, 2, 0, 0, 0, 3, 4}},
		"other samples": {digest: otherDigest, want: modelAnswer, logged: true},
	} {
		t.Run(name, func(t *testing.T) {
			store, log := makingtest.NewStore(), &makingtest.Log{}
			entry := dockedPause()
			entry.Digest = each.digest
			key := making.PausedKey("bə", makingtest.MadeFrom, entry, pauseSilence)

			path, err := pausedMaking(t, entry, store, log).Audition(voiceNamed(t, "bf_emma"), "Docked", lineAt(0))

			if err != nil || path != makingtest.PathOf("bf_emma", key) {
				t.Fatalf("Audition = %q, %v; want the paused line's path", path, err)
			}
			wrote(t, store, key, each.want)
			if each.logged {
				loggedTheLine(t, log)
			} else if lines := log.Lines(); len(lines) != 0 {
				t.Errorf("logged %q, want nothing", lines)
			}
		})
	}
}

// FR-548 with FR-553: an audition whose pause cannot be inserted answers why, keeping nothing.
func TestAnAuditionWhosePauseCannotBeInsertedAnswersWhyKeepingNothing(t *testing.T) {
	store := makingtest.NewStore()
	entry := dockedPause()
	entry.Sample = len(modelAnswer) + 1

	_, err := pausedMaking(t, entry, store, &makingtest.Log{}).Audition(voiceNamed(t, "bf_emma"), "Docked", lineAt(0))

	if !errors.Is(err, pause.ErrOutOfRange) {
		t.Errorf("Audition = %v, want ErrOutOfRange", err)
	}
	if held := store.Held("bf_emma"); len(held) != 0 {
		t.Errorf("kept %v, want nothing", held)
	}
}
