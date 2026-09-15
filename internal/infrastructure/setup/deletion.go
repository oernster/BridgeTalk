package setup

import (
	"path/filepath"
	"strconv"
)

// The install directory holds the running setup program when the Apps list opens it, so the
// directory can only go once setup has closed (FR-805). The process that deletes it is described
// here, apart from the Windows file that starts it, so what it is told can be tested anywhere.

const (
	// deletionShell is the program that waits for setup to close and then deletes. PowerShell
	// is used because it can wait on a process by its id, which cmd cannot.
	deletionShell = "powershell.exe"
	// deletionDirVariable carries the directory to that program as a value in its environment.
	// Nothing is typed into the script, so no character in a path can be read as anything but
	// itself.
	deletionDirVariable = "SETUP_DELETE_DIR"
)

// dirDeletion answers with the arguments for deletionShell and the one environment entry it
// needs to wait for the process pid to exit and then delete dir with everything in it.
//
// A process that has already exited is not waited for, so a delete started late still runs.
func dirDeletion(pid int, dir string) (args []string, env string) {
	script := "Wait-Process -Id " + strconv.Itoa(pid) + " -ErrorAction SilentlyContinue; " +
		"Remove-Item -LiteralPath $env:" + deletionDirVariable + " -Recurse -Force -ErrorAction SilentlyContinue"
	return []string{"-NoProfile", "-NonInteractive", "-Command", script}, deletionDirVariable + "=" + dir
}

// deletionWorkDir is where the delete is started: beside dir rather than wherever setup stands. A
// process holds its working directory, so a delete started inside dir empties it and leaves it
// standing; measured on 2026-09-15.
func deletionWorkDir(dir string) string { return filepath.Dir(dir) }
