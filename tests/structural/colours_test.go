package structural

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// tokenFile is the one place a colour value may be written. Everything else refers
// to a token, so a change of palette is one edit and light and dark cannot drift.
const tokenFile = "theme.css"

// frontendSource is where the front end's own files live. node_modules and the build
// output are other people's code and are not scanned.
var frontendSource = filepath.Join("frontend", "src")

// colourLiteral matches the ways a colour is written in CSS or in a style attribute:
// either a hex value or one of the rgb, rgba, hsl, hsla functions.
var colourLiteral = regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b|\b(rgba?|hsla?)\s*\(`)

// styleExtensions are the files that can carry a colour.
var styleExtensions = map[string]bool{".css": true, ".ts": true, ".tsx": true}

// TestColoursOnlyInTokens keeps every colour value in the theme file.
//
// Proved by planting a hex value in a component and reading the exit code. Without
// it a one-off colour compiles, renders and looks right in whichever theme it was
// written for, then reads as a mistake in the other one; nothing else in the build
// would have said a word.
func TestColoursOnlyInTokens(t *testing.T) {
	root := repoRoot(t)
	source := filepath.Join(root, frontendSource)
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil //nolint:nilerr // an unreadable subtree is skipped, not fatal
		}
		if !styleExtensions[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		if filepath.Base(path) == tokenFile {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Errorf("reading %s: %v", path, readErr)
			return nil
		}
		for number, line := range strings.Split(string(raw), "\n") {
			if found := colourLiteral.FindString(line); found != "" {
				relative, _ := filepath.Rel(root, path)
				t.Errorf(
					"%s:%d writes the colour %q: every colour belongs in %s as a token",
					filepath.ToSlash(relative), number+1, found, tokenFile,
				)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", source, err)
	}
}
