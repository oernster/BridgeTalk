package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
)

// pausesPath is where the shipped pauses are written, from the repository root.
const pausesPath = "internal/infrastructure/config/pauses.toml"

// onlySeparator separates voice ids given together to -only.
const onlySeparator = ","

// errShippedWithOnly means a run over some voices was asked to write the shipped pauses, which would
// ship a book missing the rest.
var errShippedWithOnly = errors.New("a run with -only would ship pauses missing voices: name another file with -out")

// options is what a run was asked for: the voices to find pauses for and where to write them.
type options struct {
	voices []machinevoice.Voice
	out    string
}

// voiceIDs collects the ids given to -only, repeated or comma separated.
type voiceIDs []string

// String answers the ids given so far, comma separated.
func (ids *voiceIDs) String() string { return strings.Join(*ids, onlySeparator) }

// Set adds the ids in one -only.
func (ids *voiceIDs) Set(value string) error {
	*ids = append(*ids, strings.Split(value, onlySeparator)...)
	return nil
}

// parseOptions reads the flags for the repository at root. Without -only every voice offered is
// found; with it, the voices named, each once in the order they are offered. -out defaults to the
// shipped pauses, which a run with -only refuses.
func parseOptions(args []string, root string, output io.Writer) (options, error) {
	flags := flag.NewFlagSet("pauses", flag.ContinueOnError)
	flags.SetOutput(output)
	shipped := filepath.Join(root, pausesPath)
	var only voiceIDs
	flags.Var(&only, "only", "find the pauses of these voices alone, repeated or comma separated; needs -out")
	out := flags.String("out", shipped, "the file the pauses are written to")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	voices := machinevoice.All()
	if len(only) == 0 {
		return options{voices: voices, out: *out}, nil
	}
	chosen := make(map[string]bool, len(only))
	for _, id := range only {
		voice, err := machinevoice.Parse(strings.TrimSpace(id))
		if err != nil {
			return options{}, err
		}
		chosen[voice.ID()] = true
	}
	if sameFile(*out, shipped) {
		return options{}, fmt.Errorf("%w: %s", errShippedWithOnly, shipped)
	}
	voices = slices.DeleteFunc(voices, func(voice machinevoice.Voice) bool { return !chosen[voice.ID()] })
	return options{voices: voices, out: *out}, nil
}

// sameFile reports whether two paths name the same file, each resolved from the working directory.
// Letter case is not told apart, as Windows does not; a path that cannot be resolved counts as the
// same, so the refusal it guards errs toward refusing.
func sameFile(first, second string) bool {
	firstAbs, firstErr := filepath.Abs(first)
	secondAbs, secondErr := filepath.Abs(second)
	return firstErr != nil || secondErr != nil || strings.EqualFold(firstAbs, secondAbs)
}
