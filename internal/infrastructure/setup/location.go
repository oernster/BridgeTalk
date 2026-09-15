package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oernster/bridge-talk/internal/refusal"
)

// Where the application is installed (FR-809).
//
// Setup offers a folder under the local application data, which needs no administrator
// rights; the Install screen may pick another. Whatever is picked, the install goes into
// a folder named for the product inside it. Uninstall deletes the install folder with
// everything in it (FR-805), so setup never installs straight into a folder that may already
// hold somebody's files.

// probePattern names the empty file written into a folder, then removed, to learn whether
// this account can write there.
const probePattern = InstallFolder + "-probe-*"

// DefaultInstallDir returns the folder setup offers, %LOCALAPPDATA%\Programs\BridgeTalk.
// Installing under LOCALAPPDATA is what keeps the whole flow free of an administrator prompt.
func DefaultInstallDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", fmt.Errorf("LOCALAPPDATA is not set")
	}
	return filepath.Join(base, installSubdir, InstallFolder), nil
}

// InstallDir returns the folder the application is installed in: the location its Apps list
// entry records where there is one, else the folder setup offers. An install made before a
// location could be chosen recorded the offered folder, so it reads back as it always did.
func InstallDir() (string, error) { return installDirFrom(recordedInstallDir()) }

// installDirFrom answers with the recorded location where there is one, else the offered
// folder, so the choice between them is tested apart from the registry read.
func installDirFrom(recorded string) (string, error) {
	if recorded != "" {
		return recorded, nil
	}
	return DefaultInstallDir()
}

// InstallDirWithin returns the folder an install into picked writes: a folder of its own
// inside it. A picked folder already named for the product is taken as it stands, so picking
// the folder shown does not nest a second one inside it.
func InstallDirWithin(picked string) string {
	if strings.EqualFold(filepath.Base(picked), InstallFolder) {
		return picked
	}
	return filepath.Join(picked, InstallFolder)
}

// CheckInstallDir answers why dir cannot take an install; nil when it can. Setup asks before
// the Install screen shows a picked folder and again before it writes, since the page holds
// only what it was told.
func CheckInstallDir(dir string) error { return checkInstallDir(dir, probeWrite) }

// checkInstallDir is CheckInstallDir with the write probe passed in, so a test can play a
// folder this account may not write to, which no temporary folder is.
func checkInstallDir(dir string, probe func(folder string) error) error {
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("%s is not a full path", dir)
	}
	if !strings.EqualFold(filepath.Base(dir), InstallFolder) {
		return fmt.Errorf("%s is not a folder named %s", dir, InstallFolder)
	}
	folder, err := NearestFolder(dir)
	if err != nil {
		return err
	}
	if folder == dir {
		if err := checkHeld(dir); err != nil {
			return err
		}
	}
	if err := probe(folder); err != nil {
		return fmt.Errorf("%s cannot be written to from this account: %w", folder, refusal.Reason(err))
	}
	return nil
}

// NearestFolder walks up from dir to the first folder that exists, which is where an install
// would first write and where the folder picker opens. A file standing on the way is refused,
// since no folder can be made beneath it.
func NearestFolder(dir string) (string, error) {
	for path := dir; ; path = filepath.Dir(path) {
		info, err := os.Stat(path)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%s is a file, so no folder can be made inside it", path)
			}
			return path, nil
		}
		if filepath.Dir(path) == path {
			return "", fmt.Errorf("no folder on the way to %s exists", dir)
		}
	}
}

// checkHeld refuses a folder that already holds something other than the application, since
// uninstalling would delete it along with the application. An empty folder is taken, as is
// one the application is already in.
func checkHeld(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("%s cannot be read: %w", dir, refusal.Reason(err))
	}
	if len(entries) == 0 {
		return nil
	}
	if _, err := os.Stat(filepath.Join(dir, ExeName)); err == nil {
		return nil
	}
	return fmt.Errorf("%s already holds files that are not %s's, which uninstalling would delete", dir, AppName)
}

// probeWrite learns whether this account can write into folder by writing an empty file there
// and removing it again.
func probeWrite(folder string) error {
	file, err := os.CreateTemp(folder, probePattern)
	if err != nil {
		return err
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}
