//go:build linux

package setup

// The sign-in entry on Linux: the autostart file autostart.go names and writes (FR-815).

import (
	"fmt"
	"os"
)

// SetLaunchOnBoot writes the autostart entry starting the program running; unticked, it removes it.
func SetLaunchOnBoot(running string, enabled bool) error {
	path, err := linuxAutostartPath()
	if err != nil {
		return err
	}
	return applyAutostart(path, autostartEntry(os.Getenv, running), enabled)
}

// IsLaunchOnBoot reports whether the autostart entry is there.
func IsLaunchOnBoot() bool { return HasLaunchOnBootEntry() }

// HasLaunchOnBootEntry reports whether the autostart entry is there.
func HasLaunchOnBootEntry() bool {
	path, err := linuxAutostartPath()
	return err == nil && autostartPresent(path)
}

// linuxAutostartPath answers where the entry goes for this user.
func linuxAutostartPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding the home directory: %w", err)
	}
	return autostartPath(os.Getenv, home), nil
}
