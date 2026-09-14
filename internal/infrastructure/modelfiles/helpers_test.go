package modelfiles_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// write puts a file in place, failing the test where it cannot.
func write(t *testing.T, dir, name string, body []byte) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// holds fails the test unless dir holds exactly the files named, each with its body.
func holds(t *testing.T, dir string, want map[string][]byte) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var names, wanted []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	for name, body := range want {
		wanted = append(wanted, name)
		path := filepath.Join(dir, name)
		if got, err := os.ReadFile(path); err != nil || string(got) != string(body) {
			t.Errorf("%s holds %q, %v; want %q", path, got, err, body)
		}
	}
	slices.Sort(wanted)
	if !slices.Equal(names, wanted) {
		t.Errorf("%s holds %v, want %v", dir, names, wanted)
	}
}
