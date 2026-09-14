// Package setup holds the per-user install logic behind the bespoke setup program:
// the payload extraction and the paths, which are portable and unit tested, plus the
// registry, shortcut and process side effects in the Windows files beside this one.
//
// Everything is per-user. Nothing here needs administrator rights, so the whole flow
// runs without an elevation prompt.
package setup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oernster/bridge-talk/internal/product"
	"github.com/oernster/bridge-talk/internal/refusal"
)

const (
	// AppName is the product name shown to the user, used for the Start Menu entry
	// and for the Apps list.
	AppName = product.Name
	// InstallFolder names the install directory and the registry key. It carries no
	// spaces so a path never needs quoting for the sake of readability.
	InstallFolder = product.Slug
	// ExeName is the installed application executable.
	ExeName = product.Slug + ".exe"
	// Publisher is recorded in the uninstall registry entry.
	Publisher = "Oliver Ernster"

	installSubdir = "Programs"
	dirPerm       = 0o755
)

// InstallDir returns the per-user install directory,
// %LOCALAPPDATA%\Programs\BridgeTalk. Installing under LOCALAPPDATA is what
// keeps the whole flow free of an administrator prompt.
func InstallDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", fmt.Errorf("LOCALAPPDATA is not set")
	}
	return filepath.Join(base, installSubdir, InstallFolder), nil
}

// StateDir returns the folder WebView2 creates for the application's window state
// and theme choice. The application sets no WebviewUserDataPath, so WebView2 falls
// back to %APPDATA% joined with the executable's own file name, the .exe suffix
// included. Uninstall offers to forget it along with the settings file; the recordings
// directory is never touched.
func StateDir() (string, error) {
	base := os.Getenv("APPDATA")
	if base == "" {
		return "", fmt.Errorf("APPDATA is not set")
	}
	return filepath.Join(base, ExeName), nil
}

// ExtractZip extracts a zip archive into dest, creating directories as needed and
// refusing any entry whose path would escape dest.
//
// The archive arrives as a string because the setup program embeds its payload as one:
// an embedded byte slice is charged to the process as private memory from the moment it
// starts, which for a payload carrying the model files is over 300 MB (FR-524).
func ExtractZip(data string, dest string) error {
	reader, err := zip.NewReader(strings.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("opening the payload: %w", err)
	}
	if err := os.MkdirAll(dest, dirPerm); err != nil {
		return fmt.Errorf("creating %s: %w", dest, refusal.Reason(err))
	}
	for _, file := range reader.File {
		if err := extractEntry(file, dest); err != nil {
			return err
		}
	}
	return nil
}

// extractEntry writes one archive entry, rejecting a name that climbs out of dest.
//
// Every refusal names its path once in words of its own, with only the system's reason
// after it (FR-237).
func extractEntry(file *zip.File, dest string) error {
	target := filepath.Join(dest, file.Name)
	fence := filepath.Clean(dest) + string(os.PathSeparator)
	if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), fence) {
		return fmt.Errorf("unsafe path in payload: %s", file.Name)
	}
	folder := filepath.Dir(target)
	if file.FileInfo().IsDir() {
		folder = target
	}
	if err := os.MkdirAll(folder, dirPerm); err != nil {
		return fmt.Errorf("creating %s: %w", folder, refusal.Reason(err))
	}
	if file.FileInfo().IsDir() {
		return nil
	}
	source, err := file.Open()
	if err != nil {
		return fmt.Errorf("opening entry %s: %w", file.Name, err)
	}
	defer source.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, dirPerm)
	if err != nil {
		return fmt.Errorf("creating %s: %w", target, refusal.Reason(err))
	}
	defer out.Close()
	if _, err := io.Copy(out, source); err != nil {
		return fmt.Errorf("writing %s: %w", target, refusal.Reason(err))
	}
	return nil
}

// DirSizeKB returns the total size of a directory tree in kilobytes, which is what
// the uninstall entry's EstimatedSize value wants.
func DirSizeKB(dir string) (uint32, error) {
	var total int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("measuring %s: %w", dir, refusal.Reason(err))
	}
	return uint32(total / 1024), nil
}

// RemoveTree deletes a directory tree. It is used for the saved window state, which
// the uninstall screen offers to keep.
func RemoveTree(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing %s: %w", dir, refusal.Reason(err))
	}
	return nil
}

// CopyFile copies one file, used to leave a copy of the setup program inside the
// install directory so the Apps list has an uninstaller to call.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, refusal.Reason(err))
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, dirPerm)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, refusal.Reason(err))
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("writing %s: %w", dst, refusal.Reason(err))
	}
	return nil
}

// HiddenFlag asks the application to start in the notification area with no window.
//
// The login entry carries it because an application that opens a window every time the
// machine is signed into is not waiting quietly, which is what the setting offers.
// Neighbouring entries in the same key do the same thing under their own spellings:
// Steam has -silent, Discord has --start-inactive.
const HiddenFlag = "-hidden"

// UninstallFlag opens setup on the removal screen. The Apps list passes it back to the
// uninstaller copy, so the registry entry and the setup program read it from here.
const UninstallFlag = "-uninstall"

// uninstallValues is the text the Apps list entry holds, keyed by registry value name.
//
// It is kept apart from the registry write that stores it, which runs on Windows only,
// so what is written can be tested on any machine.
func uninstallValues(info UninstallInfo) map[string]string {
	quoted := quotedPath(info.UninstallExe)
	return map[string]string{
		"DisplayName":     AppName,
		"DisplayVersion":  info.Version,
		"InstallLocation": info.InstallDir,
		"UninstallString": quoted + " " + UninstallFlag,
		"ModifyPath":      quoted,
		"DisplayIcon":     info.IconPath,
		"Publisher":       Publisher,
	}
}

// runValue is what the login entry holds: the path in quotes, then the hidden flag.
//
// It exists because writing it with %q shipped a broken entry. %q is Go's quoting,
// which escapes the separators inside the string, so a Windows path reached the
// registry doubled and Windows spent every sign-in looking for a place that does not
// exist. The entry is written by the setup program and by Settings, so both were
// wrong; a quoted path is what every other entry in that key looks like.
func runValue(exePath string) string {
	return quotedPath(exePath) + " " + HiddenFlag
}

// quotedPath wraps a path in plain double quotes, the way Windows reads a command line.
//
// Every registry value that names a program to run goes through here: the login entry,
// the uninstall command and the modify command. Go's %q is not a substitute, because it
// escapes the separators inside the string and the path then names nowhere.
func quotedPath(path string) string {
	return `"` + path + `"`
}

// runTarget reads a login entry back as the path it starts, so the entry can be
// checked against the file it names.
//
// The quoted form is read first because that is what is written: everything up to the
// closing quote is the path, whatever follows it is arguments. An unquoted value is
// taken whole rather than split on the first space, since an older entry was written
// without either quotes or arguments and a path with a space in it is ordinary.
func runTarget(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, `"`) {
		if end := strings.Index(trimmed[1:], `"`); end >= 0 {
			return trimmed[1 : end+1]
		}
	}
	return strings.Trim(trimmed, `"`)
}
