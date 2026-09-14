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
}

// -only takes voices repeated or comma separated, each once, in the order they are offered.
func TestOnlyTakesVoicesRepeatedOrCommaSeparatedInTheOrderTheyAreOffered(t *testing.T) {
	out := filepath.Join(t.TempDir(), "pauses.toml")
	chosen, err := parseOptions([]string{"-only", "bm_george,bf_emma", "-only", "bf_emma", "-out", out}, testRoot(t), io.Discard)
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if got, want := idsOf(chosen.voices), []string{"bf_emma", "bm_george"}; !slices.Equal(got, want) {
		t.Errorf("voices = %v, want %v", got, want)
	}
	if chosen.out != out {
		t.Errorf("out = %q, want %q", chosen.out, out)
	}
}

// A run with -only would ship a book missing voices, so it refuses the shipped file however it is named.
func TestOnlyRefusesToWriteTheShippedFile(t *testing.T) {
	root := testRoot(t)
	working, _ := os.Getwd()
	relative, err := filepath.Rel(working, filepath.Join(root, pausesPath))
	if err != nil {
		t.Fatalf("Rel: %v", err)
	}
	for _, args := range [][]string{{"-only", "bf_emma"}, {"-only", "bf_emma", "-out", relative}} {
		if _, err := parseOptions(args, root, io.Discard); !errors.Is(err, errShippedWithOnly) {
			t.Errorf("parseOptions(%q) = %v, want errShippedWithOnly", args, err)
		}
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
