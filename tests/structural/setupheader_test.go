package structural

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// setupPage is the setup program's hand-written page; setupScripts are the scripts that fill
// it, every one of which the page loads (TestTheSetupPageLoadsEveryScript).
var (
	setupPage    = "index.html"
	setupScripts = []string{"setup-ring.js", "setup-shell.js", "setup-routes.js"}
)

// setupHeader captures the page's header block: everything from the header's opening tag
// to the body that follows it.
var setupHeader = regexp.MustCompile(`(?s)<div class="head">(.*?)<div class="body">`)

// headerTitle matches a title element in that block, by its class or by the id a script
// once filled with the program's name; brandWrite matches a script reaching for that id.
var (
	headerTitle = regexp.MustCompile(`class="title"|id="brand"`)
	brandWrite  = regexp.MustCompile(`\$\('brand'\)`)
)

// TestTheSetupHeaderRepeatsNoTitle holds FR-234 on the setup program.
//
// Windows draws the setup window's title bar with the program's name and the word Setup.
// A title line in the page header directly beneath it said the same words again. The page
// has no build step and no test runner of its own, so the rule is held here by reading it.
//
// Proved by putting the title element back in the page, then separately the script line
// that filled it, reading the exit code each time.
func TestTheSetupHeaderRepeatsNoTitle(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(root, setupFrontendDir)

	raw, err := os.ReadFile(filepath.Join(dir, setupPage))
	if err != nil {
		t.Fatalf("reading the setup page: %v", err)
	}
	header := setupHeader.FindSubmatch(raw)
	if header == nil {
		t.Fatal("the setup page has no header block, the pattern is wrong")
	}
	if found := headerTitle.Find(header[1]); found != nil {
		t.Errorf("the setup header holds %q: the title bar above it already names the program", found)
	}

	for _, name := range setupScripts {
		script, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if brandWrite.Match(script) {
			t.Errorf("%s writes a title into the setup header, repeating the title bar above it", name)
		}
	}
}
