package appdata

// The product's local data folder. The rule for each platform is exercised through dir's
// parameters, so each is tested on every platform rather than only the one a test runs on.

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/product"
)

// On Windows it sits in the local data folder under the product's own name; elsewhere in
// XDG_DATA_HOME under the same name.
func TestTheDataFolderBelongsToTheProduct(t *testing.T) {
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
		{windowsOS, filepath.Join(env["LOCALAPPDATA"], product.Slug)},
		{"linux", filepath.Join(env["XDG_DATA_HOME"], product.Slug)},
	}
	for _, each := range cases {
		got, err := dir(each.goos, getenv, unasked)
		if err != nil || got != each.want {
			t.Errorf("%s: got %q, %v; want %q", each.goos, got, err, each.want)
		}
	}
}

// With no XDG_DATA_HOME the XDG rule falls back to ~/.local/share.
func TestWithNoDataHomeTheFolderFallsBackUnderHome(t *testing.T) {
	home := filepath.Join("home", "cmdr")
	got, err := dir("linux", func(string) string { return "" },
		func() (string, error) { return home, nil })

	want := filepath.Join(home, ".local", "share", product.Slug)
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

// Nothing is invented where the platform gives nothing to build on.
func TestTheFolderIsNotInventedWithoutItsSources(t *testing.T) {
	empty := func(string) string { return "" }

	if _, err := dir(windowsOS, empty, nil); !errors.Is(err, errNoLocalAppData) {
		t.Errorf("no LOCALAPPDATA: got %v", err)
	}
	homeless := errors.New("no home directory")
	if _, err := dir("linux", empty, func() (string, error) { return "", homeless }); !errors.Is(err, homeless) {
		t.Errorf("no home: got %v, want the reason carried", err)
	}
}

// Dir reads this machine's environment, touching nothing on disk.
func TestDirReadsThisMachinesEnvironment(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	t.Setenv("XDG_DATA_HOME", base)

	got, err := Dir()

	if want := filepath.Join(base, product.Slug); err != nil || got != want {
		t.Errorf("got %q, %v; want %q", got, err, want)
	}
}
