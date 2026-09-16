package taskbar

import "time"

// awaitWatcher asks whether the desktop's tray watcher is there until it answers yes or the grace
// period has passed, sleeping between asks (FR-814). Started at sign-in the application is up before
// the panel that hosts the icon, so one ask would find no tray on a desktop about to have one.
func awaitWatcher(present func() bool, every, grace time.Duration, sleep func(time.Duration)) bool {
	for waited := time.Duration(0); ; waited += every {
		if present() {
			return true
		}
		if waited+every > grace {
			return false
		}
		sleep(every)
	}
}
