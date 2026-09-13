package structural

// The setup page's keyboard (FR-808). The ring itself is exercised by the front end's suite,
// which loads the page's script and presses keys. What that suite cannot see is whether the
// page loads the script at all and whether the body wears the ring when focus lands on it, so
// both are read here.

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// setupSheet is the setup page's style sheet; setupScriptTag matches one script the page loads.
var (
	setupSheet     = "setup.css"
	setupScriptTag = regexp.MustCompile(`<script src="([^"]+)"></script>`)
)

// TestTheSetupPageLoadsEveryScript keeps the page, its scripts and setupScripts in step.
//
// A script the page never loads defines nothing that runs, which no test of the script alone
// can notice; one the page loads without a place in setupScripts escapes every rule read from
// that list.
//
// Proved by removing the ring's script tag from the page, then separately a script from the
// list, reading the exit code each time.
func TestTheSetupPageLoadsEveryScript(t *testing.T) {
	dir := filepath.Join(repoRoot(t), setupFrontendDir)
	raw, err := os.ReadFile(filepath.Join(dir, setupPage))
	if err != nil {
		t.Fatalf("reading the setup page: %v", err)
	}
	loaded := map[string]bool{}
	for _, match := range setupScriptTag.FindAllSubmatch(raw, -1) {
		loaded[string(match[1])] = true
	}
	for _, name := range setupScripts {
		if !loaded[name] {
			t.Errorf("the setup page never loads %s, so nothing it defines runs", name)
		}
	}
	for _, path := range setupFrontendFiles(t) {
		name := filepath.Base(path)
		if strings.EqualFold(filepath.Ext(name), ".js") && !slices.Contains(setupScripts, name) {
			t.Errorf("%s sits beside the setup page without a place in setupScripts", name)
		}
	}
}

// TestTheSetupBodyRingsForTheKeyboard holds the ring on the setup page's body, the one stop
// there that is a stop because it scrolls. Without it, Tab lands on the body and nothing on
// screen says so.
//
// Proved by removing the rule from the sheet, reading the exit code.
func TestTheSetupBodyRingsForTheKeyboard(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), setupFrontendDir, setupSheet))
	if err != nil {
		t.Fatalf("reading %s: %v", setupSheet, err)
	}
	if !ringsOnFocus(parseRules(setupSheet, raw), ".body") {
		t.Errorf("%s gives .body no keyboard ring, so focus landing on it shows nothing", setupSheet)
	}
}
