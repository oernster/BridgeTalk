package library

// Making a voice's folders: the one write the library package performs.
//
// A folder named exactly for a cue id is the folder form of the drop-in convention, so
// making every one of them in advance means the person filling a voice never types a
// cue id at all. They put a recording in the folder for its moment; any file name will
// do. The recorder writes into the same folders, so a recorded voice and a hand filled
// one are the same thing on disk.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/oernster/bridge-talk/internal/domain/cue"
)

// folderPerm is the permission a made folder carries: the owner writes, anyone reads.
const folderPerm fs.FileMode = 0o755

// forbiddenInNames are the characters Windows refuses in a folder name.
//
// Windows rules apply on every platform, including one that would accept them. A
// library is expected to move between Windows and Linux; a voice named on Linux with a
// colon in it would arrive on Windows as a folder nobody can open.
const forbiddenInNames = `<>:"/\|?*`

// reservedNames are the device names Windows keeps, which no folder may take, with or
// without an extension after them.
var reservedNames = map[string]struct{}{
	"con": {}, "prn": {}, "aux": {}, "nul": {},
	"com1": {}, "com2": {}, "com3": {}, "com4": {}, "com5": {}, "com6": {}, "com7": {}, "com8": {}, "com9": {},
	"lpt1": {}, "lpt2": {}, "lpt3": {}, "lpt4": {}, "lpt5": {}, "lpt6": {}, "lpt7": {}, "lpt8": {}, "lpt9": {},
}

var (
	// ErrNoRoot means no recordings directory is chosen, so there is nowhere to make
	// anything.
	ErrNoRoot = errors.New("no recordings directory is chosen")
	// ErrNoVoiceName means the name was empty.
	ErrNoVoiceName = errors.New("type a name for the voice")
	// ErrVoiceName means the name cannot be a folder name on every platform.
	ErrVoiceName = errors.New("choose another name")
)

// maker creates one directory. MakeVoiceFolders passes os.Mkdir.
//
// It is a parameter for the reason lister is: a refusal from the file system partway
// through a voice cannot be staged on a real disk from a test, since Windows lets its
// owner create a folder inside a read only one.
type maker func(path string, perm fs.FileMode) error

// CheckVoiceName reports why a name cannot be a voice's folder; nil where it can.
//
// It runs before anything is asked for or made, so a name that would be refused never
// opens a dialog first.
func CheckVoiceName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrNoVoiceName
	}
	if strings.TrimSpace(name) != name {
		return fmt.Errorf("%q starts or ends with a space, which Windows removes from a folder name: %w", name, ErrVoiceName)
	}
	if strings.HasSuffix(name, ".") {
		return fmt.Errorf("%q ends with a dot, which Windows removes from a folder name: %w", name, ErrVoiceName)
	}
	for _, r := range name {
		if strings.ContainsRune(forbiddenInNames, r) || unicode.IsControl(r) {
			return fmt.Errorf("%q holds %q, which a folder name cannot: %w", name, r, ErrVoiceName)
		}
	}
	stem, _, _ := strings.Cut(strings.ToLower(name), ".")
	if _, reserved := reservedNames[stem]; reserved {
		return fmt.Errorf("%q is a name Windows keeps for a device: %w", name, ErrVoiceName)
	}
	return nil
}

// MakeVoiceFolders makes root/name, then one folder inside it for every cue id that has
// none yet. It answers with the voice's directory and how many folders it made.
//
// It adds and never replaces. A folder already there is left as it is, recordings and
// all; so is a file standing where a folder would go. Running it again after the cue
// vocabulary grows therefore makes exactly the folders for the new cues.
func MakeVoiceFolders(root, name string, table cue.Table) (string, int, error) {
	return makeVoiceFolders(root, name, table, os.Mkdir)
}

func makeVoiceFolders(root, name string, table cue.Table, mkdir maker) (string, int, error) {
	if err := CheckVoiceName(name); err != nil {
		return "", 0, err
	}
	if root == "" {
		return "", 0, ErrNoRoot
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", 0, fmt.Errorf("reading %s: %w", root, err)
	}
	if !info.IsDir() {
		return "", 0, fmt.Errorf("%s is not a directory", root)
	}

	// The name has been checked to hold no separator and to be neither "." nor "..",
	// which is what keeps the join inside the root.
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, folderPerm); err != nil {
		return "", 0, fmt.Errorf("making %s: %w", dir, err)
	}

	made := 0
	for _, item := range table.All() {
		target := filepath.Join(dir, string(item.ID()))
		err := mkdir(target, folderPerm)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return dir, made, fmt.Errorf("making %s: %w", target, err)
		}
		made++
	}
	return dir, made, nil
}
