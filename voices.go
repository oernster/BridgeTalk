// Voice selection and reporting for the composition root.
//
// These helpers touch infrastructure and the domain but never the application
// services, so main.go remains the only file wiring the two together and the
// composition-root whitelist still holds.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/infrastructure/library"
	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
	"github.com/oernster/bridge-talk/internal/infrastructure/taskbar"
)

// playable returns the voices found as the tray offers them: each under the name it is shown
// by, chosen by the name that identifies it (FR-210).
//
// Every voice in the list holds at least one take, since that is what made it a voice
// during the scan, so there is nothing further to filter out here.
func playable(found []library.Voice) []taskbar.Choice {
	var choices []taskbar.Choice
	for _, candidate := range found {
		choices = append(choices, taskbar.Choice{
			Voice: taskbar.Voice{Kind: taskbar.Recorded, Name: candidate.Name},
			Label: candidate.Display(),
		})
	}
	return choices
}

// trayChoices is every voice the tray's Voice menu offers: the recorded voices found, then every
// machine voice, then every voice the loaded plugins offer, each under the name it is shown by and
// carrying what identifies it so a choice casts the right one (FR-509, FR-565).
//
// A plugin voice whose audio is not on this machine is left out rather than offered and refused.
// The menu has nowhere to say why: it closes on the click; the reason belongs beside the voice
// on the Cast pane, where it is already said (FR-570).
func trayChoices(found []library.Voice, offered []*plugin.Voice) []taskbar.Choice {
	choices := playable(found)
	for _, voice := range machinevoice.All() {
		choices = append(choices, taskbar.Choice{
			Voice: taskbar.Voice{Kind: taskbar.Machine, Name: voice.ID()},
			Label: voice.Name(),
		})
	}
	shown := pluginDisplays(offered)
	for index, voice := range offered {
		if !voice.Ready {
			continue
		}
		choices = append(choices, taskbar.Choice{
			Voice: taskbar.Voice{Kind: taskbar.Plugin, Plugin: voice.Plugin().Name, Name: voice.ID},
			Label: shown[index],
		})
	}
	return choices
}

// preferred answers with the first choice that was actually made.
//
// It names the rule the composition root applies to every setting that has more than
// one source: a flag is this run's instruction, so it beats what was stored, which is
// the last thing the reader chose, which in turn beats detection, the guess made for
// them. Detection is not a source here because each one detects differently and two
// of them can fail; an empty answer is the caller's cue to go and look.
func preferred(sources ...string) string {
	for _, source := range sources {
		if source != "" {
			return source
		}
	}
	return ""
}

