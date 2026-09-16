package window

// Opening a folder on Linux (FR-816), run on every platform through the runner openWith takes.

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The folder is handed to xdg-open alone and nothing is said when it opens.
func TestAFolderIsHandedToXdgOpen(t *testing.T) {
	dir := filepath.Join("home", "pilot", "Voices", "Alice")
	var ran []string
	err := openWith(dir, func(name string, args ...string) error {
		ran = append([]string{name}, args...)
		return nil
	})
	if err != nil {
		t.Fatalf("opening: %v", err)
	}
	if want := []string{"xdg-open", dir}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %v, want %v", ran, want)
	}
}

// An opener that fails says why, naming xdg-open and never the folder, which the caller names.
func TestAnOpenerThatFailsSaysWhyWithoutNamingTheFolderAgain(t *testing.T) {
	dir := filepath.Join("home", "pilot", "Voices", "Alice")
	err := openWith(dir, func(string, ...string) error { return errors.New("exit status 4") })
	if err == nil {
		t.Fatal("a failed opener was reported as opening the folder")
	}
	if !strings.Contains(err.Error(), "xdg-open") || !strings.Contains(err.Error(), "exit status 4") {
		t.Errorf("reason = %q, want it to name xdg-open and what went wrong", err)
	}
	if strings.Contains(err.Error(), dir) {
		t.Errorf("reason = %q names the folder, which the caller already names", err)
	}
}
