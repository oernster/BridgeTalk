// Command models fills models/ at the repository root with every file a machine voice is made from,
// as the list in internal/infrastructure/modelfiles gives them (FR-536). With -check it downloads
// nothing and fails naming each file missing or different (FR-538).
//
// Run it from anywhere inside the repository:
//
//	go run ./tools/models
//	go run ./tools/models -check
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
)

func main() {
	if err := start(); err != nil {
		fmt.Fprintln(os.Stderr, "models:", err)
		os.Exit(1)
	}
}

// start reads the list and finds the folder, then runs the tool over them until it ends or is
// interrupted.
func start() error {
	files, err := modelfiles.Listed()
	if err != nil {
		return err
	}
	working, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("finding the working directory: %w", err)
	}
	dir, err := modelfiles.Dir(working)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return run(ctx, os.Args[1:], dir, &http.Client{}, files, os.Stdout)
}

// run fills dir from the list; where args ask for -check it checks the folder instead.
func run(ctx context.Context, args []string, dir string, client *http.Client, files []modelfiles.File, out io.Writer) error {
	flags := flag.NewFlagSet("models", flag.ContinueOnError)
	flags.SetOutput(out)
	check := flags.Bool("check", false, "check the folder against the list, downloading nothing")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*check {
		if err := modelfiles.Fetch(ctx, client, dir, files, out); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s holds every listed file\n", dir)
		return nil
	}
	if err := modelfiles.Verify(dir, files); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s matches the list\n", dir)
	return nil
}
