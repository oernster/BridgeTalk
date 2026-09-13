//go:build windows

package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	uninstallKeyPath = `Software\Microsoft\Windows\CurrentVersion\Uninstall\` + InstallFolder
	runKeyPath       = `Software\Microsoft\Windows\CurrentVersion\Run`
	themeKeyPath     = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	themeValueName   = "AppsUseLightTheme"
	runValueName     = InstallFolder
	shortcutName     = AppName + ".lnk"
)

// UninstallInfo carries the values written to the HKCU uninstall registry entry,
// which is what puts the application in Settings and in the Apps list.
type UninstallInfo struct {
	Version      string
	InstallDir   string
	UninstallExe string
	IconPath     string
	EstimatedKB  uint32
}

// hidden keeps a shelled-out child process from flashing a console window.
func hidden() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

// WriteUninstallEntry registers the application under the current user's uninstall
// list. NoModify and NoRepair are both zero, so Windows offers Modify and Repair
// alongside Uninstall and each one reopens this setup program.
func WriteUninstallEntry(info UninstallInfo) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, uninstallKeyPath, registry.WRITE)
	if err != nil {
		return fmt.Errorf("create uninstall key: %w", err)
	}
	defer key.Close()

	for name, value := range uninstallValues(info) {
		if err := key.SetStringValue(name, value); err != nil {
			return fmt.Errorf("set %q: %w", name, err)
		}
	}
	numbers := map[string]uint32{"NoModify": 0, "NoRepair": 0, "EstimatedSize": info.EstimatedKB}
	for name, value := range numbers {
		if err := key.SetDWordValue(name, value); err != nil {
			return fmt.Errorf("set %q: %w", name, err)
		}
	}
	return nil
}

// RemoveUninstallEntry deletes the uninstall registry entry.
func RemoveUninstallEntry() error {
	if err := registry.DeleteKey(registry.CURRENT_USER, uninstallKeyPath); err != nil {
		return fmt.Errorf("delete uninstall key: %w", err)
	}
	return nil
}

// InstalledVersion returns the installed version and whether the application is
// installed at all, read from the uninstall registry entry.
func InstalledVersion() (string, bool) {
	key, err := registry.OpenKey(registry.CURRENT_USER, uninstallKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", false
	}
	defer key.Close()
	value, _, err := key.GetStringValue("DisplayVersion")
	if err != nil {
		return "", false
	}
	return value, true
}

// SetLaunchOnBoot adds or removes the current user's Run entry.
func SetLaunchOnBoot(exePath string, enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open run key: %w", err)
	}
	defer key.Close()
	if !enabled {
		_ = key.DeleteValue(runValueName)
		return nil
	}
	if err := key.SetStringValue(runValueName, runValue(exePath)); err != nil {
		return fmt.Errorf("set run value: %w", err)
	}
	return nil
}

// IsLaunchOnBoot reports whether the login entry is present AND names a real file.
//
// Presence alone was the earlier test and it was not enough. An entry written with the
// old doubled path is present, so the box read as on while nothing launched, which is
// the worst of both: the setting says yes and Windows disagrees silently. Checking the
// file means such an entry reads as off; turning it on then rewrites it correctly.
func IsLaunchOnBoot() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetStringValue(runValueName)
	if err != nil {
		return false
	}
	_, err = os.Stat(runTarget(value))
	return err == nil
}

// SystemPrefersDark reports whether Windows is set to a dark app theme, which is
// what the setup window opens in unless the user toggles it. A missing or
// unreadable value reads as light, matching a fresh Windows install.
func SystemPrefersDark() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, themeKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetIntegerValue(themeValueName)
	if err != nil {
		return false
	}
	return value == 0
}

// createShortcut writes a .lnk through the Windows Script Host, which avoids
// handling COM directly for one call.
func createShortcut(linkPath, target, workDir string) error {
	script := fmt.Sprintf(
		`$s=(New-Object -ComObject WScript.Shell).CreateShortcut(%q);`+
			`$s.TargetPath=%q;$s.IconLocation=%q;$s.WorkingDirectory=%q;$s.Save()`,
		linkPath, target, target, workDir)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = hidden()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating the shortcut %s: %w: %s", linkPath, err, string(out))
	}
	return nil
}

// Shortcuts says which shortcuts the user asked for on the install screen.
type Shortcuts struct {
	StartMenu bool
	Desktop   bool
}

// ApplyShortcuts creates the shortcuts that are wanted and removes the ones that are
// not, so unticking a box on a reinstall takes the shortcut away rather than leaving
// a stale one behind. Each location is best effort: a failure in one never stops the
// other, because a missing shortcut is not worth failing an install over.
func ApplyShortcuts(exePath, workDir string, want Shortcuts) {
	place := func(dir string, wanted bool) {
		link := filepath.Join(dir, shortcutName)
		if !wanted {
			_ = os.Remove(link)
			return
		}
		_ = createShortcut(link, exePath, workDir)
	}
	if dir, err := StartMenuProgramsDir(); err == nil {
		place(dir, want.StartMenu)
	}
	if dir, err := DesktopDir(); err == nil {
		place(dir, want.Desktop)
	}
}

// CurrentShortcuts reports which shortcuts exist, so the manage and reinstall
// screens open with the boxes already reflecting the machine.
func CurrentShortcuts() Shortcuts {
	present := func(dir string, err error) bool {
		if err != nil {
			return false
		}
		_, statErr := os.Stat(filepath.Join(dir, shortcutName))
		return statErr == nil
	}
	start, startErr := StartMenuProgramsDir()
	desktop, desktopErr := DesktopDir()
	return Shortcuts{
		StartMenu: present(start, startErr),
		Desktop:   present(desktop, desktopErr),
	}
}

// RemoveShortcuts deletes both shortcuts.
func RemoveShortcuts() {
	ApplyShortcuts("", "", Shortcuts{})
}

// DesktopDir returns the current user's Desktop directory.
func DesktopDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, "Desktop"), nil
}

// StartMenuProgramsDir returns the current user's Start Menu Programs directory.
func StartMenuProgramsDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("APPDATA is not set")
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs"), nil
}
