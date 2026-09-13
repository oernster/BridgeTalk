//go:build windows

package window

import "os/exec"

// explorer is the Windows file manager, which opens the folder it is given.
const explorer = "explorer.exe"

// Reveal opens a folder in File Explorer.
//
// The process is started rather than waited on, so the window stays responsive while
// Explorer comes up; its handle is released at once, since nothing here reads its exit.
func Reveal(dir string) error {
	command := exec.Command(explorer, dir)
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}
