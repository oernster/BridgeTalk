package main

// Shared fixtures for the facade's tests: a library on disk, a cue table over it
// and an assembled session. Everything here is built from a temporary directory, so
// no test reads the machine's real voices or writes to the user's own settings.

import (
	"math/rand"
	"path/filepath"
	"sync"
	"testing"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/application/services/makingtest"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/event"
	"github.com/oernster/bridge-talk/internal/domain/script"
	"github.com/oernster/bridge-talk/internal/domain/script/scripttest"
	"github.com/oernster/bridge-talk/internal/infrastructure/audio/audiotest"
	"github.com/oernster/bridge-talk/internal/infrastructure/library"
)

// fixtureSeed fixes the chooser so a test that plays a clip gets the same one twice.
const fixtureSeed = 1

// journalName is a journal file named the way the game names one.
const journalName = "Journal.2026-08-26T090000.01.log"

// writeJournal puts an empty journal in a directory, which is all a journal source needs
// to be built over.
func writeJournal(t *testing.T, dir string) {
	t.Helper()
	audiotest.WriteFile(t, filepath.Join(dir, journalName), nil)
}

// libraryRootFixture builds a library root holding one voice that covers both fixture
// cues, one that covers a single cue and one directory that is no voice at all,
// which is every case the cast pane has to describe.
//
// Alpha uses the folder form and Beta the flat form, so both discovery rules are
// exercised by every test that reads the fixture. Bystander holds a file the scanner
// cannot match, which is how a directory ends up reported rather than offered.
func libraryRootFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	audiotest.WriteTake(t, filepath.Join(root, "Alpha", "ShieldState_ShieldsUp_false", "a.mp3"))
	audiotest.WriteTake(t, filepath.Join(root, "Alpha", "Docked", "b.mp3"))
	audiotest.WriteTake(t, filepath.Join(root, "Alpha", "Cast_Confirmed", "yes.mp3"))
	audiotest.WriteTake(t, filepath.Join(root, "Beta", "ShieldState.ShieldsUp.false.mp3"))
	audiotest.WriteTake(t, filepath.Join(root, "Bystander", "holiday snap.mp3"))
	return root
}

// fixtureTable is the cue table the fixtures resolve against: two cues Alpha has
// recorded, one nothing has material for and the acknowledgement played on a cast.
func fixtureTable(t *testing.T) cue.Table {
	t.Helper()
	built := make([]cue.Cue, 0, 4)
	for _, each := range []struct{ id, source, event, purpose string }{
		{"ShieldState.ShieldsUp.false", "journal", "ShieldState", "When the shields fail."},
		{"Docked", "journal", "Docked", "When the ship docks."},
		{"zz.silent", "journal", "NobodyRecordsThis", "When nothing ever happens."},
		{"Cast.Confirmed", "application", "cast", "When this voice is cast."},
	} {
		one, err := cue.New(cue.Definition{
			ID: each.id, Source: each.source, Event: each.event, Purpose: each.purpose,
		})
		if err != nil {
			t.Fatalf("building cue %q: %v", each.id, err)
		}
		built = append(built, one)
	}
	return cue.NewTable(built)
}

