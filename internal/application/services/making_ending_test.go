package services_test

// FR-556 with FR-518 and FR-548: a run and an audition write a line faded where the samples made are
// those its ending was found in; where they differ, the samples as made with a log line naming the
// voice, the cue and the line. The fade goes on the samples as made, before any pause. A fade that
// cannot be applied is a line that cannot be made.

import (
	"errors"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/ending"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/pause"
)

// endingFade is the fade the books below apply, in samples.
const endingFade = 2

// fadedAnswer is modelAnswer faded from its second sample over endingFade samples.
var fadedAnswer = []float32{1, 2, 1.5, 0}

// dockedEnding is bf_emma's ending for Docked's first line, "Docking complete.", found in modelAnswer
// at its second sample.
func dockedEnding() ending.Entry {
	return ending.Entry{Cue: "Docked", Index: 0, Sounds: "bə", Digest: pause.Digest(modelAnswer), Sample: 1}
}

// measured is the service with the books it was given and the key Docked's first line is made under.
type measured struct {
	service *services.MakingService
	key     string
}

// endedMaking builds the service over twoCues with books giving bf_emma the pauses and endings given, a
// maker answering modelAnswer and the log given.
func endedMaking(t *testing.T, pauses []pause.Entry, endings []ending.Entry, store *makingtest.Store, log *makingtest.Log) measured {
	t.Helper()
	pauseBook, err := pause.NewBook(pauseSilence, makingtest.MadeFrom.Model, map[string]pause.Voice{
		"bf_emma": {Style: makingtest.MadeFrom.Style, Entries: pauses},
	})
	if err != nil {
		t.Fatalf("pause.NewBook: %v", err)
	}
	endingBook, err := ending.NewBook(endingFade, makingtest.MadeFrom.Model, map[string]ending.Voice{
		"bf_emma": {Style: makingtest.MadeFrom.Style, Entries: endings},
	})
	if err != nil {
		t.Fatalf("ending.NewBook: %v", err)
	}
	voiced := twoCues(t, "di")
	maker := makingtest.NewMaker()
	maker.Answer = modelAnswer
	files := makingtest.Files{Material: makingtest.Material()}
	line := making.New(voiced, voiceNamed(t, "bf_emma"), makingtest.MadeFrom, pauseBook, endingBook, nil).ToMake()[0]
	return measured{
		service: services.NewMakingService(voiced, pauseBook, endingBook, "Docked", files, maker, store, log),
		key:     line.Key,
	}
}

// nothingLogged asserts that the run logged no line.
func nothingLogged(t *testing.T, log *makingtest.Log) {
	t.Helper()
	if lines := log.Lines(); len(lines) != 0 {
		t.Errorf("logged %q, want nothing", lines)
	}
}

// FR-556's acceptance in a run: the line's samples are those measured, so it is written faded from its
// ending's sample keeping its length; nothing is logged and a line with no ending is written as made.
func TestARunWritesALineFadedWhereItsSamplesAreThoseMeasured(t *testing.T) {
	store, log := makingtest.NewStore(), &makingtest.Log{}
	m := endedMaking(t, nil, []ending.Entry{dockedEnding()}, store, log)
	castAndWait(t, m.service)

	wrote(t, store, m.key, fadedAnswer)
	wrote(t, store, makingtest.Key("bɪ"), modelAnswer)
	nothingLogged(t, log)
}

// FR-556's acceptance in a run: samples other than those measured are written as made, logging the
// voice, the cue, the line and both digests under FR-556.
func TestARunWritesSamplesOtherThanThoseMeasuredWithoutTheFadeLoggingTheLine(t *testing.T) {
	store, log := makingtest.NewStore(), &makingtest.Log{}
	entry := dockedEnding()
	entry.Digest = otherDigest
	m := endedMaking(t, nil, []ending.Entry{entry}, store, log)
	castAndWait(t, m.service)

	wrote(t, store, m.key, modelAnswer)
	loggedTheLine(t, log)
	if lines := log.Lines(); len(lines) == 1 && !strings.Contains(lines[0], "(FR-556)") {
		t.Errorf("logged %q, which does not cite FR-556", lines[0])
	}
}

