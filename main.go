// Command bridge-talk watches Elite Dangerous and speaks.
//
// This file is the composition root. It and app.go are the only files permitted to
// wire concrete infrastructure to the application layer; everything else works
// against the ports.
package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/infrastructure/audio"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/journal"
	"github.com/oernster/bridge-talk/internal/infrastructure/library"
	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/setup"
	"github.com/oernster/bridge-talk/internal/infrastructure/speechmodel"
	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
	"github.com/oernster/bridge-talk/internal/product"
)

// pollInterval is how often both sources are asked for new events. The journal read
// is a seek and a read of only the appended bytes, usually none, so this is cheap
// enough to run beside the game.
const pollInterval = 250 * time.Millisecond

// appTitle names the application in the window title, the tray tooltip and the
// About dialog: the product name as a reader sees it.
const appTitle = product.Name

// systemClock is the real clock, injected so the domain never reads the wall clock.
type systemClock struct{}

// Now returns the current time.
func (systemClock) Now() time.Time { return time.Now() }

// randomChooser is the real source of randomness, injected for the same reason.
type randomChooser struct{ source *rand.Rand }

// Intn returns a value in [0, n).
func (r randomChooser) Intn(n int) int { return r.source.Intn(n) }

func main() {
	keepLog()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "bridge-talk: %v\n", err)
		os.Exit(1)
	}
}

// session holds everything that has to be rebuilt when the voice changes.
// audioPlayer is everything the session and the facade ask of the output device.
//
// It is declared here, where it is consumed, rather than in ports: the application
// layer needs only ports.AudioPlayer, while the facade also reads the volume and
// whether the device opened at all. Declaring it makes the device replaceable in a
// test, which is the only way the facade's behaviour can be exercised at all.
type audioPlayer interface {
	Play(clips []string, gap time.Duration) error
	PlayIfIdle(clips []string, gap time.Duration) (bool, error)
	Stop()
	Playing() bool
	Done() <-chan struct{}
	Silent() bool
	Stalls() (count int, worst time.Duration)
	Volume() float64
	SetVolume(level float64)
	Close() error
}

// trayIcon is everything the session and the facade ask of the notification area icon. It is
// declared here, where it is consumed, for the reason audioPlayer is: a test can then read what
// the icon was told, which the icon itself keeps to its own thread.
type trayIcon interface {
	Commands() <-chan taskbar.Command
	SetMuted(muted bool)
	SetActiveVoice(name, label string, machine bool)
	Stop()
}

type session struct {
	table     cue.Table
	available []library.Voice
	chooser   randomChooser
	player    audioPlayer
	tray      trayIcon
	reporter  ports.Reporter

	// making makes a cast machine voice's lines; maker is the model they are made with, released
	// when the application closes.
	making *services.MakingService
	maker  releaser
	// plugins are the loaded plugins and the thread they are called on, closed with everything
	// else when the application closes. A session built without them, as some tests build one,
	// simply has no plugin voices.
	plugins *plugin.Set
	// announced is what the page was last told about making, so it is told only of a change.
	announced makingKey
	// confirming is set while a cast machine voice's confirmation waits for its line (FR-521). The
	// cast sets it from the window's goroutine; the poll tick reads it from its own.
	confirming atomic.Bool
	// auditions holds the machine voice audition whose line is being made (FR-547).
	auditions auditionGate

	active    castVoice
	catalogue *library.Catalogue
	scheduler *services.Scheduler
	reactions *services.ReactionService

	// muted lives here rather than on the reaction service because the service only
	// exists once a voice is cast. With no voices installed there is nothing to
	// silence and the control still has to answer, so the session holds the answer
	// and hands it to each service it builds.
	muted bool

	// chatter holds which moments are switched off. It is built once and handed to each reaction
	// service and scheduler a cast builds, so casting another voice leaves every switch as it
	// stands (FR-630).
	chatter *services.ChatterService
}

// hasVoice reports whether a voice is cast.
//
// Nothing is cast until a voice is found. The window opens either way, so every path
// that would speak has to be able to do nothing instead of assuming a voice.
func (s *session) hasVoice() bool { return s.reactions != nil }

// useVoice casts a recorded voice: making stops, keeping every made line (FR-516, FR-527); then
// the voice speaks from its recordings.
func (s *session) useVoice(chosen library.Voice) {
	s.making.CastRecorded()
	s.speakWith(chosen, castVoice{Name: chosen.Name, Display: chosen.Display()})
}

