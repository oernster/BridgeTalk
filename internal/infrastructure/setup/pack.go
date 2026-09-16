package setup

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/oernster/bridge-talk/internal/refusal"
)

// Payload names what the setup program carries (FR-524): every file under App, placed at the
// root of the install directory, then each file named in Models, taken from ModelsDir and placed
// in the folder Folder beside the application, where it reads them (FR-539).
type Payload struct {
	// App is the folder holding the built application.
	App string
	// ModelsDir is the folder the model files are taken from.
	ModelsDir string
	// Folder names the folder the model files are placed in, beside the application.
	Folder string
	// Models names the model files carried.
	Models []string
}

// Pack writes the payload to out as a zip archive. An application folder holding no application
// executable is refused before anything is written; a file that cannot be read or packed is
// refused naming it.
//
// Each file is copied into the archive as it is read, so a payload carrying the model is never
// held in memory whole.
func Pack(out io.Writer, payload Payload) error {
	if _, err := os.Stat(filepath.Join(payload.App, ExeName)); err != nil {
		return fmt.Errorf("%s holds no %s: %w", payload.App, ExeName, refusal.Reason(err))
	}
	archive := zip.NewWriter(out)
	err := filepath.WalkDir(payload.App, func(from string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("reading %s: %w", from, refusal.Reason(err))
		}
		// The plugins folder is never carried, whatever the built application's folder holds. The
		// application looks for plugins beside itself, so a build tried out with a plugin in place
		// would otherwise ship that plugin; an update would then write it over the user's own
		// (FR-577). Setup makes the folder empty instead (FR-576).
		if entry.IsDir() && from == PluginsDir(payload.App) {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		// The walk only reaches paths under App, which Rel always relates to it.
		name, _ := filepath.Rel(payload.App, from)
		return add(archive, filepath.ToSlash(name), from)
	})
	if err != nil {
		return err
	}
	for _, name := range payload.Models {
		if err := add(archive, path.Join(payload.Folder, name), filepath.Join(payload.ModelsDir, name)); err != nil {
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("finishing the payload: %w", err)
	}
	return nil
}

// add copies the file at from into the archive under name, which is written with forward
// slashes as the zip format has it.
func add(archive *zip.Writer, name, from string) error {
	source, err := os.Open(from)
	if err != nil {
		return fmt.Errorf("reading %s: %w", from, refusal.Reason(err))
	}
	defer source.Close()
	member, err := archive.Create(name)
	if err != nil {
		return fmt.Errorf("packing %s: %w", from, err)
	}
	if _, err := io.Copy(member, source); err != nil {
		return fmt.Errorf("packing %s: %w", from, refusal.Reason(err))
	}
	return nil
}
