// Command pauses finds, for every machine voice, where the pause before a final commander goes in each
// line the script joins, then writes the pauses as pauses.toml beside sounds.toml (FR-551, FR-552).
//
// Each line is made with the model in models/, which go run ./tools/models fills; the break in it is
// found with Praat by pauses.py, run in the tool's own venv. Run it from anywhere inside the repository
// after the saved speech sounds or the model files change:
//
//	go run ./tools/pauses
//
// A full run makes every joined line for all 28 voices. For a quick check over some of them:
//
//	go run ./tools/pauses -only bf_emma,bm_george -out <path>
//
// -only names the voices, repeated or comma separated. -out names the file the pauses are written to;
// by default the shipped pauses.toml, which a run with -only refuses, since it would ship pauses
// missing voices. The venv is made once, as tools/pauses/requirements.txt says.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/oernster/bridge-talk/internal/infrastructure/config"
	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/reporoot"
	"github.com/oernster/bridge-talk/internal/infrastructure/speechmodel"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

// fileMode is how the tool writes files: as any source file, readable by everyone and writable by the
// owner.
const fileMode = 0o644

// workPattern names the temporary folder a run writes its lines in for the finder.
const workPattern = "bridge-talk-pauses-*"

// fillModels says how to fill models/ where it does not match the list.
const fillModels = "the model files are not all in models/ as the list gives them: run go run ./tools/models, then run this again"

func main() {
	if err := start(); err != nil {
		fmt.Fprintln(os.Stderr, "pauses:", err)
		os.Exit(1)
	}
}

// start reads the flags, the voiced script and the model files, then finds every voice's pauses and
// writes them, until it ends or is interrupted.
func start() error {
	working, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("finding the working directory: %w", err)
	}
	root, err := reporoot.Find(working)
	if err != nil {
		return err
	}
	chosen, err := parseOptions(os.Args[1:], root, os.Stderr)
	if err != nil {
		return err
	}
	table, err := config.LoadCueTable("")
	if err != nil {
		return err
	}
	voiced, err := config.LoadVoicedScript(table)
	if err != nil {
		return err
	}
	listed, err := modelfiles.Listed()
	if err != nil {
		return err
	}
	dir, err := modelfiles.Dir(working)
	if err != nil {
		return err
	}
	if err := modelfiles.Verify(dir, listed); err != nil {
		return fmt.Errorf("%s: %w", fillModels, err)
	}
	work, err := os.MkdirTemp("", workPattern)
	if err != nil {
		return fmt.Errorf("making a temporary folder for the lines: %w", err)
	}
	defer os.RemoveAll(work)
	model := speechmodel.New(dir)
	defer model.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	run := finding{files: voicefiles.New(dir), maker: model, finder: python{root: root}, work: work, out: os.Stdout}
	book, err := run.book(ctx, voiced, chosen.voices)
	if err != nil {
		return err
	}
	written, err := config.EncodePauses(book)
	if err != nil {
		return err
	}
	if err := os.WriteFile(chosen.out, written, fileMode); err != nil {
		return err
	}
	fmt.Printf("wrote the pauses to %s\n", chosen.out)
	return nil
}
