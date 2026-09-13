//go:build !windows

package taskbar

// Tray is the non-Windows stand-in for the notification-area icon.
//
// Linux and macOS have no equivalent this application can rely on without pulling in
// a desktop toolkit, so the tray is a no-op there rather than a dependency. Every
// method is safe to call; the command channel simply never yields anything.
type Tray struct {
	commands chan Command
	options  Options
}

// New builds a tray that does nothing.
func New(options Options) *Tray {
	return &Tray{commands: make(chan Command), options: options}
}

// Commands yields nothing on a platform with no tray.
func (t *Tray) Commands() <-chan Command { return t.commands }

// SetMuted records nothing.
func (t *Tray) SetMuted(bool) {}

// SetActiveVoice records nothing.
func (t *Tray) SetActiveVoice(string) {}

// Start succeeds without showing anything.
func (t *Tray) Start() error { return nil }

// Stop closes the command channel so a caller ranging over it finishes.
func (t *Tray) Stop() { close(t.commands) }
