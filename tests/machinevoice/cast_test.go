//go:build benchmarks && windows

package machinevoice

import (
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/domain/making"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
	"github.com/oernster/bridge-talk/internal/infrastructure/speechmodel"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

const (
	// castLimit is how soon a cast machine voice's confirmation reaches the player (NFR-P-205).
	castLimit = 5 * time.Second
	// madeOnCallLimit is how soon a cue's line is written once the cue fires with none (FR-514).
	madeOnCallLimit = 2 * time.Second
	// handOver is the poll interval through which the facade hands a written confirmation to the
	// player (FR-615), counted in full since where in it the line lands cannot be known.
	handOver = 250 * time.Millisecond
	// lookInterval is how often the test looks for a line, fine against the limits it measures.
	lookInterval = 5 * time.Millisecond
)

// shipped loads the shipped cue table and the script voiced with its saved speech sounds.
func shipped(t *testing.T) (cue.Table, script.Voiced) {
	t.Helper()
	table, err := config.LoadCueTable("")
	if err != nil {
		t.Fatalf("loading the shipped cue table: %v", err)
	}
	voiced, err := config.LoadVoicedScript(table)
	if err != nil {
		t.Fatalf("script.toml with sounds.toml: %v", err)
	}
	return table, voiced
}

// TestAMachineVoiceIsCastWithinFiveSeconds casts the measured voice over an empty store with the real
// model, then holds how soon its confirmation is current, with the hand-over added, to NFR-P-205. It
// then asks for the last cue in the making order, which is certainly unmade; it holds how soon that
// cue's first line is written to FR-514.
func TestAMachineVoiceIsCastWithinFiveSeconds(t *testing.T) {
	dir := modelfilestest.Require(t)
	table, voiced := shipped(t)
	voice, err := machinevoice.Parse(measuredVoice)
	if err != nil {
		t.Fatalf("Parse(%q): %v", measuredVoice, err)
	}
	order := making.Order(table)
	confirmation, last := order[0], order[len(order)-1]

	maker := speechmodel.New(dir)
	defer maker.Close()
	service := services.NewMakingService(voiced, order, voicefiles.New(dir), maker, madelines.New(t.TempDir()))
	defer service.Stop()

	started := time.Now()
	source, err := service.Cast(voice)
	if err != nil {
		t.Fatalf("Cast(%s): %v", measuredVoice, err)
	}
	confirmed := until(source, confirmation, started, castLimit) + handOver
	t.Logf("%s's confirmation reached the player %v after the cast", measuredVoice, confirmed.Round(time.Millisecond))
	if confirmed > castLimit {
		t.Errorf("NFR-P-205: the confirmation took %v, over the %v limit", confirmed.Round(time.Millisecond), castLimit)
	}

	onCall, ok := source.(ports.CueMaker)
	if !ok {
		t.Fatalf("the audio source %T makes no line on call", source)
	}
	fired := time.Now()
	if !onCall.MakeNext(last) {
		t.Fatalf("no line was on its way for %s", last)
	}
	made := until(source, last, fired, madeOnCallLimit)
	t.Logf("%s's first line was written %v after it was asked for", last, made.Round(time.Millisecond))
	if made > madeOnCallLimit {
		t.Errorf("FR-514: the line took %v, over the %v limit", made.Round(time.Millisecond), madeOnCallLimit)
	}
}

// until answers how long after from the cue had a current line, looking until limit has passed.
func until(source ports.AudioSource, id cue.ID, from time.Time, limit time.Duration) time.Duration {
	for {
		if _, ok := source.Lookup(id); ok || time.Since(from) > limit {
			return time.Since(from)
		}
		time.Sleep(lookInterval)
	}
}
