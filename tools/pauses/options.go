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

// endingsPath is where the shipped endings are written, from the repository root (FR-555).
const endingsPath = "internal/infrastructure/config/endings.toml"

// onlySeparator separates voice ids given together to -only.
const onlySeparator = ","

// errShippedWithOnly means a run over some voices was asked to write a shipped file, which would ship
// a book missing the rest.
var errShippedWithOnly = errors.New("a run with -only would ship a book missing voices: name other files with -out and -endings")

// options is what a run was asked for: the voices to measure and where to write their pauses and
// their endings.
type options struct {
	voices  []machinevoice.Voice
	out     string
	endings string
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
// measured; with it, the voices named, each once in the order they are offered. -out and -endings
// default to the shipped pauses and endings, either of which a run with -only refuses.
func parseOptions(args []string, root string, output io.Writer) (options, error) {
	flags := flag.NewFlagSet("pauses", flag.ContinueOnError)
	flags.SetOutput(output)
	shippedPauses, shippedEndings := filepath.Join(root, pausesPath), filepath.Join(root, endingsPath)
	var only voiceIDs
	flags.Var(&only, "only", "measure these voices alone, repeated or comma separated; needs -out and -endings")
	out := flags.String("out", shippedPauses, "the file the pauses are written to")
	endings := flags.String("endings", shippedEndings, "the file the endings are written to")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	chosen := options{voices: machinevoice.All(), out: *out, endings: *endings}
	if len(only) == 0 {
		return chosen, nil
	}
	picked := make(map[string]bool, len(only))
	for _, id := range only {
		voice, err := machinevoice.Parse(strings.TrimSpace(id))
		if err != nil {
			return options{}, err
		}
		picked[voice.ID()] = true
	}
	for _, each := range [][2]string{{*out, shippedPauses}, {*endings, shippedEndings}} {
		if sameFile(each[0], each[1]) {
			return options{}, fmt.Errorf("%w: %s", errShippedWithOnly, each[1])
		}
	}
	chosen.voices = slices.DeleteFunc(chosen.voices, func(voice machinevoice.Voice) bool { return !picked[voice.ID()] })
	return chosen, nil
}

// sameFile reports whether two paths name the same file, each resolved from the working directory.
// Letter case is not told apart, as Windows does not; a path that cannot be resolved counts as the
// same, so the refusal it guards errs toward refusing.
func sameFile(first, second string) bool {
	firstAbs, firstErr := filepath.Abs(first)
	secondAbs, secondErr := filepath.Abs(second)
	return firstErr != nil || secondErr != nil || strings.EqualFold(firstAbs, secondAbs)
}
