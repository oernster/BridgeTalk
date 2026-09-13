package structural

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// styleManifest lists the style files in the order they are read. themeDir holds the
// parts it lists, one per region of the window.
var (
	styleManifest = filepath.Join("frontend", "src", "styles.css")
	themeDir      = filepath.Join("frontend", "src", "theme")
)

// importLine matches one entry of the manifest and captures the path it names.
var importLine = regexp.MustCompile(`@import\s+'([^']+)'`)

// TestEveryStylePartIsRead keeps the manifest and the parts in step.
//
// Proved by planting an unlisted file in the theme directory and reading the exit
// code. A part that nothing imports is the failure this split invites: it still
// parses, it still reads correctly, it is still edited by whoever meant to change the
// window and none of its rules ever reach a screen. Nothing else in the build,
// neither the type check nor the bundler, would say a word about it.
func TestEveryStylePartIsRead(t *testing.T) {
	root := repoRoot(t)

	raw, err := os.ReadFile(filepath.Join(root, styleManifest))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(styleManifest), err)
	}

	listed := map[string]bool{}
	for _, match := range importLine.FindAllStringSubmatch(string(raw), -1) {
		listed[path(match[1])] = true
	}

	entries, err := os.ReadDir(filepath.Join(root, themeDir))
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.ToSlash(themeDir), err)
	}

	var unread []string
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".css" {
			continue
		}
		if !listed["theme/"+entry.Name()] {
			unread = append(unread, entry.Name())
		}
	}
	sort.Strings(unread)
	for _, name := range unread {
		t.Errorf(
			"theme/%s is never read: add it to %s in the position its rules belong",
			name, filepath.ToSlash(styleManifest),
		)
	}
}

// path reduces a manifest entry to the form the directory scan produces, so a leading
// './' does not read as a different file from the one it names.
func path(entry string) string {
	return strings.TrimPrefix(entry, "./")
}
