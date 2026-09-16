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
	// deletionKeepVariable carries the name of the one folder inside the directory to leave
	// standing, on the same terms (FR-578).
	deletionKeepVariable = "SETUP_KEEP_FOLDER"
)

// dirDeletion answers with the arguments for deletionShell and the environment entries it needs to
// wait for the process pid to exit and then delete dir.
//
// With keep empty, dir goes with everything in it. With keep naming a folder inside dir, everything
// in dir goes except that folder, which stays with its contents, so dir stays standing around it
// (FR-578). Only what is directly inside dir is compared with keep: a folder of that name deeper
// down is not the one the user was offered.
//
// A process that has already exited is not waited for, so a delete started late still runs.
func dirDeletion(pid int, dir, keep string) (args, env []string) {
	wait := "Wait-Process -Id " + strconv.Itoa(pid) + " -ErrorAction SilentlyContinue; "
	env = []string{deletionDirVariable + "=" + dir}
	script := wait + "Remove-Item -LiteralPath $env:" + deletionDirVariable +
		" -Recurse -Force -ErrorAction SilentlyContinue"
	if keep != "" {
		script = wait + "Get-ChildItem -LiteralPath $env:" + deletionDirVariable +
			" -Force -ErrorAction SilentlyContinue | Where-Object { $_.Name -ne $env:" +
			deletionKeepVariable + " } | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue"
		env = append(env, deletionKeepVariable+"="+keep)
	}
	return []string{"-NoProfile", "-NonInteractive", "-Command", script}, env
}

// deletionWorkDir is where the delete is started: beside dir rather than wherever setup stands. A
// process holds its working directory, so a delete started inside dir empties it and leaves it
// standing; measured on 2026-09-15.
func deletionWorkDir(dir string) string { return filepath.Dir(dir) }
