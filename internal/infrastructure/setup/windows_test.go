//go:build windows

package setup

// The Windows side of setup acts on the machine itself, so these tests are careful
// about which half of it they touch.
//
// Everything that reads is called for real: a registry read tells the truth about
// this machine and cannot damage it. Everything that WRITES is exercised only after
// the user's own directories have been redirected into a temporary tree and that
// redirection has been proved to have taken effect. The registry writes are not
// exercised at all, because there is nowhere to redirect them to: writing the login
// entry or the uninstall key would change the machine the test is running on, which
// is not a test's to change. Those are named in TESTING.md as the deliberate gap.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// under reports whether a path sits inside a directory, which is what proves a
// redirection took. It answers false rather than guessing: an empty path, an unrelated
// one and a sibling whose name merely starts the same way are all outside. Case is
// folded because the paths come from Windows.
func under(path, dir string) bool {
	if path == "" || dir == "" {
		return false
	}
	relative, err := filepath.Rel(strings.ToLower(dir), strings.ToLower(path))
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// redirectUserDirectories points the Desktop and Start Menu lookups into a temporary
// tree and proves it worked, so nothing below can reach the real ones.
func redirectUserDirectories(t *testing.T) (desktop, startMenu string) {
	t.Helper()
	home := t.TempDir()
	appData := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", appData)

	desktop, err := DesktopDir()
	if err != nil {
		t.Fatalf("desktop dir: %v", err)
	}
	startMenu, err = StartMenuProgramsDir()
	if err != nil {
		t.Fatalf("start menu dir: %v", err)
	}

	// The guard: if either still points outside the temporary tree the redirection
	// did not take; a test that removes shortcuts would then remove the real ones.
	for _, each := range []string{desktop, startMenu} {
		if !under(each, home) && !under(each, appData) {
			t.Fatalf("%q is outside the temporary tree; the redirection did not take", each)
		}
	}
	for _, each := range []string{desktop, startMenu} {
		if err := os.MkdirAll(each, dirPerm); err != nil {
			t.Fatalf("creating %q: %v", each, err)
		}
	}
	return desktop, startMenu
}

func TestTheUsersOwnDirectoriesComeFromTheEnvironment(t *testing.T) {
	home := t.TempDir()
	appData := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", appData)

	desktop, err := DesktopDir()
	if err != nil {
		t.Fatalf("desktop dir: %v", err)
	}
	if desktop != filepath.Join(home, "Desktop") {
		t.Fatalf("desktop = %q", desktop)
	}

	programs, err := StartMenuProgramsDir()
	if err != nil {
		t.Fatalf("start menu dir: %v", err)
	}
	want := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs")
	if programs != want {
		t.Fatalf("start menu = %q, want %q", programs, want)
	}
}

func TestAMachineWithNoHomeOrApplicationDataIsReported(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	t.Setenv("APPDATA", "")

	if _, err := DesktopDir(); err == nil {
		t.Error("a desktop directory was invented with no home")
	}
	if _, err := StartMenuProgramsDir(); err == nil {
		t.Error("a start menu directory was invented with no APPDATA")
	}
}

// The manage and reinstall screens open with the boxes reflecting the machine, so a
// shortcut that is there has to read as there and one that is not has to read as not.
func TestTheShortcutBoxesReflectWhatIsOnTheMachine(t *testing.T) {
	desktop, startMenu := redirectUserDirectories(t)

	if present := CurrentShortcuts(); present.Desktop || present.StartMenu {
		t.Fatalf("got %+v with nothing on disk, want both false", present)
	}

	if err := os.WriteFile(filepath.Join(desktop, shortcutName), []byte("x"), 0o644); err != nil {
		t.Fatalf("planting the desktop shortcut: %v", err)
	}
	if present := CurrentShortcuts(); !present.Desktop || present.StartMenu {
		t.Fatalf("got %+v with only the desktop shortcut on disk", present)
	}

	if err := os.WriteFile(filepath.Join(startMenu, shortcutName), []byte("x"), 0o644); err != nil {
		t.Fatalf("planting the start menu shortcut: %v", err)
	}
	if present := CurrentShortcuts(); !present.Desktop || !present.StartMenu {
		t.Fatalf("got %+v with both shortcuts on disk", present)
	}
}

// Where the directories cannot be resolved at all there is nothing to look in, so
// both boxes read as empty rather than as an error the screen cannot show.
func TestTheShortcutBoxesReadEmptyWhereThereAreNoDirectories(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	t.Setenv("APPDATA", "")

	if present := CurrentShortcuts(); present.Desktop || present.StartMenu {
		t.Fatalf("got %+v with nowhere to look, want both false", present)
	}
}

// Unticking a box on a reinstall has to take the shortcut away rather than leaving a
// stale one behind; an uninstall removes both. Only the removing half is
// exercised here: creating one shells out to the Windows Script Host.
func TestUntickingAShortcutRemovesIt(t *testing.T) {
	desktop, startMenu := redirectUserDirectories(t)
	for _, dir := range []string{desktop, startMenu} {
		if err := os.WriteFile(filepath.Join(dir, shortcutName), []byte("x"), 0o644); err != nil {
			t.Fatalf("planting a shortcut in %q: %v", dir, err)
		}
	}

	RemoveShortcuts()

	for _, dir := range []string{desktop, startMenu} {
		if _, err := os.Stat(filepath.Join(dir, shortcutName)); err == nil {
			t.Errorf("the shortcut in %q survived", dir)
		}
	}
	// Removing what is already gone is not a failure: an uninstall run twice must
	// not report one.
	RemoveShortcuts()
}

// A missing shortcut is not worth failing an install over, so a location that cannot
// be resolved is skipped rather than reported.
func TestApplyingShortcutsWithNowhereToPutThemIsNotAFailure(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	t.Setenv("APPDATA", "")

	ApplyShortcuts("", "", Shortcuts{})
}

// The three registry reads answer for this machine, so what they return is not a
// test's to assert. What is a test's to assert is that each one answers rather than
// failing, since every one of them is on the path the setup window opens on.
func TestTheRegistryReadsAnswerWithoutFailing(t *testing.T) {
	_ = SystemPrefersDark()
	_ = IsLaunchOnBoot()
	_ = HasLaunchOnBootEntry()

	if version, installed := InstalledVersion(); installed && version == "" {
		t.Fatal("the application reported as installed with no version")
	}
}

// The login entry is written quoted, because writing it with Go's own quoting shipped
// a path with every separator doubled and Windows spent every sign-in looking for a
// place that does not exist.
func TestTheLoginEntryNamesTheRealPathAndReadsBack(t *testing.T) {
	path := filepath.Join(`C:\Users\Someone\AppData\Local`,
		installSubdir, InstallFolder, ExeName)

	value := runValue(path)
	if got := runTarget(value); got != path {
		t.Fatalf("the entry read back as %q, want %q", got, path)
	}
	// An older entry was written without quotes or arguments; a path with a
	// space in it is ordinary, so an unquoted value is taken whole.
	if got := runTarget(`C:\Program Files\App\app.exe`); got != `C:\Program Files\App\app.exe` {
		t.Fatalf("an unquoted entry read back as %q", got)
	}
}
