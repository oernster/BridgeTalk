package reporoot

// The repository root, found by walking up to go.mod, for the tools and tests that work on the tree.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// From the root itself or from any folder beneath it, the root is the nearest folder holding go.mod.
func TestTheRootIsTheNearestFolderHoldingGoMod(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, moduleFile), []byte("module example\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}
	nested := filepath.Join(root, "internal", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("making %s: %v", nested, err)
	}

	for _, from := range []string{root, nested} {
		if got, err := Find(from); err != nil || got != root {
			t.Errorf("Find(%s) = %q, %v; want %q", from, got, err, root)
		}
	}
}

// A walk that reaches the top of the drive without meeting go.mod is refused, naming where it
// started. Whether a real drive holds a go.mod at its top is not this test's to know, so the
// question is asked of a stand-in that answers no.
func TestAWalkThatMeetsNoGoModIsRefused(t *testing.T) {
	t.Parallel()
	from := filepath.Join(t.TempDir(), "somewhere")
	var asked int

	_, err := find(from, func(string) bool { asked++; return false })

	if !errors.Is(err, ErrNoModule) || !strings.Contains(err.Error(), from) {
		t.Errorf("find = %v, want %v naming %s", err, ErrNoModule, from)
	}
	if want := strings.Count(filepath.Clean(from), string(filepath.Separator)) + 1; asked != want {
		t.Errorf("asked about %d folders, want %d: one for each folder up to the top", asked, want)
	}
}
