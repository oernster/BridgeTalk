package journal

// Where the journal is looked for when nothing was chosen (FR-811, FR-812), each platform's rule run
// on every platform through the parameters standardLocation takes.

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// pilotHome is the home directory of the worked examples.
var pilotHome = filepath.Join("home", "pilot")

// environment answers the variables it holds and nothing for any other.
func environment(set map[string]string) func(string) string {
	return func(name string) string { return set[name] }
}

// at answers a home directory.
func at(home string) func() (string, error) {
	return func() (string, error) { return home, nil }
}

// journalIn names the journal directory inside a prefix for one account.
func journalIn(prefix, account string) string {
	return filepath.Join(prefix, "drive_c", "users", account, "Saved Games", "Frontier Developments", "Elite Dangerous")
}

// steamPrefix names the game's Proton prefix under a Steam root inside the pilot's home.
func steamPrefix(root ...string) string {
	return filepath.Join(append(append([]string{pilotHome}, root...), "steamapps", "compatdata", "359320", "pfx")...)
}

// asked records every directory a lookup asks about, answering yes only for the one given.
func asked(present string, looked *[]string) func(string) bool {
	return func(path string) bool {
		*looked = append(*looked, path)
		return path == present
	}
}

// Off Linux the journal is named under Saved Games without asking whether it is there, so NewSource
// is the one place that words a directory that is not (FR-237).
func TestOffLinuxTheJournalIsNamedUnderSavedGames(t *testing.T) {
	var looked []string
	got, err := standardLocation("windows", environment(nil), at(pilotHome), asked("", &looked))
	if err != nil {
		t.Fatalf("standard location: %v", err)
	}
	if want := filepath.Join(pilotHome, "Saved Games", "Frontier Developments", "Elite Dangerous"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if len(looked) != 0 {
		t.Errorf("asked about %v, want nothing asked", looked)
	}
}

// FR-811's worked example: only the prefix under ~/.local/share/Steam holds the journal.
func TestOnLinuxTheFirstPrefixHoldingTheJournalIsWatched(t *testing.T) {
	present := journalIn(steamPrefix(".local", "share", "Steam"), "steamuser")
	var looked []string
	got, err := standardLocation(linuxOS, environment(map[string]string{"USER": "pilot"}), at(pilotHome), asked(present, &looked))
	if err != nil {
		t.Fatalf("standard location: %v", err)
	}
	if got != present {
		t.Errorf("got %q, want %q", got, present)
	}
}

// FR-811's order in full: the compatibility prefix Steam names, each Steam root, the Wine prefix
// named, then the default Wine prefix; Proton's own account first inside a Proton prefix, the user's
// own first inside a Wine one.
func TestOnLinuxEveryPlaceIsLookedInItsOrder(t *testing.T) {
	compat := filepath.Join("games", "compatdata", "359320")
	wine := filepath.Join("games", "wine")
	set := map[string]string{"USER": "pilot", "STEAM_COMPAT_DATA_PATH": compat, "WINEPREFIX": wine}
	var looked []string
	if _, err := standardLocation(linuxOS, environment(set), at(pilotHome), asked("", &looked)); err == nil {
		t.Fatal("a lookup that found nothing answered a directory")
	}
	var want []string
	for _, prefix := range []string{
		filepath.Join(compat, "pfx"),
		steamPrefix(".steam", "steam"),
		steamPrefix(".steam", "root"),
		steamPrefix(".local", "share", "Steam"),
		steamPrefix(".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
	} {
		want = append(want, journalIn(prefix, "steamuser"), journalIn(prefix, "pilot"))
	}
	for _, prefix := range []string{wine, filepath.Join(pilotHome, ".wine")} {
		want = append(want, journalIn(prefix, "pilot"), journalIn(prefix, "steamuser"))
	}
	if !reflect.DeepEqual(looked, want) {
		t.Errorf("looked in\n%s\nwant\n%s", strings.Join(looked, "\n"), strings.Join(want, "\n"))
	}
}

// A variable that is not set contributes no directory (FR-811): no compatibility prefix, no named
// Wine prefix and no account of the user's own.
func TestOnLinuxAVariableNotSetContributesNothing(t *testing.T) {
	var looked []string
	_, _ = standardLocation(linuxOS, environment(nil), at(pilotHome), asked("", &looked))
	want := []string{
		journalIn(steamPrefix(".steam", "steam"), "steamuser"),
		journalIn(steamPrefix(".steam", "root"), "steamuser"),
		journalIn(steamPrefix(".local", "share", "Steam"), "steamuser"),
		journalIn(steamPrefix(".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"), "steamuser"),
		journalIn(filepath.Join(pilotHome, ".wine"), "steamuser"),
	}
	if !reflect.DeepEqual(looked, want) {
		t.Errorf("looked in\n%s\nwant\n%s", strings.Join(looked, "\n"), strings.Join(want, "\n"))
	}
}

// FR-812: where no prefix holds the journal, the reason names every place looked, in order.
func TestOnLinuxNoPrefixNamesEveryPlaceLooked(t *testing.T) {
	var looked []string
	_, err := standardLocation(linuxOS, environment(map[string]string{"USER": "pilot"}), at(pilotHome), asked("", &looked))
	if err == nil {
		t.Fatal("a lookup that found nothing answered a directory")
	}
	reason := err.Error()
	from := 0
	for _, place := range looked {
		at := strings.Index(reason[from:], place)
		if at < 0 {
			t.Fatalf("the reason %q does not name %q after what it named before", reason, place)
		}
		from += at + len(place)
	}
}

// With no home directory there is nowhere to look, on any platform.
func TestNoHomeIsRefusedOnEveryPlatform(t *testing.T) {
	noHome := func() (string, error) { return "", errors.New("no home") }
	for _, goos := range []string{"windows", linuxOS} {
		if _, err := standardLocation(goos, environment(nil), noHome, func(string) bool { return true }); err == nil {
			t.Errorf("%s: an unresolvable home directory was accepted", goos)
		}
	}
}

// The real lookup answers for this machine as its platform's rule says, reading the real home and
// the real disk.
func TestTheRealLookupReadsThisMachine(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("USER", "pilot")
	t.Setenv("STEAM_COMPAT_DATA_PATH", "")
	t.Setenv("WINEPREFIX", "")

	want, wantErr := standardLocation(runtime.GOOS, os.Getenv, os.UserHomeDir, isDirectory)
	got, err := StandardLocation()
	if got != want || (err == nil) != (wantErr == nil) {
		t.Errorf("StandardLocation() = %q, %v; want %q, %v", got, err, want, wantErr)
	}
}

// isDirectory answers yes for a directory alone.
func TestIsDirectoryAnswersForADirectoryAlone(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Journal.log")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatalf("writing: %v", err)
	}
	if !isDirectory(dir) || isDirectory(file) || isDirectory(filepath.Join(dir, "gone")) {
		t.Errorf("isDirectory: directory %v, file %v, missing %v", isDirectory(dir), isDirectory(file), isDirectory(filepath.Join(dir, "gone")))
	}
}
