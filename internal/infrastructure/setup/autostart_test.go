package setup

// The Linux sign-in entry (FR-815), run on every platform through the parameters autostart.go takes.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/product"
)

// settingsOf answers the variables it holds and nothing for any other.
func settingsOf(set map[string]string) func(string) string {
	return func(name string) string { return set[name] }
}

// pilot is the home directory of the worked examples.
var pilot = filepath.Join("home", "pilot")

// FR-815's worked example: inside the flatpak the entry goes to the real ~/.config/autostart, not
// to the sandbox's own configuration XDG_CONFIG_HOME names; it starts the flatpak.
func TestInsideTheFlatpakTheEntryIgnoresTheSandboxConfiguration(t *testing.T) {
	getenv := settingsOf(map[string]string{
		"FLATPAK_ID":      product.AppID,
		"XDG_CONFIG_HOME": filepath.Join(pilot, ".var", "app", product.AppID, "config"),
	})
	if got, want := autostartPath(getenv, pilot), filepath.Join(pilot, ".config", "autostart", product.AppID+".desktop"); got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
	if entry := autostartEntry(getenv, "/app/bin/"+product.Slug); !strings.Contains(entry, "\nExec=flatpak run "+product.AppID+" -hidden\n") {
		t.Errorf("entry = %q, want it to start the flatpak hidden", entry)
	}
}

// Outside the flatpak the entry follows XDG_CONFIG_HOME where it is set, else ~/.config.
func TestOutsideTheFlatpakTheEntryFollowsTheConfigurationDirectory(t *testing.T) {
	custom := filepath.Join("elsewhere", "config")
	if got, want := autostartPath(settingsOf(map[string]string{"XDG_CONFIG_HOME": custom}), pilot), filepath.Join(custom, "autostart", product.AppID+".desktop"); got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
	if got, want := autostartPath(settingsOf(nil), pilot), filepath.Join(pilot, ".config", "autostart", product.AppID+".desktop"); got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

// Outside the flatpak the entry starts the program running, quoted so a space or a reserved
// character in its path reaches the desktop whole.
func TestOutsideTheFlatpakTheEntryStartsTheRunningProgramQuoted(t *testing.T) {
	entry := autostartEntry(settingsOf(nil), "/home/pilot/my $games/quoted\"name")
	if want := "\nExec=\"/home/pilot/my \\$games/quoted\\\"name\" -hidden\n"; !strings.Contains(entry, want) {
		t.Errorf("entry = %q, want it to hold %q", entry, want)
	}
	if !strings.HasPrefix(entry, "[Desktop Entry]\nType=Application\nName="+product.Name+"\n") {
		t.Errorf("entry = %q, want a desktop entry naming the product", entry)
	}
}

// Ticking writes the entry, making its directory; the box then reads ticked. Unticking removes it;
// unticking again is no failure.
func TestTheEntryIsWrittenReadBackAndRemoved(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".config", "autostart", autostartFile)
	if autostartPresent(path) {
		t.Fatal("an entry was present before anything wrote one")
	}
	if err := applyAutostart(path, "contents", true); err != nil {
		t.Fatalf("writing the entry: %v", err)
	}
	if written, err := os.ReadFile(path); err != nil || string(written) != "contents" {
		t.Fatalf("entry = %q, %v; want what was written", written, err)
	}
	if !autostartPresent(path) {
		t.Error("a written entry did not read back as present")
	}
	for range 2 {
		if err := applyAutostart(path, "", false); err != nil {
			t.Fatalf("removing the entry: %v", err)
		}
	}
	if autostartPresent(path) {
		t.Error("a removed entry still read as present")
	}
}

// A directory that cannot be made is refused with the reason.
func TestAnEntryThatCannotBeWrittenSaysWhy(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(blocker, nil, autostartMode); err != nil {
		t.Fatalf("placing a file where the directory would go: %v", err)
	}
	if err := applyAutostart(filepath.Join(blocker, "autostart", autostartFile), "contents", true); err == nil {
		t.Fatal("an entry under a file was reported written")
	}
}
