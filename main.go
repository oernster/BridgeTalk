// Command bridge-talk watches Elite Dangerous and speaks.
//
// This file is the composition root. It and app.go are the only files permitted to
// wire concrete infrastructure to the application layer; everything else works
// against the ports.
package main

import (
	"embed"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/infrastructure/audio"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/journal"
	"github.com/oernster/bridge-talk/internal/infrastructure/library"
	"github.com/oernster/bridge-talk/internal/infrastructure/setup"
	"github.com/oernster/bridge-talk/internal/infrastructure/status"
	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
	"github.com/oernster/bridge-talk/internal/product"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// pollInterval is how often both sources are asked for new events. The journal read
// is a seek and a read of only the appended bytes, usually none, so this is cheap
// enough to run beside the game.
const pollInterval = 250 * time.Millisecond

// appTitle names the application in the window title, the tray tooltip and the
// About dialog: the product name as a reader sees it.
const appTitle = product.Name

// Window geometry. The default is wide enough for the reaction log's columns without
// horizontal scrolling; the minimum is where the nav band stops fitting on one row.
//
// The minimum is measured rather than chosen; it is re-measured whenever the band
// gains a button. The band needs 1022 pixels for its eight buttons, the volume slider
// at its full width, the gaps and its own padding, so anything under that squeezes
// the slider or pushes the last button off the row; 1045 leaves the two groups
// visibly apart. The default grew with the icons, so a window opened at it has room
// for the band and a useful pane rather than the band and a sliver.
const (
	windowWidth     = 1120
	windowHeight    = 800
	windowMinWidth  = 1045
	windowMinHeight = 600
)

// systemClock is the real clock, injected so the domain never reads the wall clock.
type systemClock struct{}

// Now returns the current time.
func (systemClock) Now() time.Time { return time.Now() }

// randomChooser is the real source of randomness, injected for the same reason.
type randomChooser struct{ source *rand.Rand }

// Intn returns a value in [0, n).
func (r randomChooser) Intn(n int) int { return r.source.Intn(n) }

func main() {
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
	Stop()
	Playing() bool
	Done() <-chan struct{}
	Silent() bool
	Stalls() (count int, worst time.Duration)
	Volume() float64
	SetVolume(level float64)
	Close() error
}

type session struct {
	table     cue.Table
	available []library.Voice
	chooser   randomChooser
	player    audioPlayer
	tray      *taskbar.Tray
	reporter  ports.Reporter

	active    library.Voice
	catalogue *library.Catalogue
	scheduler *services.Scheduler
	reactions *services.ReactionService

	// muted lives here rather than on the reaction service because the service only
	// exists once a voice is cast. With no voices installed there is nothing to
	// silence and the control still has to answer, so the session holds the answer
	// and hands it to each service it builds.
	muted bool
}

// hasVoice reports whether a voice is cast.
//
// Nothing is cast until a voice is found. The window opens either way, so every path
// that would speak has to be able to do nothing instead of assuming a voice.
func (s *session) hasVoice() bool { return s.reactions != nil }

// useVoice rebuilds the catalogue and the reaction path for a different voice.
//
// The picker's memory of what each cue last played goes with the old service, which
// is correct: those clip paths belong to the voice being left behind.
func (s *session) useVoice(chosen library.Voice) {
	s.player.Stop()

	s.active = chosen
	s.catalogue = library.NewCatalogue(chosen, s.table, s.chooser)
	s.scheduler = services.NewScheduler(s.player, s.reporter, systemClock{})
	s.reactions = services.NewReactionService(
		s.table, s.catalogue, s.scheduler, s.chooser, s.reporter, systemClock{},
	)
	s.reactions.SetMuted(s.muted)
	if s.tray != nil {
		s.tray.SetActiveVoice(chosen.Name)
	}
}

// catalogueFor builds a catalogue over any voice, cast or not. The cast pane reports
// on voices the user has not chosen and the audition pane plays from them, so the
// catalogue cannot be tied to the active one.
func (s *session) catalogueFor(voice library.Voice) *library.Catalogue {
	return library.NewCatalogue(voice, s.table, s.chooser)
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

// coverageOf reports what a voice that is not the cast one could serve: the cues it
// has recorded, the size of the table and the takes it holds. All three are what the
// cast pane shows before the user commits to casting it.
func (s *session) coverageOf(voice library.Voice) (int, int, int) {
	catalogue := s.catalogueFor(voice)
	covered, total := catalogue.Coverage()
	reachable, _ := catalogue.Files()
	return covered, total, reachable
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

	current := &session{
		table: table, available: found,
		chooser: chooser, player: player,
	}
	if !*noTray {
		current.tray = startTray(found, chosen.Name)
	}

	directory := preferred(*journalDir, stored.JournalDir)
	if directory == "" {
		if directory, err = journal.StandardLocation(); err != nil {
			return err
		}
	}
	journalSource, err := journal.NewSource(directory, systemClock{}.Now)
	if err != nil {
		return err
	}
	statusSource, err := status.NewWatcher(directory, systemClock{}.Now)
	if err != nil {
		return err
	}

	// The facade is the reporter, so every decision reaches the front end. It is
	// built before the first useVoice call so the very first cue is already logged.
	app := newApp(
		current,
		[]ports.EventSource{journalSource, statusSource},
		directory, statusSource.Path(), root,
		settings,
	)
	current.reporter = reporter{app}
	if len(found) > 0 {
		current.useVoice(chosen)
	}

	// Hidden with no tray would leave nothing on screen and no way to summon it, so
	// the window is shown rather than starting a program the user cannot reach. The
	// login entry never asks for that pair; only a hand-typed command can.
	return launch(app, *hidden && current.tray != nil)
}

// launch runs the Wails window.
//
// Started hidden, the window exists but is not shown: the tray icon summons it, which
// is what the login entry wants. Anything else opens it as usual.
func launch(app *App, hidden bool) error {
	err := wails.Run(&options.App{
		StartHidden:      hidden,
		Title:            appTitle,
		Width:            windowWidth,
		Height:           windowHeight,
		MinWidth:         windowMinWidth,
		MinHeight:        windowMinHeight,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 12, G: 12, B: 14, A: 1},
		OnBeforeClose:    app.beforeClose,
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
	})
	if err != nil {
		return fmt.Errorf("running the window: %w", err)
	}
	return nil
}

// startTray builds and shows the tray, returning nil when it cannot appear.
//
// A tray that fails to start is not fatal. The application still watches the journal
// and still speaks, which is the whole point of it.
func startTray(found []library.Voice, active string) *taskbar.Tray {
	tray := taskbar.New(taskbar.Options{
		Title: appTitle, Voices: playable(found), ActiveVoice: active,
	})
	if err := tray.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v (running without a tray icon)\n", err)
		return nil
	}
	return tray
}