// pick finds a voice by name, accepting any unambiguous case-insensitive prefix.
func pick(found []library.Voice, want string) (library.Voice, error) {
	// No preference takes the first voice there is. Scan returns them sorted, so that
	// is the alphabetically first one installed, whatever it happens to be called.
	// Naming a particular voice here would mean the application started only for
	// somebody who owned that one.
	if want == "" {
		if len(found) == 0 {
			return library.Voice{}, fmt.Errorf("no voices to choose from")
		}
		return found[0], nil
	}
	lowered := strings.ToLower(want)
	for _, candidate := range found {
		if strings.EqualFold(candidate.Name, want) {
			return candidate, nil
		}
	}
	var matches []library.Voice
	for _, candidate := range found {
		if strings.HasPrefix(strings.ToLower(candidate.Name), lowered) {
			matches = append(matches, candidate)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	names := make([]string, 0, len(found))
	for _, candidate := range found {
		names = append(names, candidate.Name)
	}
	return library.Voice{}, fmt.Errorf("voice %q not found, choose one of: %s", want, strings.Join(names, ", "))
}

// catalogueOf builds the catalogue over a recorded voice: its takes answered through the
// audio source port (FR-501), under the name the voice is shown by (FR-210).
func catalogueOf(voice library.Voice, table cue.Table, chooser randomChooser) *library.Catalogue {
	return catalogueOver(voice, voice.Display(), table, chooser)
}

// catalogueOver builds the catalogue over any audio source (FR-501), under the name its voice is
// shown by: a recorded voice's takes or a machine voice's made lines.
func catalogueOver(source ports.AudioSource, shown string, table cue.Table, chooser randomChooser) *library.Catalogue {
	return library.NewCatalogue(source, shown, table, chooser)
}

// unboundReport prints the cues a voice has nothing recorded for, then exits.
func unboundReport(chosen library.Voice, table cue.Table, chooser randomChooser) error {
	catalogue := catalogueOf(chosen, table, chooser)
	unbound := catalogue.Unbound()
	fmt.Printf("%s has nothing recorded for %d of %d cues:\n",
		chosen.Name, len(unbound), table.Len())
	for _, item := range unbound {
		fmt.Printf("  %-32s %s\n", item.ID(), item.Title())
	}
	return nil
}

// scanLibrary finds the voices under root, warning where a chosen root cannot be read.
//
// No root at all is the ordinary state of an application nobody has yet pointed at
// their recordings, so it finds nothing and says nothing. Scanning it anyway asked the
// file system to read a directory with no name and printed the refusal on every start,
// including the one Wails makes while generating bindings during a build. A root that
// was chosen but cannot be read still warns, because somebody chose it.
func scanLibrary(root string, table cue.Table) ([]library.Voice, library.Report) {
	if root == "" {
		return nil, library.Report{}
	}
	found, report, err := library.Scan(root, table)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		return nil, library.Report{}
	}
	return found, report
}

// warnAbout prints what a scan passed over, so nothing is skipped in silence.
//
// A misspelled folder name and a directory holding no recordings both leave an empty
// list otherwise; the reader cannot tell which they are looking at.
func warnAbout(report library.Report) {
	groups := [][]library.Reason{
		report.Empty, report.Unmatched, report.Duplicated, report.Undecodable, report.Manifest,
	}
	for _, group := range groups {
		for _, reason := range group {
			fmt.Fprintf(os.Stderr, "note: %s: %s\n", reason.Path, reason.Why)
		}
	}
}

// listing prints every voice with its takes and cue coverage, then exits.
func listing(found []library.Voice, table cue.Table, chooser randomChooser) error {
	fmt.Printf("%-20s %8s  %s\n", "voice", "takes", "cues recorded")
	for _, voice := range found {
		catalogue := catalogueOf(voice, table, chooser)
		served, total := catalogue.Coverage()
		fmt.Printf("%-20s %8d  %d of %d\n", voice.Name, voice.Takes, served, total)
	}
	return nil
}

// catalogueFor builds a catalogue over any voice, cast or not. The cast pane reports
// on voices the user has not chosen and the audition pane plays from them, so the
// catalogue cannot be tied to the active one.
func (s *session) catalogueFor(voice library.Voice) *library.Catalogue {
	return catalogueOf(voice, s.table, s.chooser)
}

// heard answers whether a moment is switched on in Chatter as it stands, for an audition to draw on
// (FR-745). A session built without the switches, as a test may build one, hears every moment.
func (s *session) heard() cue.Heard {
	if s.chatter == nil {
		return cue.HeardAll
	}
	chatter := s.chatter
	return func(id cue.ID) bool { return !chatter.Off(id) }
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

// startTray builds and shows the tray, returning nil when it cannot appear.
//
// A tray that fails to start is not fatal. The application still watches the journal
// and still speaks, which is the whole point of it. The nil it answers then is the
// interface's own: a nil *taskbar.Tray held as a trayIcon would read as an icon that is there.
func startTray(found []library.Voice, offered []*plugin.Voice, active taskbar.Voice) trayIcon {
	tray := taskbar.New(taskbar.Options{
		Title: appTitle, Voices: trayChoices(found, offered), Active: active, Icon: applicationIcon,
	})
	if err := tray.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v (running without a tray icon)\n", err)
		return nil
	}
	return tray
}
