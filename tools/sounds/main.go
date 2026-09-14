// Command sounds makes the speech sounds of every line in script.toml in each accent with misaki,
// run in the tool's own venv, then saves them as sounds.toml beside the script (FR-532, FR-534).
//
// Run it from the repository root whenever script.toml changes:
//
//	go run ./tools/sounds
//
// The venv is made once, as tools/sounds/requirements.txt says.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/infrastructure/config"
)

// soundsPath is where the saved speech sounds are written, from the repository root.
const soundsPath = "internal/infrastructure/config/sounds.toml"

// soundsFileMode is how the saved speech sounds are written: as any source file, readable by
// everyone and writable by the owner.
const soundsFileMode = 0o644

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sounds:", err)
		os.Exit(1)
	}
}

// run makes every line's speech sounds and saves them.
func run() error {
	if _, err := os.Stat(filepath.Dir(soundsPath)); err != nil {
		return fmt.Errorf("run it from the repository root: %w", err)
	}
	table, err := config.LoadCueTable("")
	if err != nil {
		return err
	}
	loaded, err := config.LoadScript(table)
	if err != nil {
		return err
	}
	saved, err := makeSounds(loaded, python{})
	if err != nil {
		return err
	}
	written, err := config.EncodeSounds(saved)
	if err != nil {
		return err
	}
	if err := os.WriteFile(soundsPath, written, soundsFileMode); err != nil {
		return err
	}
	fmt.Printf("saved the speech sounds of every line in script.toml to %s\n", soundsPath)
	return nil
}
