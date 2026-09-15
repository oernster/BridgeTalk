package main

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// wailsBuildTags are the tags Wails v2.12.0 builds a desktop application with for release: its
// output type then the production mode (pkg/commands/build/base.go, read on 2026-09-15).
const wailsBuildTags = "desktop,production"

// FR-712: About credits every Go module the released binary links, found by asking the toolchain
// for the build graph under the tags the release is built with.
func TestEveryModuleTheReleasedBinaryLinksIsCredited(t *testing.T) {
	listed, err := exec.Command("go", "list", "-tags", wailsBuildTags, "-deps",
		"-f", "{{if and .Module (not .Module.Main)}}{{.Module.Path}}{{end}}", ".").Output()
	if err != nil {
		t.Fatalf("listing the modules the binary links: %v", err)
	}
	credited := make(map[string]bool, len(shipped))
	for _, each := range shipped {
		credited[each.module] = true
	}
	var missing []string
	for _, module := range strings.Fields(string(listed)) {
		if !credited[module] && !slices.Contains(missing, module) {
			missing = append(missing, module)
		}
	}
	if len(missing) > 0 {
		t.Errorf("About credits none of these modules the binary links: %s", strings.Join(missing, ", "))
	}
}

// The dialog under Help shows the LICENSE file byte for byte, so the terms a reader is
// shown and the terms the source carries cannot drift apart.
func TestTheLicenceDialogShowsTheLicenceFileItself(t *testing.T) {
	app, _, _ := fixtureApp(t)

	raw, err := os.ReadFile("LICENSE")
	if err != nil {
		t.Fatalf("reading LICENSE: %v", err)
	}
	if app.Licence() != string(raw) {
		t.Fatal("the licence dialog text differs from the LICENSE file")
	}
}

// About names the terms in a sentence and points at the full text, so the licence is
// stated where a reader looks for who made the application.
func TestAboutNamesTheLicence(t *testing.T) {
	app, _, _ := fixtureApp(t)

	if got := app.About().Licence; !strings.Contains(got, "GNU General Public License") {
		t.Fatalf("About licence = %q, want it to name the GNU General Public License", got)
	}
}
