package main

import (
	"os"
	"strings"
	"testing"
)

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