// fixtureSession assembles a session over the fixture voices with none cast yet. Each prepare
// runs over the library root before it is scanned, which is how a test adds to the fixture.
func fixtureSession(t *testing.T, player audioPlayer, prepare ...func(root string)) (*session, string) {
	t.Helper()
	root := libraryRootFixture(t)
	for _, each := range prepare {
		each(root)
	}
	table := fixtureTable(t)
	found, report, err := library.Scan(root, table)
	if err != nil {
		t.Fatalf("scanning the fixture library: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("found %d voices, want Alpha and Beta", len(found))
	}
	// Bystander holds audio but nothing named for a cue, so it is reported rather
	// than offered. Asserting it here keeps the fixture honest about the difference.
	if len(report.Empty) != 1 {
		t.Fatalf("report named %d empty directories, want Bystander alone", len(report.Empty))
	}
	current := &session{
		table:     table,
		available: found,
		chooser:   randomChooser{source: rand.New(rand.NewSource(fixtureSeed))},
		player:    player,
	}
	fixtureMaking(t, current, offeredFiles(nil), makingtest.NewStore())
	return current, root
}

// fixtureScript gives the fixture's docking cue and its confirmation three lines each, so a machine
// voice cast over the fixture has lines to make and a confirmation among them.
func fixtureScript(t *testing.T) script.Voiced {
	t.Helper()
	voiced, err := scripttest.Build(map[string]script.Saved{
		"Docked": {
			Lines:   []string{"Line one.", "Line two.", "Line three."},
			British: []string{"bə", "bɪ", "bi"}, American: []string{"æə", "æɪ", "æi"},
		},
		"Cast.Confirmed": {
			Lines:   []string{"Line four.", "Line five.", "Line six."},
			British: []string{"du", "dɪ", "di"}, American: []string{"dæ", "dæɪ", "dæi"},
		},
	})
	if err != nil {
		t.Fatalf("building the fixture script: %v", err)
	}
	return voiced
}

// offeredFiles answers every machine voice with the fakes' material, refusing those refused names.
func offeredFiles(refused map[string]error) makingtest.Files {
	return makingtest.Files{Material: makingtest.Material(), Refused: refused}
}

// fixtureMaking gives a session a making service speaking the fixture script over the files and the
// store given, with a maker of its own. Making stops when the test ends.
func fixtureMaking(t *testing.T, current *session, files ports.VoiceFiles, store *makingtest.Store) *makingtest.Maker {
	t.Helper()
	maker := makingtest.NewMaker()
	current.making = services.NewMakingService(fixtureScript(t), files, maker, store)
	current.maker = maker
	t.Cleanup(current.making.Stop)
	return maker
}

// fixtureWatch is a journal watch over nothing, with names a test can tell apart from the
// facade's other directories.
var fixtureWatch = journalWatch{directory: "journal-dir", statusPath: "status-file"}

// fixtureApp assembles a facade over the fixture session, with the emitter captured.
func fixtureApp(t *testing.T, prepare ...func(root string)) (*App, *fakePlayer, *recorder) {
	t.Helper()
	player := newFakePlayer()
	current, root := fixtureSession(t, player, prepare...)
	app := newApp(current, fixtureWatch, root, nil)
	app.reveal = func(string) error { return nil }
	log := newRecorder()
	app.emit = log.emit
	return app, player, log
}

// fakeSettings is a settings store held in memory, so a test can watch what the
// facade decided to remember without writing to the user's own configuration.
type fakeSettings struct {
	held    ports.Settings
	saves   int
	failure error
}

func (f *fakeSettings) Load() ports.Settings { return f.held }

func (f *fakeSettings) Save(chosen ports.Settings) error {
	f.saves++
	if f.failure != nil {
		return f.failure
	}
	f.held = chosen
	return nil
}

// fakeSource is an event source that answers with whatever a test planted, including
// a failure, which is how the poll loop's error path is reached without a disk.
type fakeSource struct {
	mu     sync.Mutex
	name   string
	events []event.Event
	err    error
	polls  int
}

func (f *fakeSource) Name() string { return f.name }

func (f *fakeSource) Poll() ([]event.Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.polls++
	if f.err != nil {
		return nil, f.err
	}
	out := f.events
	f.events = nil
	return out, nil
}

// polled reports how many times the source has been asked, safely from another
// goroutine, since the poll loop runs in one of its own.
func (f *fakeSource) polled() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.polls
}

// lastEmitted returns the most recent payload recorded under a name. It is the
// counterpart to await: await waits for an event the run loop will raise later,
// while this reads one the call under test already raised before returning.
func (r *recorder) lastEmitted(t *testing.T, name string) any {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := len(r.events) - 1; index >= 0; index-- {
		if r.events[index].name == name {
			return r.events[index].payload
		}
	}
	t.Fatalf("no %q event was emitted; got %v", name, r.emittedNames())
	return nil
}

// emittedNames lists what was announced, for a failure message.
func (r *recorder) emittedNames() []string {
	out := make([]string, 0, len(r.events))
	for _, each := range r.events {
		out = append(out, each.name)
	}
	return out
}

// countEmitted reports how many times a name was announced.
func (r *recorder) countEmitted(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := 0
	for _, each := range r.events {
		if each.name == name {
			seen++
		}
	}
	return seen
}
