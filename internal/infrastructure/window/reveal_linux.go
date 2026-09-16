//go:build linux

package window

import "os/exec"

// Reveal opens a folder in the user's own file manager through xdg-open (FR-816). Inside the flatpak
// xdg-open reaches the file manager through the desktop portal, which needs no grant of its own.
func Reveal(dir string) error {
	return openWith(dir, func(name string, args ...string) error {
		return exec.Command(name, args...).Run()
	})
}
