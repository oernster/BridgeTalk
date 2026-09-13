package library

// The default recordings directory (FR-227). The rule for each platform is exercised
// through defaultRoot's parameters; the directory itself is made only under a temporary
// data folder, so no test writes to the machine it runs on.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/product"
)

// On Windows it sits in the local data folder under the product's own name; elsewhere
// in XDG_DATA_HOME under the same name.
func TestTheDefaultRecordingsDirectoryBelongsToTheProduct(t *testing.T) {
	env := map[string]string{
		"LOCALAPPDATA":  filepath.FromSlash("C:/Users/cmdr/AppData/Local"),
		"XDG_DATA_HOME": filepath.FromSlash("/home/cmdr/.data"),
	}
	getenv := func(key string) string { return env[key] }
	unasked := func() (string, error) {
		t.Fatal("the home directory was asked for with a data folder already named")
		return "", nil
	}

	cases := []struct {
		goos string
		want string
	}{
		{windowsOS, filepath.Join(env["LOCALAPPDATA"], product.Slug, recordingsFolder)},
		{"linux", filepath.Join(env["XDG_DATA_HOME"], product.Slug, recordingsFolder)},
	}
	for _, each := range cases {
		got, err := defaultRoot(each.goos, getenv, unasked)
		if err != nil || got != each.want {
			t.Errorf("%s: got %q, %v; want %q", each.goos, got, err, each.want)
		}
	}
}

// With no XDG_DATA_HOME the XDG rule falls back to ~/.local/share.
func TestWithNoDataHomeTheDefaultFallsBackUnderHome(t *testing.T) {
	home := filepath.Join("home", "cmdr")
	got, err := defaultRoot("linux", func(string) string { return "" },
		func() (string, error) { return home, nil })

	want := filepath.Join(home, ".local", "share", product.Slug, recordingsFolder)
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

// Nothing is invented where the platform gives nothing to build on.
func TestTheDefaultIsNotInventedWithoutItsSources(t *testing.T) {
	empty := func(string) string { return "" }

	if _, err := defaultRoot(windowsOS, empty, nil); !errors.Is(err, errNoLocalAppData) {
		t.Errorf("no LOCALAPPDATA: got %v", err)
	}
	homeless := errors.New("no home directory")
	if _, err := defaultRoot("linux", empty, func() (string, error) { return "", homeless }); !errors.Is(err, homeless) {
		t.Errorf("no home: got %v, want the reason carried", err)
	}
}

// It is made where missing, so a folder dialog has somewhere to open; asking again
// finds it rather than failing.
func TestTheDefaultRecordingsDirectoryIsMadeWhereMissing(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	t.Setenv("XDG_DATA_HOME", base)

	for range 2 {
		dir, err := DefaultRoot()
		if err != nil {
			t.Fatalf("making the default: %v", err)
		}
		if dir != filepath.Join(base, product.Slug, recordingsFolder) {
			t.Fatalf("made %q, want it under the data folder", dir)
		}
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("no directory at %q: %v", dir, err)
		}
	}
}

// A default that cannot be worked out or cannot be made is reported, not guessed at.
func TestADefaultThatCannotBeMadeIsReported(t *testing.T) {
	file := filepath.Join(t.TempDir(), "plain")
	writeTake(t, file)
	t.Setenv("LOCALAPPDATA", file)
	t.Setenv("XDG_DATA_HOME", file)
	if _, err := DefaultRoot(); err == nil {
		t.Error("a default was made inside a file")
	}

	for _, key := range []string{"LOCALAPPDATA", "XDG_DATA_HOME", "HOME", "USERPROFILE"} {
		t.Setenv(key, "")
	}
	if _, err := DefaultRoot(); err == nil {
		t.Error("a default was invented with nowhere to put it")
	}
}
