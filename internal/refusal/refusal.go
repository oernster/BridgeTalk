// Package refusal words a refusal over a file or folder for the person reading it (FR-237).
//
// A file-system error from the standard library already carries the path it was about
// and the name of the system call that failed: "GetFileAttributesEx C:\Recordings: The
// system cannot find the file specified." A caller that names the path in its own words
// and wraps that error shows the path twice, with the call name between the two. Reason
// keeps only the part a reader can act on, so the caller's own words name the path once.
//
// It sits under internal rather than in a layer for the reason product does. Every
// infrastructure package and the setup program read it; none of them may depend on
// another for it.
package refusal

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// systemCalls are the names the standard library gives the calls behind a file-system
// error, each written straight before the path it was about.
var systemCalls = []string{
	"CreateFile", "CreateProcess", "FindFirstFile", "GetFileAttributesEx", "RemoveAll",
	"chmod", "fork/exec", "lstat", "mkdir", "open", "readdir", "remove", "rename", "stat",
	"unlinkat",
}

// Reason returns what went wrong without the path or the system call. Hand it the error
// straight from the call: one already wrapped in words of its own comes back unchanged,
// so those words are never lost, as does an error that names no path at all.
//
// The reason is the error the system gave, so errors.Is asks the same questions of it
// that it asked of the original.
func Reason(err error) error {
	switch failure := err.(type) {
	case *fs.PathError:
		return failure.Err
	case *os.LinkError:
		return failure.Err
	case *os.SyscallError:
		return failure.Err
	}
	return err
}

// PassedOver words one thing the application decided not to use, so a plugin, a voice inside a
// plugin that otherwise loaded and a part of a take that would not open all read the same way in
// the run log (FR-567, FR-574).
//
// It is here rather than beside any one of them for the reason Reason is here: three packages
// need these words and none of them may depend on another. A voice that gave no reason arrives
// with one filled in by the plugin loader, which is the one home for those words.
func PassedOver(what, why string) string {
	return "note: " + what + " was passed over: " + why
}

// Check lists what a refusal over path gets wrong under FR-237: nothing refused at all,
// the path named other than once, its separators doubled or a system call written before
// a path. An empty list is a refusal that reads as it should.
//
// The system call is looked for before the start of any path on the same drive or root,
// not only before this one: a folder that cannot be made under a plain file is refused by
// the system over the plain file, so the reader would otherwise be shown the parent again.
//
// Every package that refuses over a file or folder holds its refusals to this in its
// tests, so the rule is written down in one place.
func Check(err error, path string) []string {
	if err == nil {
		return []string{"nothing was refused"}
	}
	said := err.Error()
	var problems []string
	if count := strings.Count(said, path); count != 1 {
		problems = append(problems, fmt.Sprintf("names %s %d times in %q", path, count, said))
	}
	if doubled := strings.ReplaceAll(path, `\`, `\\`); doubled != path && strings.Contains(said, doubled) {
		problems = append(problems, fmt.Sprintf("doubles the separators of %s in %q", path, said))
	}
	root := path
	if cut := strings.IndexAny(path, `\/`); cut >= 0 {
		root = path[:cut+1]
	}
	for _, call := range systemCalls {
		if strings.Contains(said, call+" "+root) {
			problems = append(problems, fmt.Sprintf("names the system call %s in %q", call, said))
		}
	}
	return problems
}
