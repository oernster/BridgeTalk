package journal

// Where the game keeps its journal when nothing was chosen (FR-811, FR-812).
//
// On Windows that is one place under the home directory. On Linux the game runs under Proton or
// Wine, so it writes inside a Windows prefix instead. A machine has several places that prefix may
// be. The order is o7 Debrief's, which found the live journal this way from inside its flatpak.

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// linuxOS is runtime.GOOS on Linux, where the journal sits inside the game's prefix.
const linuxOS = "linux"

// steamAppID is the game's Steam application id, which names its Proton prefix under compatdata.
const steamAppID = "359320"

// steamUser is the account Proton creates inside every prefix it makes.
const steamUser = "steamuser"

// savedGames is the path from a Windows account's folder to the journal directory.
var savedGames = []string{"Saved Games", "Frontier Developments", "Elite Dangerous"}

// steamRoots are the places Steam keeps its data under the home directory, in the order looked:
// the two links a native install makes, the directory they usually point at, then Steam installed
// as a flatpak.
var steamRoots = [][]string{
	{".steam", "steam"},
	{".steam", "root"},
	{".local", "share", "Steam"},
	{".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"},
}

// StandardLocation returns where the game keeps its journal for this user.
//
// On Windows it names the place without looking there. Whether the directory exists is asked by
// NewSource, which words the answer for the reader (FR-237); asking here as well gave a machine
// where the game has never run a second refusal, naming the path twice with every separator
// doubled. On Linux there is no one place to name, so it answers the first that exists; where none
// does, it names every place it looked (FR-812).
func StandardLocation() (string, error) {
	return standardLocation(runtime.GOOS, os.Getenv, os.UserHomeDir, isDirectory)
}

// standardLocation works the place out from the platform, the environment, the home directory and
// a question about the disk, each a parameter so every platform's rule runs on every platform.
func standardLocation(goos string, getenv func(string) string, home func() (string, error), exists func(string) bool) (string, error) {
	found, err := home()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	if goos != linuxOS {
		return filepath.Join(append([]string{found}, savedGames...)...), nil
	}
	looked := prefixCandidates(found, getenv)
	for _, directory := range looked {
		if exists(directory) {
			return directory, nil
		}
	}
	return "", fmt.Errorf("the game's journal directory is in none of the places it is looked for: %s", strings.Join(looked, "; "))
}

// prefixCandidates lists every directory the journal may be in on Linux, in the order looked
// (FR-811). A variable that is not set contributes nothing.
func prefixCandidates(home string, getenv func(string) string) []string {
	account := getenv("USER")
	var out []string
	if compat := getenv("STEAM_COMPAT_DATA_PATH"); compat != "" {
		out = append(out, inPrefix(filepath.Join(compat, "pfx"), steamUser, account)...)
	}
	for _, root := range steamRoots {
		prefix := filepath.Join(append(append([]string{home}, root...), "steamapps", "compatdata", steamAppID, "pfx")...)
		out = append(out, inPrefix(prefix, steamUser, account)...)
	}
	if wine := getenv("WINEPREFIX"); wine != "" {
		out = append(out, inPrefix(wine, account, steamUser)...)
	}
	return append(out, inPrefix(filepath.Join(home, ".wine"), account, steamUser)...)
}

// inPrefix names the journal directory inside a Windows prefix for each account in the order given,
// passing over an account with no name.
func inPrefix(prefix string, accounts ...string) []string {
	var out []string
	for _, account := range accounts {
		if account == "" {
			continue
		}
		out = append(out, filepath.Join(append([]string{prefix, "drive_c", "users", account}, savedGames...)...))
	}
	return out
}

// isDirectory reports whether a directory is there.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
