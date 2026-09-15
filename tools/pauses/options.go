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

// outFlag names the flag giving the file the pauses are written to.
const outFlag = "out"

// errShippedWithOnly means a run over some voices was asked to write a shipped file, which would ship
// a book missing the rest.
var errShippedWithOnly = errors.New("a run with -only would ship a book missing voices: name other files with -out and -endings")

// errOutWithEndingsOnly means a run finding the endings alone was named a pauses file it would never
// write.
var errOutWithEndingsOnly = errors.New("a run with -endings-only writes no pauses: leave out -out")

// options is what a run was asked for: the voices to measure, where to write their pauses and their
// endings and whether the endings are found alone.
type options struct {
	voices  []machinevoice.Voice
	out     string
	endings string
	// endingsOnly reports whether the run finds the endings alone, leaving the pauses file untouched
	// (FR-555).
	endingsOnly bool
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
// default to the shipped pauses and endings, either of which a run with -only refuses where it would
// write it. -endings-only finds the endings alone, so it refuses -out (FR-555).
func parseOptions(args []string, root string, output io.Writer) (options, error) {
	flags := flag.NewFlagSet("pauses", flag.ContinueOnError)
	flags.SetOutput(output)
	shippedPauses, shippedEndings := filepath.Join(root, pausesPath), filepath.Join(root, endingsPath)
	var only voiceIDs
	flags.Var(&only, "only", "measure these voices alone, repeated or comma separated; needs -endings with -out too unless -endings-only")
	out := flags.String(outFlag, shippedPauses, "the file the pauses are written to")
	endings := flags.String("endings", shippedEndings, "the file the endings are written to")
	endingsOnly := flags.Bool("endings-only", false, "find and write the endings alone, leaving the pauses untouched; takes no -out")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	outNamed := false
	flags.Visit(func(set *flag.Flag) { outNamed = outNamed || set.Name == outFlag })
	if *endingsOnly && outNamed {
		return options{}, errOutWithEndingsOnly
	}
	chosen := options{voices: machinevoice.All(), out: *out, endings: *endings, endingsOnly: *endingsOnly}
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
	written := [][2]string{{*endings, shippedEndings}}
	if !*endingsOnly {
		written = [][2]string{{*out, shippedPauses}, {*endings, shippedEndings}}
	}
	for _, each := range written {
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
