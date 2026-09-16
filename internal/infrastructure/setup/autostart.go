package setup

// The sign-in entry on Linux: an autostart desktop file (FR-815).
//
// It is worked out here with no build tag, taking the environment and the home directory as
// parameters, so the rule runs on every platform; boot_linux.go reads and writes the file itself.
//
// Inside a flatpak XDG_CONFIG_HOME points into the sandbox's own configuration, where no session
// reads an autostart entry. o7 Debrief wrote its entry there, read it back as on and started
// nothing until the real directory was used, so the flatpak writes to ~/.config/autostart whatever
// that variable says.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// autostartMode lets the user read and change the entry and everyone else read it, as a desktop
// file under the configuration directory is written.
const autostartMode = 0o644

// autostartDirMode is the autostart directory's own mode where it has to be made.
const autostartDirMode = 0o755

// autostartFile is the entry's file name, named for the application id as the desktop expects.
const autostartFile = product.AppID + ".desktop"

// flatpakVariable is set by flatpak inside the sandbox, naming the application running there.
const flatpakVariable = "FLATPAK_ID"

// desktopReserved are the characters a desktop entry's Exec value escapes inside quotes.
const desktopReserved = "\"`$\\"

// inFlatpak reports whether the application runs inside its flatpak.
func inFlatpak(getenv func(string) string) bool { return getenv(flatpakVariable) != "" }

// autostartPath answers where the sign-in entry is written for a home directory.
func autostartPath(getenv func(string) string, home string) string {
	base := getenv("XDG_CONFIG_HOME")
	if base == "" || inFlatpak(getenv) {
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "autostart", autostartFile)
}

// autostartEntry answers the entry's contents: inside the flatpak it starts the flatpak; outside it
// starts the program running, which is the path given.
func autostartEntry(getenv func(string) string, running string) string {
	command := "flatpak run " + product.AppID
	if !inFlatpak(getenv) {
		command = desktopQuote(running)
	}
	return "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=" + product.Name + "\n" +
		"Exec=" + command + " " + HiddenFlag + "\n" +
		"X-GNOME-Autostart-enabled=true\n"
}

// applyAutostart writes the entry at path when enabled; otherwise it removes whatever entry is there.
// An entry already gone is what removing it asks for, so that is no failure.
func applyAutostart(path, contents string, enabled bool) error {
	if !enabled {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("removing %s: %w", path, refusal.Reason(err))
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), autostartDirMode); err != nil {
		return fmt.Errorf("making %s: %w", filepath.Dir(path), refusal.Reason(err))
	}
	if err := os.WriteFile(path, []byte(contents), autostartMode); err != nil {
		return fmt.Errorf("writing %s: %w", path, refusal.Reason(err))
	}
	return nil
}

// autostartPresent reports whether an entry is at path.
func autostartPresent(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// desktopQuote quotes a path for an Exec value, escaping the characters the desktop entry
// specification reserves inside quotes.
func desktopQuote(path string) string {
	var quoted strings.Builder
	quoted.WriteByte('"')
	for _, character := range path {
		if strings.ContainsRune(desktopReserved, character) {
			quoted.WriteByte('\\')
		}
		quoted.WriteRune(character)
	}
	quoted.WriteByte('"')
	return quoted.String()
}
