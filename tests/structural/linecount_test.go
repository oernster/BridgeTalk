package structural

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLineCountCountsTheLinesAnEditorShows holds the size rule to the number an editor
// shows beside the last line. A newline ends a line rather than starting another, so a
// file ending in one is not a line longer than the same file without it.
func TestLineCountCountsTheLinesAnEditorShows(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		{"an empty file", "", 0},
		{"one line ending in a newline", "one\n", 1},
		{"one line with no newline at the end", "one", 1},
		{"two lines ending in a newline", "one\ntwo\n", 2},
		{"two lines with no newline at the end", "one\ntwo", 2},
		{"two lines ending in carriage returns and newlines", "one\r\ntwo\r\n", 2},
		{"a blank line before the end", "one\n\n", 2},
	}
	dir := t.TempDir()
	for index, each := range cases {
		path := filepath.Join(dir, "case"+string(rune('a'+index))+".txt")
		if err := os.WriteFile(path, []byte(each.content), 0o600); err != nil {
			t.Fatalf("writing %s: %v", each.name, err)
		}
		if got := lineCount(t, path); got != each.want {
			t.Errorf("%s: lineCount gave %d, want %d", each.name, got, each.want)
		}
	}
}
