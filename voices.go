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

	"github.com/oernster/bridge-talk/internal/domain/cue"
	"github.com/oernster/bridge-talk/internal/infrastructure/library"
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
		choices = append(choices, taskbar.Choice{Name: candidate.Name, Label: candidate.Display()})
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

// unboundReport prints the cues a voice has nothing recorded for, then exits.
func unboundReport(chosen library.Voice, table cue.Table, chooser randomChooser) error {
	catalogue := library.NewCatalogue(chosen, table, chooser)
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
		catalogue := library.NewCatalogue(voice, table, chooser)
		served, total := catalogue.Coverage()
		fmt.Printf("%-20s %8d  %d of %d\n", voice.Name, voice.Takes, served, total)
	}
	return nil
}
