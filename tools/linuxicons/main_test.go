package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/iconfile"
	"github.com/oernster/bridge-talk/internal/product"
)

// committed is the repository's icon, read from where the tool is run.
var committed = filepath.Join("..", "..", iconPath)

// Every picture in the committed icon is installed under the hicolor theme at its own size, named
// for the application id. Each written path is named.
func TestEveryPictureIsInstalledAtItsSize(t *testing.T) {
	prefix := t.TempDir()
	var out bytes.Buffer
	if err := run([]string{"-prefix", prefix, "-icon", committed}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	ico, err := os.ReadFile(committed)
	if err != nil {
		t.Fatalf("reading the icon: %v", err)
	}
	sides, _ := iconfile.Sides(ico)
	for _, side := range sides {
		size := fmt.Sprintf("%dx%d", side, side)
		path := filepath.Join(prefix, "share", "icons", "hicolor", size, "apps", product.AppID+".png")
		want, _ := iconfile.Frame(ico, side)
		if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, want) {
			t.Errorf("%d: %s holds %d bytes (%v), want the icon's own %d", side, path, len(got), err, len(want))
		}
		if !strings.Contains(out.String(), path) {
			t.Errorf("the output does not name %s", path)
		}
	}
}

// Nothing is installed without a prefix, from an icon that is not there or from one that is not an
// icon; each says why.
func TestAnInstallThatCannotGoAheadSaysWhy(t *testing.T) {
	var out bytes.Buffer
	notIcon := filepath.Join(t.TempDir(), "not.ico")
	if err := os.WriteFile(notIcon, []byte("text"), filePerm); err != nil {
		t.Fatalf("writing: %v", err)
	}
	for name, args := range map[string][]string{
		"no prefix":       {"-icon", committed},
		"no icon":         {"-prefix", t.TempDir(), "-icon", filepath.Join(t.TempDir(), "gone.ico")},
		"not an icon":     {"-prefix", t.TempDir(), "-icon", notIcon},
		"an unknown flag": {"-nonsense"},
	} {
		if err := run(args, &out); err == nil {
			t.Errorf("%s: the install went ahead", name)
		}
	}
}
