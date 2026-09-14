package main

// The model files tool: filling the folder (FR-536), then checking it without downloading (FR-538).

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles/modelfilestest"
)

// Checking an empty folder fails naming the missing file and asks for nothing; filling it asks once;
// checking again passes.
func TestTheToolFillsTheFolderThenChecksIt(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := t.TempDir()
	files := []modelfiles.File{server.Entry("bf_alice.bin", []byte("alice"))}
	tool := func(args ...string) error {
		return run(context.Background(), args, dir, server.Client(), files, io.Discard)
	}

	if err := tool("-check"); err == nil || !strings.Contains(err.Error(), "bf_alice.bin") {
		t.Errorf("checking an empty folder = %v, want a failure naming bf_alice.bin", err)
	}
	if asked := server.Asked(); len(asked) != 0 {
		t.Errorf("checking asked for %v", asked)
	}
	if err := tool(); err != nil {
		t.Fatalf("filling the folder: %v", err)
	}
	if err := tool("-check"); err != nil {
		t.Errorf("checking the filled folder: %v", err)
	}
	if asked := server.Asked(); len(asked) != 1 {
		t.Errorf("asked for %v, want one download", asked)
	}
}

// A file present but different fails the check as well as one missing.
func TestTheCheckFailsOnAFileThatDiffers(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	dir := t.TempDir()
	files := []modelfiles.File{server.Entry("bf_alice.bin", []byte("alice"))}
	if err := run(context.Background(), nil, dir, server.Client(), files, io.Discard); err != nil {
		t.Fatalf("filling the folder: %v", err)
	}
	files[0].SHA256 = modelfilestest.Digest([]byte("ALICE"))

	err := run(context.Background(), []string{"-check"}, dir, server.Client(), files, io.Discard)

	if err == nil || !strings.Contains(err.Error(), "bf_alice.bin") {
		t.Errorf("checking a different file = %v, want a failure naming bf_alice.bin", err)
	}
}

// A fill that cannot download a file fails naming it.
func TestAFillThatCannotDownloadFails(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)
	file := server.Entry("bf_alice.bin", []byte("alice"))
	file.Address += "-gone"

	err := run(context.Background(), nil, t.TempDir(), server.Client(), []modelfiles.File{file}, io.Discard)

	if err == nil || !strings.Contains(err.Error(), "bf_alice.bin") {
		t.Errorf("filling from a missing address = %v, want a failure naming bf_alice.bin", err)
	}
}

// A flag the tool does not know is refused rather than taken as a fill.
func TestAnUnknownFlagIsRefused(t *testing.T) {
	t.Parallel()
	server := modelfilestest.Serve(t)

	err := run(context.Background(), []string{"-fetch"}, t.TempDir(), server.Client(), nil, io.Discard)

	if err == nil {
		t.Error("an unknown flag was accepted")
	}
}