// speakWith rebuilds the catalogue and the reaction path over the audio source of the voice cast.
//
// The picker's memory of what each cue last played goes with the old service, which
// is correct: those clip paths belong to the voice being left behind.
func (s *session) speakWith(source ports.AudioSource, cast castVoice) {
	s.player.Stop()
	s.confirming.Store(false)

	s.active = cast
	s.catalogue = catalogueOver(source, cast.Display, s.table, s.chooser)
	s.scheduler = services.NewScheduler(s.player, s.reporter, systemClock{})
	s.reactions = services.NewReactionService(
		s.table, s.catalogue, s.scheduler, s.chooser, s.reporter, systemClock{},
	)
	s.reactions.SetMuted(s.muted)
	// The switches outlive the cast, so the ones in force stay in force (FR-630). A session built
	// without them, as some tests build one, has every moment on.
	if s.chatter != nil {
		s.scheduler.SetSwitchboard(s.chatter)
		s.reactions.SetSwitchboard(s.chatter)
	}
	// A machine voice makes a cue's lines when the cue fires with none (FR-514); a recorded voice has
	// nothing more to make.
	if maker, ok := source.(ports.CueMaker); ok {
		s.reactions.SetCueMaker(maker)
	}
	if s.tray != nil {
		s.tray.SetActiveVoice(cast.Name, cast.Display, cast.Machine)
	}
}

// catalogueFor builds a catalogue over any voice, cast or not. The cast pane reports
// on voices the user has not chosen and the audition pane plays from them, so the
// catalogue cannot be tied to the active one.
func (s *session) catalogueFor(voice library.Voice) *library.Catalogue {
	return catalogueOf(voice, s.table, s.chooser)
}

// voiceNamed finds a voice by exact name among those found at startup.
func (s *session) voiceNamed(name string) (library.Voice, bool) {
	for _, candidate := range s.available {
		if candidate.Name == name {
			return candidate, true
		}
	}
	return library.Voice{}, false
}

// coverageOf reports what a voice that is not the cast one could serve: the cues it has
// recorded, the files it uses and the recordings present in its directory. All three are
// what the cast pane shows before the user commits to casting it (FR-215).
func (s *session) coverageOf(voice library.Voice) (int, int, int) {
	catalogue := s.catalogueFor(voice)
	covered, _ := catalogue.Coverage()
	used, present := voice.Files()
	return covered, used, present
}