// FR-555 with FR-556: a line whose ending gives no fade is written as made under the key it had before
// endings, logging nothing.
func TestALineWithNoFadeIsWrittenAsMadeLoggingNothing(t *testing.T) {
	store, log := makingtest.NewStore(), &makingtest.Log{}
	entry := dockedEnding()
	entry.Sample = 0
	m := endedMaking(t, nil, []ending.Entry{entry}, store, log)
	castAndWait(t, m.service)

	if m.key != makingtest.Key("bə") {
		t.Errorf("the line is keyed %s, want %s", m.key, makingtest.Key("bə"))
	}
	wrote(t, store, m.key, modelAnswer)
	nothingLogged(t, log)
}

// FR-518 with FR-556: a fade beyond the samples made cannot be applied, so its line cannot be made and
// is reported with why; making goes on with the other lines.
func TestAFadeThatCannotBeAppliedIsALineThatCannotBeMade(t *testing.T) {
	store, log := makingtest.NewStore(), &makingtest.Log{}
	entry := dockedEnding()
	entry.Sample = len(modelAnswer) + 1
	m := endedMaking(t, nil, []ending.Entry{entry}, store, log)
	castAndWait(t, m.service)

	got := m.service.Progress()
	if len(got.Failed) != 1 || got.Failed[0].Cue != "Docked" || got.Failed[0].Index != 0 || !errors.Is(got.Failed[0].Reason, ending.ErrOutOfRange) {
		t.Errorf("failed = %+v, want Docked's first line out of range", got.Failed)
	}
	if got.Current != 2 || len(store.Held("bf_emma")) != 2 {
		t.Errorf("current %d with %v on disk; want the other 2 lines made", got.Current, store.Held("bf_emma"))
	}
}

// FR-556: the fade goes on the samples as made, then the pause is inserted, each checked against the
// digest of the samples as made; nothing is logged.
func TestTheFadeGoesOnTheSamplesAsMadeBeforeThePauseIsInserted(t *testing.T) {
	store, log := makingtest.NewStore(), &makingtest.Log{}
	m := endedMaking(t, []pause.Entry{dockedPause()}, []ending.Entry{dockedEnding()}, store, log)
	castAndWait(t, m.service)

	wrote(t, store, m.key, []float32{1, 2, 0, 0, 0, 1.5, 0})
	nothingLogged(t, log)
}

// FR-556 in an audition: the line drawn is written faded where its samples are those measured; where
// they differ it is written as made, logging the line.
func TestAnAuditionWritesTheLineFadedOrAsMadeLoggingWhereItsSamplesDiffer(t *testing.T) {
	for name, each := range map[string]struct {
		digest string
		want   []float32
		logged bool
	}{
		"measured":      {digest: pause.Digest(modelAnswer), want: fadedAnswer},
		"other samples": {digest: otherDigest, want: modelAnswer, logged: true},
	} {
		t.Run(name, func(t *testing.T) {
			store, log := makingtest.NewStore(), &makingtest.Log{}
			entry := dockedEnding()
			entry.Digest = each.digest
			m := endedMaking(t, nil, []ending.Entry{entry}, store, log)

			path, err := m.service.Audition(voiceNamed(t, "bf_emma"), "Docked", cue.HeardAll, lineAt(0))

			if err != nil || path != makingtest.PathOf("bf_emma", m.key) {
				t.Fatalf("Audition = %q, %v; want the faded line's path", path, err)
			}
			wrote(t, store, m.key, each.want)
			if each.logged {
				loggedTheLine(t, log)
			} else {
				nothingLogged(t, log)
			}
		})
	}
}
