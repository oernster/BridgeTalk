//go:build benchmarks && windows

package machinevoice

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oernster/bridge-talk/internal/application/ports"
	"github.com/oernster/bridge-talk/internal/application/services"
	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
	"github.com/oernster/bridge-talk/internal/infrastructure/runlog"
	"github.com/oernster/bridge-talk/internal/infrastructure/speechmodel"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

// bytesPerMB is how many bytes the requirements count as a megabyte.
const bytesPerMB = 1 << 20

const (
	// diskLimit is the most a complete script's made lines for one voice may take (NFR-C-502).
	diskLimit = 60 * bytesPerMB
	// pollInterval is how often the test asks how far making has got.
	pollInterval = 100 * time.Millisecond
)

// measuredVoice is the voice the script is made for: the one the per-line timings in REQUIREMENTS.md
// section 6.1 were measured with.
const measuredVoice = "bf_emma"

// TestMakingACompleteScriptKeepsWithinDisk asks for every cue's lines of the shipped script for one
// voice, then holds what the made lines take on disk to NFR-C-502, logging how long making took. While
// the script lacks lines for some cues it skips, since a part of the script measures nothing.
func TestMakingACompleteScriptKeepsWithinDisk(t *testing.T) {
	dir := modelfilestest.Require(t)
	table, voiced := shipped(t)
	loaded, err := config.LoadScript(table)
	if err != nil {
		t.Fatalf("script.toml: %v", err)
	}
	if missing := loaded.Missing(table); len(missing) > 0 {
		t.Skipf("script.toml holds lines for %d of %d cues; a complete script is measured",
			table.Len()-len(missing), table.Len())
	}
	voice, err := machinevoice.Parse(measuredVoice)
	if err != nil {
		t.Fatalf("Parse(%q): %v", measuredVoice, err)
	}

	store := t.TempDir()
	maker := speechmodel.New(dir)
	defer maker.Close()
	confirmation, _ := table.Confirmation()
	service := services.NewMakingService(
		voiced, shippedPauses(t), confirmation, voicefiles.New(dir), maker, madelines.New(store), runlog.NewLines(os.Stderr),
	)
	defer service.Stop()

	started := time.Now()
	source, err := service.Cast(voice)
	if err != nil {
		t.Fatalf("Cast(%s): %v", measuredVoice, err)
	}
	onCall, ok := source.(ports.CueMaker)
	if !ok {
		t.Fatalf("the audio source %T makes no line on call", source)
	}
	for _, id := range voiced.Cues() {
		onCall.MakeNext(id)
	}
	for service.Progress().Making {
		time.Sleep(pollInterval)
	}
	took := time.Since(started)

	progress := service.Progress()
	t.Logf("made %d of %d lines for %s in %v", progress.Current, progress.Total, measuredVoice, took.Round(time.Millisecond))
	if len(progress.Failed) > 0 || progress.Stopped != nil || progress.Current != progress.Total {
		t.Fatalf("made %d of %d lines; %d failed, stopped by %v", progress.Current, progress.Total, len(progress.Failed), progress.Stopped)
	}

	size := sizeOf(t, store)
	t.Logf("the made lines take %.1f MB", float64(size)/bytesPerMB)
	if size > diskLimit {
		t.Errorf("NFR-C-502: the made lines take %.1f MB, over the %d MB limit", float64(size)/bytesPerMB, diskLimit/bytesPerMB)
	}
}

// sizeOf sums the bytes of every file under dir.
func sizeOf(t *testing.T, dir string) int64 {
	t.Helper()
	var total int64
	err := filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		t.Fatalf("measuring %s: %v", dir, err)
	}
	return total
}