// run wires everything together and hands the assembled facade to Wails.
func run() error {
	libraryRoot := flag.String("library", "", "the directory holding your recordings")
	journalDir := flag.String("journal", "", "journal directory (default: the game's saved-games directory)")
	voice := flag.String("voice", "", "voice to speak with (default: the first one found)")
	listVoices := flag.Bool("list", false, "list the voices found with their takes and cue coverage, then exit")
	showUnbound := flag.Bool("unbound", false, "list the cues the chosen voice cannot serve, then exit")
	noTray := flag.Bool("no-tray", false, "run without a notification-area icon")
	hidden := flag.Bool(
		strings.TrimPrefix(setup.HiddenFlag, "-"), false,
		"start in the notification area with no window, as the login entry does",
	)
	flag.Parse()
	// The reports are read in the terminal that asked for them (FR-703).
	if *listVoices || *showUnbound {
		reportToTerminal()
	}

	table, err := config.LoadCueTable("")
	if err != nil {
		return err
	}
	// Finding nothing is not a failure to start. A window that never appears reads as
	// a broken application rather than an unconfigured one, so the search result is
	// carried rather than returned: the root is kept whether or not it exists, so the
	// pane can name where it looked. An unreadable root is treated the same way as an
	// empty one, because the reader can do nothing different about either.
	// Two sources, in order: the flag, then what was chosen in Settings. There is no
	// third, because the recordings belong to the user and only the user knows where
	// they are; nothing is detected and nothing is guessed at.
	settings := config.NewSettings()
	stored := settings.Load()

	root := preferred(*libraryRoot, stored.LibraryRoot)
	found, report := scanLibrary(root, table)
	warnAbout(report)

	chooser := randomChooser{source: rand.New(rand.NewSource(time.Now().UnixNano()))}

	// The reporting flags are run from a terminal, where a refusal is read. They keep
	// it: a report over no voices is an empty page rather than an answer.
	if *listVoices || *showUnbound {
		if len(found) == 0 {
			// %s rather than %q for the same reason the window uses it: %q escapes
			// the path, so a Windows path is printed with every separator doubled.
			return fmt.Errorf("no voices under %s", root)
		}
	}
	if *listVoices {
		return listing(found, table, chooser)
	}

	// The same sources in the same order as the directories above; here an empty
	// answer reaches pick, which takes the first voice there is.
	wanted := preferred(*voice, stored.Voice)

	var chosen library.Voice
	if len(found) > 0 {
		chosen, err = pick(found, wanted)
		if err != nil {
			if *showUnbound {
				return err
			}
			// A voice that is not installed falls back to the first one rather than
			// stopping the application, for the same reason an empty library root does.
			// It covers a name typed on the command line and a stored one alike: a
			// voice uninstalled since it was last cast is the ordinary way this
			// happens, so it warns and speaks rather than refusing to start.
			fmt.Fprintf(os.Stderr, "warning: %v (using %q)\n", err, found[0].Name)
			chosen = found[0]
		}
	}
	if *showUnbound {
		return unboundReport(chosen, table, chooser)
	}

	player, err := audio.NewPlayer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v (running silent)\n", err)
	}

	making, maker, err := newMaking(table)
	if err != nil {
		return err
	}
	// Plugins are loaded before the window opens, so a voice one offers is there to be cast
	// rather than appearing later (FR-560). None of it can stop the run: every plugin passed
	// over is a line in the log (FR-567) and nothing else.
	executable, whereabouts := os.Executable()
	current := &session{
		table: table, available: found,
		chooser: chooser, player: player,
		making: making, maker: maker,
		plugins: loadPlugins(executable, whereabouts, runLog()),
		// Read once here, over the same store the directories came from (FR-629).
		chatter: services.NewChatterService(table, settings),
	}
	if !*noTray {
		current.tray = startTray(found, chosen.Name)
	}

	// A journal directory that cannot be watched is carried to the window rather than
	// returned (FR-238). The window is where it can be put right, so ending the run over
	// it left the one control that could fix it out of reach.
	watched := openJournal(*journalDir, stored.JournalDir, journal.StandardLocation)
	if watched.problem != "" {
		fmt.Fprintf(os.Stderr, "warning: %s (watching nothing)\n", watched.problem)
	}

	// The facade is the reporter, so every decision reaches the front end. It is
	// built before the first useVoice call so the very first cue is already logged.
	app := newApp(current, watched, root, settings)
	current.reporter = reporter{app}
	current.castAtStart(keptMachineVoice(*voice, stored.MachineVoice), chosen, os.Stderr)

	// Hidden with no tray would leave nothing on screen and no way to summon it, so
	// the window is shown rather than starting a program the user cannot reach. The
	// login entry never asks for that pair; only a hand-typed command can.
	return launch(app, *hidden && current.tray != nil)
}

// startTray builds and shows the tray, returning nil when it cannot appear.
//
// A tray that fails to start is not fatal. The application still watches the journal
// and still speaks, which is the whole point of it. The nil it answers then is the
// interface's own: a nil *taskbar.Tray held as a trayIcon would read as an icon that is there.
func startTray(found []library.Voice, active string) trayIcon {
	tray := taskbar.New(taskbar.Options{
		Title: appTitle, Voices: trayChoices(found), ActiveVoice: active,
	})
	if err := tray.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v (running without a tray icon)\n", err)
		return nil
	}
	return tray
}

// newMaking builds what a machine voice is made through: the model files beside the application
// (FR-539), the made lines in the product's local data folder (FR-523) and the model itself.
//
// Neither folder going unfound stops the application. Every machine voice is refused with why
// instead, before anything is written, so a store with no folder never writes into the one the
// application was started from.
func newMaking(table cue.Table) (*services.MakingService, *speechmodel.Maker, error) {
	voiced, err := config.LoadVoicedScript(table)
	if err != nil {
		return nil, nil, err
	}
	pauses, err := config.LoadPauses()
	if err != nil {
		return nil, nil, err
	}
	endings, err := config.LoadEndings()
	if err != nil {
		return nil, nil, err
	}
	executable, notFound := os.Executable()
	made, noStore := madelines.Dir()
	dir := voicefiles.Beside(executable)
	maker := speechmodel.New(dir)
	files := filesUnless(voicefiles.New(dir), errors.Join(notFound, noStore))
	// A table without the confirmation's cue makes nothing on a cast; every line is then made on call.
	confirmation, _ := table.Confirmation()
	return services.NewMakingService(voiced, pauses, endings, confirmation, files, maker, madelines.New(made), runLog()), maker, nil
}
