// Command payload packs the setup program's payload (FR-543): every file of the built application,
// then every model file setup installs, placed in the folder the application reads them from beside
// itself (FR-524, FR-539). The model files are checked against the list first, downloading nothing;
// where they do not match, the archive is left as it was.
//
// build.ps1 runs it from the repository root:
//
//	go run ./tools/payload -app build/bin -out installer/payload.zip
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/setup"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
	"github.com/oernster/bridge-talk/internal/infrastructure/wholefile"
)

const (
	// packingSuffix follows the archive's name while it is packed, so a packing that fails leaves the
	// archive that was there before.
	packingSuffix = ".packing"
	// filePerm is how the archive is written: readable by everyone, writable by the owner.
	filePerm = 0o644
)

// errNoFolders means the tool was not told where the application is or where the archive goes.
var errNoFolders = errors.New("both -app and -out are needed")

func main() {
	if err := start(); err != nil {
		fmt.Fprintln(os.Stderr, "payload:", err)
		os.Exit(1)
	}
}

// start reads the list and finds the models folder, then runs the tool over them.
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
	return run(os.Args[1:], dir, files, os.Stdout)
}

// run checks modelsDir against files, then packs the application folder args name, with every model
// file setup installs, into the archive args name.
func run(args []string, modelsDir string, files []modelfiles.File, out io.Writer) error {
	flags := flag.NewFlagSet("payload", flag.ContinueOnError)
	flags.SetOutput(out)
	app := flags.String("app", "", "the folder holding the built application")
	archive := flags.String("out", "", "the archive to write")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *app == "" || *archive == "" {
		return errNoFolders
	}
	if err := modelfiles.Verify(modelsDir, files); err != nil {
		return err
	}
	payload := setup.Payload{App: *app, ModelsDir: modelsDir, Folder: voicefiles.Folder, Models: modelfiles.Installed(files)}
	err := wholefile.Write(*archive, packingSuffix, filePerm, func(packed io.Writer) error {
		return setup.Pack(packed, payload)
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s holds the application and %d model files\n", *archive, len(payload.Models))
	return nil
}
