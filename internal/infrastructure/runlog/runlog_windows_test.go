//go:build windows

package runlog

// FR-715 on Windows, where a windowed program started from a shortcut has no error output: the log
// takes all of it, a fatal error's first line included. The data folder comes from LOCALAPPDATA.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/product"
)

// The child has an error output of its own, so it is sent to the log directly: finding that a run
// has none is not something a test binary can be started without.
func TestWhereTheRunHasNoErrorOutputEverythingWrittenToItIsInTheLog(t *testing.T) {
	t.Parallel()
	logged, errorOutput := runChild(t, fatalAct)

	for _, want := range []string{startLine(started), plantedWarning, fatalHeadline} {
		if !strings.Contains(logged, want) {
			t.Errorf("the log lacks %q: %q", want, logged)
		}
		if strings.Contains(errorOutput, strings.TrimSpace(want)) && want != startLine(started) {
			t.Errorf("the error output still took %q: %q", want, errorOutput)
		}
	}
}

func TestTheLogSitsInTheProductsDataFolder(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)

	got, err := Path()

	if want := filepath.Join(base, product.Slug, FileName); err != nil || got != want {
		t.Errorf("got %q, %v; want %q", got, err, want)
	}
}

func TestWithNoDataFolderThereIsNoLog(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")

	if got, err := Path(); err == nil {
		t.Errorf("got %q with no data folder, want a refusal", got)
	}
}
