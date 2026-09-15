package main

// FR-551: a full run finds every voice's pauses for the shipped pauses.toml; -only takes some voices
// for a quick check with -out naming where they go, never the shipped file.

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/infrastructure/reporoot"
)

// testRoot finds the repository the test runs in.
func testRoot(t *testing.T) string {
	t.Helper()
	working, err := os.Getwd()
	if err != nil {
		t.Fatalf("finding the working directory: %v", err)
	}
	root, err := reporoot.Find(working)
	if err != nil {
		t.Fatalf("finding the repository root: %v", err)
	}
	return root
}

// idsOf answers the ids of voices in order.
func idsOf(voices []machinevoice.Voice) []string {
	ids := make([]string, 0, len(voices))
	for _, voice := range voices {
		ids = append(ids, voice.ID())
	}
	return ids
}

// With no flags every voice offered is found and the shipped file is written.
func TestWithoutFlagsEveryVoiceIsFoundForTheShippedFile(t *testing.T) {
	root := testRoot(t)
	chosen, err := parseOptions(nil, root, io.Discard)
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if got, want := idsOf(chosen.voices), idsOf(machinevoice.All()); !slices.Equal(got, want) {
		t.Errorf("voices = %v, want every voice %v", got, want)
	}
	if want := filepath.Join(root, pausesPath); chosen.out != want {
		t.Errorf("out = %q, want %q", chosen.out, want)
	}
	if want := filepath.Join(root, endingsPath); chosen.endings != want {
		t.Errorf("endings = %q, want %q (FR-555)", chosen.endings, want)
	}
}

// -only takes voices repeated or comma separated, each once, in the order they are offered; -out and
// -endings name where the pauses and the endings go.
func TestOnlyTakesVoicesRepeatedOrCommaSeparatedInTheOrderTheyAreOffered(t *testing.T) {
	dir := t.TempDir()
	out, endings := filepath.Join(dir, "pauses.toml"), filepath.Join(dir, "endings.toml")
	chosen, err := parseOptions([]string{"-only", "bm_george,bf_emma", "-only", "bf_emma", "-out", out, "-endings", endings}, testRoot(t), io.Discard)
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if got, want := idsOf(chosen.voices), []string{"bf_emma", "bm_george"}; !slices.Equal(got, want) {
		t.Errorf("voices = %v, want %v", got, want)
	}
	if chosen.out != out || chosen.endings != endings {
		t.Errorf("out = %q endings = %q, want %q and %q", chosen.out, chosen.endings, out, endings)
	}
}

// A run with -only would ship a book missing voices, so it refuses either shipped file however it is
// named.
func TestOnlyRefusesToWriteTheShippedFile(t *testing.T) {
	root := testRoot(t)
	working, _ := os.Getwd()
	dir := t.TempDir()
	for _, shipped := range []string{pausesPath, endingsPath} {
		relative, err := filepath.Rel(working, filepath.Join(root, shipped))
		if err != nil {
			t.Fatalf("Rel: %v", err)
		}
		for _, args := range [][]string{
			{"-only", "bf_emma", "-endings", filepath.Join(dir, "endings.toml")},
			{"-only", "bf_emma", "-out", filepath.Join(dir, "pauses.toml")},
			{"-only", "bf_emma", "-out", relative, "-endings", relative},
		} {
			if _, err := parseOptions(args, root, io.Discard); !errors.Is(err, errShippedWithOnly) {
				t.Errorf("parseOptions(%q) = %v, want errShippedWithOnly", args, err)
			}
		}
	}
}

// FR-555: -endings-only asks for the endings alone, written to the shipped endings; a run without it
// asks for both.
func TestEndingsOnlyAsksForTheEndingsAlone(t *testing.T) {
	root := testRoot(t)
	full, fullErr := parseOptions(nil, root, io.Discard)
	alone, aloneErr := parseOptions([]string{"-endings-only"}, root, io.Discard)
	if fullErr != nil || aloneErr != nil {
		t.Fatalf("parseOptions: %v, %v", fullErr, aloneErr)
	}
	if full.endingsOnly || !alone.endingsOnly || alone.endings != filepath.Join(root, endingsPath) {
		t.Errorf("full run endings only %v; -endings-only %v writing %q, want false then true writing the shipped endings", full.endingsOnly, alone.endingsOnly, alone.endings)
	}
}

// A run with -endings-only writes no pauses, so a pauses file named beside it is refused rather than
// ignored.
func TestEndingsOnlyRefusesAPausesFile(t *testing.T) {
	_, err := parseOptions([]string{"-endings-only", "-out", filepath.Join(t.TempDir(), "pauses.toml")}, testRoot(t), io.Discard)
	if !errors.Is(err, errOutWithEndingsOnly) {
		t.Errorf("got %v, want errOutWithEndingsOnly", err)
	}
}

// With -only, -endings-only needs -endings and no -out; the shipped endings are still refused.
func TestEndingsOnlyWithOnlyNeedsOnlyAnEndingsFile(t *testing.T) {
	root := testRoot(t)
	endings := filepath.Join(t.TempDir(), "endings.toml")
	chosen, err := parseOptions([]string{"-endings-only", "-only", "bf_emma", "-endings", endings}, root, io.Discard)
	if err != nil || !chosen.endingsOnly || chosen.endings != endings || !slices.Equal(idsOf(chosen.voices), []string{"bf_emma"}) {
		t.Errorf("parseOptions = %+v, %v; want bf_emma's endings alone written to %q", chosen, err, endings)
	}
	if _, err := parseOptions([]string{"-endings-only", "-only", "bf_emma"}, root, io.Discard); !errors.Is(err, errShippedWithOnly) {
		t.Errorf("got %v, want errShippedWithOnly for the shipped endings", err)
	}
}

// A voice that is not offered is refused, naming it.
func TestAnUnknownVoiceIsRefused(t *testing.T) {
	_, err := parseOptions([]string{"-only", "xf_nobody", "-out", filepath.Join(t.TempDir(), "p.toml")}, testRoot(t), io.Discard)
	if !errors.Is(err, machinevoice.ErrUnknownVoice) {
		t.Errorf("got %v, want ErrUnknownVoice", err)
	}
}

// A flag the tool does not have is refused.
func TestAFlagTheToolDoesNotHaveIsRefused(t *testing.T) {
	if _, err := parseOptions([]string{"-every"}, testRoot(t), io.Discard); err == nil {
		t.Error("an unknown flag was accepted")
	}
}
